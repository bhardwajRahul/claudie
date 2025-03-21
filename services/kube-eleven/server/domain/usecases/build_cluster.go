package usecases

import (
	"fmt"

	"github.com/berops/claudie/internal/loggerutils"
	"github.com/berops/claudie/proto/pb"
	kube_eleven "github.com/berops/claudie/services/kube-eleven/server/domain/utils/kube-eleven"
)

// BuildCluster builds all cluster defined in the desired state
func (u *Usecases) BuildCluster(req *pb.BuildClusterRequest) (*pb.BuildClusterResponse, error) {
	logger := loggerutils.WithProjectAndCluster(req.ProjectName, req.Desired.ClusterInfo.Id())

	logger.Info().Msgf("Building kubernetes cluster")

	k := kube_eleven.KubeEleven{
		K8sCluster:           req.Desired,
		LoadBalancerEndpoint: req.LoadBalancerEndpoint,
		SpawnProcessLimit:    u.SpawnProcessLimit,
	}

	if err := k.BuildCluster(); err != nil {
		logger.Error().Msgf("Error while building a cluster: %s", err)
		return nil, fmt.Errorf("error while building cluster %s for project %s : %w", req.Desired.ClusterInfo.Name, req.ProjectName, err)
	}

	logger.Info().Msgf("Kubernetes cluster was successfully build")
	return &pb.BuildClusterResponse{Desired: req.Desired}, nil
}
