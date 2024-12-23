package usecases

import (
	"context"
	"fmt"

	comm "github.com/berops/claudie/internal/command"
	"github.com/berops/claudie/internal/envs"
	"github.com/berops/claudie/internal/kubectl"
	"github.com/berops/claudie/internal/loggerutils"
	"github.com/berops/claudie/proto/pb"
)

// DeleteKubeconfig deletes the K8s secret (in the management cluster) containing kubeconfig
// for the given K8s cluster.
func (u *Usecases) DeleteKubeconfig(ctx context.Context, request *pb.DeleteKubeconfigRequest) (*pb.DeleteKubeconfigResponse, error) {
	namespace := envs.Namespace
	if namespace == "" {
		// If kuber deployed locally, return.
		return &pb.DeleteKubeconfigResponse{}, nil
	}
	clusterID := request.Cluster.ClusterInfo.Id()
	logger := loggerutils.WithClusterName(clusterID)
	var err error
	// Log success/error message.
	defer func() {
		if err != nil {
			logger.Warn().Msgf("Failed to remove kubeconfig, secret most likely already removed : %v", err)
		} else {
			logger.Info().Msgf("Deleted kubeconfig secret")
		}
	}()

	logger.Info().Msgf("Deleting kubeconfig secret")
	kc := kubectl.Kubectl{MaxKubectlRetries: 3}
	kc.Stdout = comm.GetStdOut(clusterID)
	kc.Stderr = comm.GetStdErr(clusterID)

	// Save error and return as errors are ignored here.
	err = kc.KubectlDeleteResource("secret", fmt.Sprintf("%s-kubeconfig", clusterID), "-n", namespace)
	return &pb.DeleteKubeconfigResponse{}, nil
}
