// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/containerengine"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciOke) id() (string, error) {
	return "oci.oke", nil
}

func (o *mqlOciOke) clusters() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	return ociCollect(o.MqlRuntime, ociScopeTenancyRoot,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			log.Debug().Msgf("calling oci oke with region %s", region)

			svc, err := conn.ContainerEngineClient(region)
			if err != nil {
				return nil, err
			}

			clusters, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]containerengine.ClusterSummary, *string, error) {
				response, err := svc.ListClusters(ctx, containerengine.ListClustersRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, nil, err
				}
				return response.Items, response.OpcNextPage, nil
			})
			if err != nil {
				return nil, err
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
				var securityAttributes map[string]any
				if cluster.EndpointConfig != nil {
					isPublicEndpointEnabled = boolValue(cluster.EndpointConfig.IsPublicIpEnabled)
					securityAttributes = definedTagsToAny(cluster.EndpointConfig.SecurityAttributes)
				}

				// Extract endpoints
				var publicEndpoint, privateEndpoint string
				if cluster.Endpoints != nil {
					publicEndpoint = stringValue(cluster.Endpoints.PublicEndpoint)
					privateEndpoint = stringValue(cluster.Endpoints.PrivateEndpoint)
				}

				// endpointConfig.isPublicIpEnabled only exists for clusters using
				// native VCN networking; it is absent for older clusters that
				// still serve a reachable public API endpoint. Trust an actual
				// published public endpoint over the missing flag, otherwise a
				// publicly reachable control plane reports as private.
				if publicEndpoint != "" {
					isPublicEndpointEnabled = true
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

				freeformTags := make(map[string]interface{}, len(cluster.FreeformTags))
				for k, v := range cluster.FreeformTags {
					freeformTags[k] = v
				}
				definedTags := make(map[string]interface{}, len(cluster.DefinedTags))
				for k, v := range cluster.DefinedTags {
					definedTags[k] = v
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
					"freeformTags":                llx.MapData(freeformTags, types.String),
					"definedTags":                 llx.MapData(definedTags, types.Any),
					"securityAttributes":          llx.MapData(securityAttributes, types.Dict),
					"systemTags":                  llx.MapData(definedTagsToAny(cluster.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlCluster := mqlInstance.(*mqlOciOkeCluster)
				mqlCluster.cacheVcnId = stringValue(cluster.VcnId)
				mqlCluster.region = region
				res = append(res, mqlCluster)
			}

			return res, nil
		})
}

type mqlOciOkeClusterInternal struct {
	lock       sync.Mutex
	fetched    atomic.Bool
	cluster    *containerengine.Cluster
	cacheVcnId string
	region     string
}

func (o *mqlOciOkeCluster) id() (string, error) {
	return "oci.oke.cluster/" + o.Id.Data, nil
}

// initOciOkeCluster resolves a single OKE cluster from the scan asset's
// PlatformId when policies reference `oci.oke.cluster` on a discovered
// oci-oke-cluster asset. Explicit id takes precedence.
func initOciOkeCluster(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	idVal := ociArgString(args, "id")
	if idVal == "" {
		conn := runtime.Connection.(*connection.OciConnection)
		if conn.Conf == nil || conn.Conf.PlatformId == "" {
			return args, nil, nil
		}
		parsed, ok := parseOciObjectPlatformID(conn.Conf.PlatformId)
		if !ok || parsed.service != "oke" || parsed.objectType != "cluster" {
			return args, nil, nil
		}
		idVal = parsed.id
	}

	obj, err := CreateResource(runtime, "oci.oke", nil)
	if err != nil {
		return nil, nil, err
	}
	oke := obj.(*mqlOciOke)

	clusters := oke.GetClusters()
	if clusters.Error != nil {
		return nil, nil, clusters.Error
	}

	for _, raw := range clusters.Data {
		c := raw.(*mqlOciOkeCluster)
		if c.Id.Data == idVal {
			return args, c, nil
		}
	}

	return nil, nil, errors.New("oci.oke.cluster not found: " + idVal)
}

func (o *mqlOciOkeCluster) vcn() (*mqlOciNetworkVcn, error) {
	if o.cacheVcnId == "" || !isOcid(o.cacheVcnId) {
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
	if o.fetched.Load() {
		return o.cluster, nil
	}
	o.lock.Lock()
	defer o.lock.Unlock()
	if o.fetched.Load() {
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
	o.fetched.Store(true)
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
	pools, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]containerengine.NodePoolSummary, *string, error) {
		response, err := svc.ListNodePools(ctx, containerengine.ListNodePoolsRequest{
			CompartmentId: common.String(o.CompartmentID.Data),
			ClusterId:     common.String(clusterId),
			Page:          page,
		})
		if err != nil {
			return nil, nil, err
		}
		return response.Items, response.OpcNextPage, nil
	})
	if err != nil {
		return nil, err
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

		// nodeImageName and nodeImageId are both deprecated in the SDK in favour
		// of nodeSourceDetails, and OCI leaves them empty on node pools created
		// through the modern path - which is every Terraform-provisioned pool.
		// Read the image from nodeSourceDetails so "which image do the workers
		// run" has an answer on current node pools.
		var nodeImageId string
		var bootVolumeSizeInGBs *int64
		if src, ok := np.NodeSourceDetails.(containerengine.NodeSourceViaImageDetails); ok {
			nodeImageId = stringValue(src.ImageId)
			bootVolumeSizeInGBs = src.BootVolumeSizeInGBs
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.oke.nodePool", map[string]*llx.RawData{
			"id":                  llx.StringDataPtr(np.Id),
			"name":                llx.StringDataPtr(np.Name),
			"compartmentID":       llx.StringDataPtr(np.CompartmentId),
			"kubernetesVersion":   llx.StringDataPtr(np.KubernetesVersion),
			"nodeShape":           llx.StringDataPtr(np.NodeShape),
			"nodeShapeConfig":     llx.DictData(nodeShapeConfig),
			"nodeImageName":       llx.StringDataPtr(np.NodeImageName),
			"bootVolumeSizeInGBs": llx.IntDataPtr(bootVolumeSizeInGBs),
			"sshPublicKey":        llx.StringDataPtr(np.SshPublicKey),
			"state":               llx.StringData(string(np.LifecycleState)),
			"systemTags":          llx.MapData(definedTagsToAny(np.SystemTags), types.Dict),
		})
		if err != nil {
			return nil, err
		}
		mqlPool := mqlInstance.(*mqlOciOkeNodePool)
		mqlPool.cacheSubnetIds = subnetIds
		mqlPool.cacheNsgIds = nsgIds
		mqlPool.cacheClusterID = stringValue(np.ClusterId)
		mqlPool.cacheNodeImageId = nodeImageId
		res = append(res, mqlPool)
	}

	return res, nil
}

