package usecases

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/berops/claudie/internal/kubectl"
	"github.com/berops/claudie/internal/loggerutils"
	"github.com/berops/claudie/internal/nodepools"
	"github.com/berops/claudie/proto/pb"

	"gopkg.in/yaml.v3"
)

// Number of retries to perform to try to unmarshal the kubeadm config map
// before giving up.
const ReadKubeadmConfigRetries = 3

func (u *Usecases) PatchKubeadmConfigMap(ctx context.Context, request *pb.PatchKubeadmConfigMapRequest) (*pb.PatchKubeadmConfigMapResponse, error) {
	logger := loggerutils.WithClusterName(request.DesiredCluster.ClusterInfo.Id())
	logger.Info().Msgf("Patching kubeadm ConfigMap")

	certSANs := []string{request.LbEndpoint}
	if request.LbEndpoint == "" {
		certSANs = certSANs[:len(certSANs)-1]
		for n := range nodepools.Control(request.DesiredCluster.ClusterInfo.NodePools) {
			for _, n := range n.Nodes {
				certSANs = append(certSANs, n.Public)
			}
		}
	}

	// Kubeadm uses this config map when joining new nodes, we need to update it with correct certSANs
	// after api endpoint change.
	// https://github.com/berops/claudie/issues/1597

	k := kubectl.Kubectl{
		Kubeconfig:        request.DesiredCluster.Kubeconfig,
		MaxKubectlRetries: 3,
	}

	var err error
	var configMap []byte
	var rawKubeadmConfigMap map[string]any

	for i := range ReadKubeadmConfigRetries {
		if i > 0 {
			wait := time.Duration(150+rand.IntN(300)) * time.Millisecond
			logger.Warn().Msgf("reading kubeadm-config failed err: %v, retrying again in %s ms [%v/%v]",
				err,
				wait,
				i+1,
				ReadKubeadmConfigRetries,
			)
			time.Sleep(wait)
		}

		configMap, err = k.KubectlGet("cm kubeadm-config", "-oyaml", "-n kube-system")
		if err != nil || len(configMap) == 0 {
			continue
		}
		if err = yaml.Unmarshal(configMap, &rawKubeadmConfigMap); err != nil {
			continue
		}
		break
	}

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve kubeadm-config map after %v retries: %w", ReadKubeadmConfigRetries, err)
	}

	if len(configMap) == 0 {
		logger.Warn().Msgf("kubeadm-config config map was not found, skip patching kubeadm-config map")
		return &pb.PatchKubeadmConfigMapResponse{}, nil
	}

	data, ok := rawKubeadmConfigMap["data"].(map[string]any)
	if !ok {
		logger.Warn().Msgf("Expected 'data' field to be present in the kubeadm config map but was not")
		return nil, fmt.Errorf("expected 'data' field to present, but was missing inside the kubeadm config map")
	}

	config, ok := data["ClusterConfiguration"].(string)
	if !ok {
		logger.Warn().Msgf("Expected 'ClusterConfiguration' field to be present in the kubeadm config map but was not")
		return nil, fmt.Errorf("expected 'ClusterConfiguration' field to present, but was missing inside the kubeadm config map")
	}

	var rawKubeadmConfig map[string]any
	if err := yaml.Unmarshal([]byte(config), &rawKubeadmConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal 'ClusterConfiguration' inside kubeadm-config config map: %w", err)
	}

	apiServer, ok := rawKubeadmConfig["apiServer"].(map[string]any)
	if !ok {
		logger.Warn().Msgf("Expected 'apiServer' field to be present in the 'ClusterConfiguration' for kubeadm, but was not")
		return nil, fmt.Errorf("expected 'apiServer' field to present in the 'ClusterConfiguration' for kubeadm, but was missing: %w", err)
	}

	apiServer["certSANs"] = certSANs

	b, err := yaml.Marshal(rawKubeadmConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated 'ClusterConfiguration' for kubeadm: %w", err)
	}

	data["ClusterConfiguration"] = string(b)

	b, err = yaml.Marshal(rawKubeadmConfigMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated kubeadm-config config map: %w", err)
	}

	if err := k.KubectlApplyString(string(b), "-n kube-system"); err != nil {
		return nil, fmt.Errorf("failed to patch kubeadm-config config map")
	}

	logger.Info().Msgf("Kubeadm-config Config Map patched successfully")
	return &pb.PatchKubeadmConfigMapResponse{}, nil
}
