package usecases

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/berops/claudie/internal/fileutils"
	"github.com/berops/claudie/internal/hash"
	"github.com/berops/claudie/internal/loggerutils"
	"github.com/berops/claudie/proto/pb"
	"github.com/berops/claudie/services/kuber/server/domain/utils/autoscaler"
)

// DestroyClusterAutoscaler removes deployment of Cluster Autoscaler from the management cluster for given k8s cluster.
func (u *Usecases) DestroyClusterAutoscaler(ctx context.Context, request *pb.DestroyClusterAutoscalerRequest) (*pb.DestroyClusterAutoscalerResponse, error) {
	logger := loggerutils.WithClusterName(request.Cluster.ClusterInfo.Id())

	var err error
	// Log success/error message.
	defer func() {
		if err != nil {
			logger.Err(err).Msgf("Error while destroying cluster autoscaler")
		} else {
			logger.Info().Msgf("Cluster autoscaler successfully destroyed")
		}
	}()

	// Create output dir
	tempClusterID := fmt.Sprintf("%s-%s", request.Cluster.ClusterInfo.Name, hash.Create(5))
	clusterDir := filepath.Join(outputDir, tempClusterID)
	if err = fileutils.CreateDirectory(clusterDir); err != nil {
		return nil, fmt.Errorf("error while creating directory %s : %w", clusterDir, err)
	}

	// Destroy cluster autoscaler.
	logger.Info().Msgf("Destroying Cluster Autoscaler deployment")
	autoscalerManager := autoscaler.NewAutoscalerManager(request.ProjectName, request.Cluster, clusterDir)
	if err := autoscalerManager.DestroyClusterAutoscaler(); err != nil {
		logger.Debug().Msgf("Ignoring Destroy Autoscaler error: %v", err.Error())
	}

	return &pb.DestroyClusterAutoscalerResponse{}, nil
}