type mqlOciOkeNodePoolInternal struct {
	cacheSubnetIds   []string
	cacheNsgIds      []string
	cacheClusterID   string
	cacheNodeImageId string
}

func (o *mqlOciOkeNodePool) id() (string, error) {
	return "oci.oke.nodePool/" + o.Id.Data, nil
}

func (o *mqlOciOkeNodePool) nodeImage() (*mqlOciComputeImage, error) {
	return resolveOciImage(o.MqlRuntime, o.cacheNodeImageId, &o.NodeImage)
}

func (o *mqlOciOkeNodePool) subnets() ([]any, error) {
	res := make([]any, 0, len(o.cacheSubnetIds))
	for _, id := range o.cacheSubnetIds {
		mqlSubnet, err := NewResource(o.MqlRuntime, "oci.network.subnet", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			// Skip an element we cannot resolve rather than failing the
			// whole list and losing the ones that did resolve.
			log.Debug().Err(err).Str("subnet", id).Msg("skipping unresolvable oci reference")
			continue
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
			// Skip an element we cannot resolve rather than failing the
			// whole list and losing the ones that did resolve.
			log.Debug().Err(err).Str("nsg", id).Msg("skipping unresolvable oci reference")
			continue
		}
		res = append(res, mqlNsg)
	}
	return res, nil
}
