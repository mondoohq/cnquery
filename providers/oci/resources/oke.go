// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/containerengine"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciOke) id() (string, error) {
	return "oci.oke", nil
}

func (o *mqlOciOke) clusters() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getClusters(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciOke) getClusters(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci oke with region %s", regionResource.Id.Data)

			svc, err := conn.ContainerEngineClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			clusters := []containerengine.ClusterSummary{}
			var page *string
			for {
				response, err := svc.ListClusters(ctx, containerengine.ListClustersRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				clusters = append(clusters, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range clusters {
				cluster := clusters[i]

				var created *time.Time
				if cluster.Metadata != nil && cluster.Metadata.TimeCreated != nil {
					created = &cluster.Metadata.TimeCreated.Time
				}

				// Extract endpoint config
				var isPublicEndpointEnabled bool
				if cluster.EndpointConfig != nil {
					isPublicEndpointEnabled = boolValue(cluster.EndpointConfig.IsPublicIpEnabled)
				}

				// Extract endpoints
				var publicEndpoint, privateEndpoint string
				if cluster.Endpoints != nil {
					publicEndpoint = stringValue(cluster.Endpoints.PublicEndpoint)
					privateEndpoint = stringValue(cluster.Endpoints.PrivateEndpoint)
				}

				// Extract image policy
				var isImagePolicyEnabled bool
				if cluster.ImagePolicyConfig != nil {
					isImagePolicyEnabled = boolValue(cluster.ImagePolicyConfig.IsPolicyEnabled)
				}

				// Extract admission controller options
				var isPodSecurityPolicyEnabled bool
				if cluster.Options != nil && cluster.Options.AdmissionControllerOptions != nil {
					isPodSecurityPolicyEnabled = boolValue(cluster.Options.AdmissionControllerOptions.IsPodSecurityPolicyEnabled)
				}

				// Available upgrades
				upgrades := make([]any, 0, len(cluster.AvailableKubernetesUpgrades))
				for _, u := range cluster.AvailableKubernetesUpgrades {
					upgrades = append(upgrades, u)
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.oke.cluster", map[string]*llx.RawData{
					"id":                          llx.StringDataPtr(cluster.Id),
					"name":                        llx.StringDataPtr(cluster.Name),
					"compartmentID":               llx.StringDataPtr(cluster.CompartmentId),
					"kubernetesVersion":           llx.StringDataPtr(cluster.KubernetesVersion),
					"type":                        llx.StringData(string(cluster.Type)),
					"isPublicEndpointEnabled":     llx.BoolData(isPublicEndpointEnabled),
					"publicEndpoint":              llx.StringData(publicEndpoint),
					"privateEndpoint":             llx.StringData(privateEndpoint),
					"isImagePolicyEnabled":        llx.BoolData(isImagePolicyEnabled),
					"availableKubernetesUpgrades": llx.ArrayData(upgrades, types.String),
					"isPodSecurityPolicyEnabled":  llx.BoolData(isPodSecurityPolicyEnabled),
					"state":                       llx.StringData(string(cluster.LifecycleState)),
					"created":                     llx.TimeDataPtr(created),
				})
				if err != nil {
					return nil, err
				}
				mqlCluster := mqlInstance.(*mqlOciOkeCluster)
				mqlCluster.cacheVcnId = stringValue(cluster.VcnId)
				mqlCluster.region = regionResource.Id.Data
				res = append(res, mqlCluster)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciOkeClusterInternal struct {
	lock       sync.Mutex
	fetched    bool
	cluster    *containerengine.Cluster
	cacheVcnId string
	region     string
}

func (o *mqlOciOkeCluster) id() (string, error) {
	return "oci.oke.cluster/" + o.Id.Data, nil
}

func (o *mqlOciOkeCluster) vcn() (*mqlOciNetworkVcn, error) {
	if o.cacheVcnId == "" {
		o.Vcn.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlVcn, err := NewResource(o.MqlRuntime, "oci.network.vcn", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheVcnId),
	})
	if err != nil {
		return nil, err
	}
	return mqlVcn.(*mqlOciNetworkVcn), nil
}

func (o *mqlOciOkeCluster) fetchCluster() (*containerengine.Cluster, error) {
	if o.fetched {
		return o.cluster, nil
	}
	o.lock.Lock()
	defer o.lock.Unlock()
	if o.fetched {
		return o.cluster, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	svc, err := conn.ContainerEngineClient(o.region)
	if err != nil {
		return nil, err
	}

	resp, err := svc.GetCluster(context.Background(), containerengine.GetClusterRequest{
		ClusterId: common.String(o.Id.Data),
	})
	if err != nil {
		return nil, err
	}

	o.cluster = &resp.Cluster
	o.fetched = true
	return o.cluster, nil
}

func (o *mqlOciOkeCluster) kmsKey() (*mqlOciKmsKey, error) {
	cluster, err := o.fetchCluster()
	if err != nil {
		return nil, err
	}

	kmsKeyId := stringValue(cluster.KmsKeyId)
	if kmsKeyId == "" {
		o.KmsKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlKey, err := NewResource(o.MqlRuntime, "oci.kms.key", map[string]*llx.RawData{
		"id": llx.StringData(kmsKeyId),
	})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlOciKmsKey), nil
}

func (o *mqlOciOkeCluster) nodePools() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	ctx := context.Background()

	svc, err := conn.ContainerEngineClient(o.region)
	if err != nil {
		return nil, err
	}

	clusterId := o.Id.Data
	pools := []containerengine.NodePoolSummary{}
	var page *string
	for {
		response, err := svc.ListNodePools(ctx, containerengine.ListNodePoolsRequest{
			CompartmentId: common.String(o.CompartmentID.Data),
			ClusterId:     common.String(clusterId),
			Page:          page,
		})
		if err != nil {
			return nil, err
		}

		pools = append(pools, response.Items...)

		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	res := make([]any, 0, len(pools))
	for i := range pools {
		np := pools[i]

		nodeShapeConfig, err := convert.JsonToDict(np.NodeShapeConfig)
		if err != nil {
			return nil, err
		}

		subnetIds := []string{}
		nsgIds := []string{}
		if np.NodeConfigDetails != nil {
			for _, placement := range np.NodeConfigDetails.PlacementConfigs {
				if placement.SubnetId != nil {
					subnetIds = append(subnetIds, *placement.SubnetId)
				}
			}
			nsgIds = append(nsgIds, np.NodeConfigDetails.NsgIds...)
		}
		if len(subnetIds) == 0 {
			subnetIds = append(subnetIds, np.SubnetIds...)
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.oke.nodePool", map[string]*llx.RawData{
			"id":                llx.StringDataPtr(np.Id),
			"name":              llx.StringDataPtr(np.Name),
			"compartmentID":     llx.StringDataPtr(np.CompartmentId),
			"kubernetesVersion": llx.StringDataPtr(np.KubernetesVersion),
			"nodeShape":         llx.StringDataPtr(np.NodeShape),
			"nodeShapeConfig":   llx.DictData(nodeShapeConfig),
			"nodeImageName":     llx.StringDataPtr(np.NodeImageName),
			"sshPublicKey":      llx.StringDataPtr(np.SshPublicKey),
			"state":             llx.StringData(string(np.LifecycleState)),
		})
		if err != nil {
			return nil, err
		}
		mqlPool := mqlInstance.(*mqlOciOkeNodePool)
		mqlPool.cacheSubnetIds = subnetIds
		mqlPool.cacheNsgIds = nsgIds
		res = append(res, mqlPool)
	}

	return res, nil
}

type mqlOciOkeNodePoolInternal struct {
	cacheSubnetIds []string
	cacheNsgIds    []string
}

func (o *mqlOciOkeNodePool) id() (string, error) {
	return "oci.oke.nodePool/" + o.Id.Data, nil
}

func (o *mqlOciOkeNodePool) subnets() ([]any, error) {
	res := make([]any, 0, len(o.cacheSubnetIds))
	for _, id := range o.cacheSubnetIds {
		mqlSubnet, err := NewResource(o.MqlRuntime, "oci.network.subnet", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSubnet)
	}
	return res, nil
}

func (o *mqlOciOkeNodePool) networkSecurityGroups() ([]any, error) {
	res := make([]any, 0, len(o.cacheNsgIds))
	for _, id := range o.cacheNsgIds {
		mqlNsg, err := NewResource(o.MqlRuntime, "oci.network.networkSecurityGroup", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlNsg)
	}
	return res, nil
}
