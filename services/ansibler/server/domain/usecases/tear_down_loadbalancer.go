package usecases

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/berops/claudie/internal/hash"
	"github.com/berops/claudie/proto/pb"
	"github.com/berops/claudie/proto/pb/spec"
	"github.com/berops/claudie/services/ansibler/server/utils"
	"github.com/rs/zerolog/log"

	"golang.org/x/sync/semaphore"
)

// TeardownLoadBalancers correctly destroys loadbalancers by selecting the new ApiServer endpoint.
func (u *Usecases) TeardownLoadBalancers(ctx context.Context, request *pb.TeardownLBRequest) (*pb.TeardownLBResponse, error) {
	// If no LB clusters were deleted from the manifest, then just return.
	if len(request.DeletedLbs) == 0 {
		return &pb.TeardownLBResponse{
			PreviousAPIEndpoint: "",
			Desired:             request.Desired,
			DesiredLbs:          request.DesiredLbs,
			DeletedLbs:          request.DeletedLbs,
		}, nil
	}

	logger := log.With().Str("project", request.ProjectName).Str("cluster", request.Desired.ClusterInfo.Name).Logger()
	logger.Info().Msgf("Tearing down the loadbalancers")

	var isApiServerTypeDesiredLBClusterPresent bool
	for _, lbCluster := range request.DesiredLbs {
		if lbCluster.HasApiRole() {
			isApiServerTypeDesiredLBClusterPresent = true
		}
	}

	// For each load-balancer that is being deleted construct LbClusterData.
	lbClustersInfo := &utils.LBClustersInfo{
		ClusterID:         request.Desired.ClusterInfo.Id(),
		TargetK8sNodepool: request.Desired.ClusterInfo.NodePools,
	}
	for _, lbCluster := range request.DeletedLbs {
		lbClustersInfo.LbClusters = append(lbClustersInfo.LbClusters, &utils.LBClusterData{
			DesiredLbCluster: nil,
			CurrentLbCluster: lbCluster,
		})
	}

	previousApiEndpoint, err := teardownLoadBalancers(request.Desired, lbClustersInfo, request.ProxyEnvs, isApiServerTypeDesiredLBClusterPresent, u.SpawnProcessLimit)
	if err != nil {
		logger.Err(err).Msgf("Error encountered while tearing down the LoadBalancers")
		return nil, fmt.Errorf("error encountered while tearing down loadbalancers for cluster %s project %s : %w", request.Desired.ClusterInfo.Name, request.ProjectName, err)
	}

	logger.Info().Msgf("Loadbalancers were successfully torn down")

	return &pb.TeardownLBResponse{
		PreviousAPIEndpoint: previousApiEndpoint,
		Desired:             request.Desired,
		DesiredLbs:          request.DesiredLbs,
		DeletedLbs:          request.DeletedLbs,
	}, nil
}

// tearDownLoadBalancers will correctly destroy LB clusters (including correctly selecting the new ApiServer if present).
// If for a K8s cluster a new ApiServerLB is being attached instead of handling the apiEndpoint immediately
// it will be delayed and will send the data to the dataChan which will be used later for the SetupLoadbalancers
// function to bypass generating the certificates for the endpoint multiple times.
func teardownLoadBalancers(
	desiredK8sCluster *spec.K8Scluster,
	lbClustersInfo *utils.LBClustersInfo,
	proxyEnvs *spec.ProxyEnvs,
	attached bool,
	processLimit *semaphore.Weighted,
) (string, error) {
	clusterName := desiredK8sCluster.ClusterInfo.Name

	clusterDirectory := filepath.Join(baseDirectory, outputDirectory, fmt.Sprintf("%s-%s-lbs", clusterName, hash.Create(hash.Length)))
	if err := utils.GenerateLBBaseFiles(clusterDirectory, lbClustersInfo); err != nil {
		return "", fmt.Errorf("error encountered while generating base files for %s", clusterName)
	}

	currentApiServerTypeLBCluster := utils.FindCurrentAPIServerTypeLBCluster(lbClustersInfo.LbClusters)
	// If there is an Api server type LB cluster currently that will be deleted, and we're attaching a
	// new Api server type LB cluster to the K8s cluster, we store the endpoint being used by the
	// current Api server type LB cluster.
	// This will be reused later in the SetUpLoadbalancers function.
	if currentApiServerTypeLBCluster != nil && attached {
		return currentApiServerTypeLBCluster.CurrentLbCluster.Dns.Endpoint, os.RemoveAll(clusterDirectory)
	}

	if err := utils.HandleAPIEndpointChange(currentApiServerTypeLBCluster, lbClustersInfo, proxyEnvs, clusterDirectory, processLimit); err != nil {
		return "", err
	}

	return "", os.RemoveAll(clusterDirectory)
}
