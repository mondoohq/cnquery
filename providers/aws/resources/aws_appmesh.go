// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/appmesh"
	appmesh_types "github.com/aws/aws-sdk-go-v2/service/appmesh/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsAppmesh) id() (string, error) {
	return "aws.appmesh", nil
}

func (a *mqlAwsAppmesh) meshes() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getMeshes(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsAppmesh) getMeshes(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("appmesh>getMeshes>calling aws with region %s", region)

			svc := conn.AppMesh(region)
			ctx := context.Background()
			res := []any{}

			paginator := appmesh.NewListMeshesPaginator(svc, &appmesh.ListMeshesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Warn().Str("region", region).Msg("App Mesh is not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, mesh := range page.Meshes {
					mqlMesh, err := CreateResource(a.MqlRuntime, "aws.appmesh.mesh",
						map[string]*llx.RawData{
							"__id":      llx.StringDataPtr(mesh.Arn),
							"arn":       llx.StringDataPtr(mesh.Arn),
							"name":      llx.StringDataPtr(mesh.MeshName),
							"region":    llx.StringData(region),
							"meshOwner": llx.StringDataPtr(mesh.MeshOwner),
							"createdAt": llx.TimeDataPtr(mesh.CreatedAt),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlMesh)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsAppmeshMesh) id() (string, error) {
	return a.Arn.Data, nil
}

type mqlAwsAppmeshMeshInternal struct {
	fetched  bool
	lock     sync.Mutex
	descResp *appmesh.DescribeMeshOutput
}

func (a *mqlAwsAppmeshMesh) fetchDetail() (*appmesh.DescribeMeshOutput, error) {
	if a.fetched {
		return a.descResp, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.descResp, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	svc := conn.AppMesh(region)
	ctx := context.Background()

	name := a.Name.Data
	resp, err := svc.DescribeMesh(ctx, &appmesh.DescribeMeshInput{
		MeshName: &name,
	})
	if err != nil {
		return nil, err
	}
	a.fetched = true
	a.descResp = resp
	return resp, nil
}

func (a *mqlAwsAppmeshMesh) status() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if resp.Mesh.Status == nil {
		return "", nil
	}
	return string(resp.Mesh.Status.Status), nil
}

func (a *mqlAwsAppmeshMesh) egressFilterType() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if resp.Mesh.Spec == nil || resp.Mesh.Spec.EgressFilter == nil {
		return "", nil
	}
	return string(resp.Mesh.Spec.EgressFilter.Type), nil
}

func (a *mqlAwsAppmeshMesh) serviceDiscoveryIpPreference() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if resp.Mesh.Spec == nil || resp.Mesh.Spec.ServiceDiscovery == nil {
		return "", nil
	}
	return string(resp.Mesh.Spec.ServiceDiscovery.IpPreference), nil
}

func (a *mqlAwsAppmeshMesh) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	svc := conn.AppMesh(region)
	ctx := context.Background()

	arn := a.Arn.Data
	resp, err := svc.ListTagsForResource(ctx, &appmesh.ListTagsForResourceInput{
		ResourceArn: &arn,
	})
	if err != nil {
		return nil, err
	}
	tags := make(map[string]any)
	for _, tag := range resp.Tags {
		tags[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
	}
	return tags, nil
}

func (a *mqlAwsAppmeshMesh) virtualServices() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	svc := conn.AppMesh(region)
	ctx := context.Background()

	meshName := a.Name.Data
	res := []any{}
	paginator := appmesh.NewListVirtualServicesPaginator(svc, &appmesh.ListVirtualServicesInput{
		MeshName: &meshName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, vs := range page.VirtualServices {
			// Describe the virtual service to get status and provider info
			desc, err := svc.DescribeVirtualService(ctx, &appmesh.DescribeVirtualServiceInput{
				MeshName:           &meshName,
				VirtualServiceName: vs.VirtualServiceName,
			})
			if err != nil {
				return nil, err
			}

			status := ""
			providerType := ""
			providerName := ""
			if desc.VirtualService != nil {
				if desc.VirtualService.Status != nil {
					status = string(desc.VirtualService.Status.Status)
				}
				if desc.VirtualService.Spec != nil && desc.VirtualService.Spec.Provider != nil {
					switch p := desc.VirtualService.Spec.Provider.(type) {
					case *appmesh_types.VirtualServiceProviderMemberVirtualNode:
						providerType = "virtualNode"
						providerName = convert.ToValue(p.Value.VirtualNodeName)
					case *appmesh_types.VirtualServiceProviderMemberVirtualRouter:
						providerType = "virtualRouter"
						providerName = convert.ToValue(p.Value.VirtualRouterName)
					}
				}
			}

			mqlVs, err := CreateResource(a.MqlRuntime, "aws.appmesh.virtualService",
				map[string]*llx.RawData{
					"__id":         llx.StringDataPtr(vs.Arn),
					"arn":          llx.StringDataPtr(vs.Arn),
					"name":         llx.StringDataPtr(vs.VirtualServiceName),
					"meshName":     llx.StringData(meshName),
					"region":       llx.StringData(region),
					"status":       llx.StringData(status),
					"providerType": llx.StringData(providerType),
					"providerName": llx.StringData(providerName),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlVs)
		}
	}
	return res, nil
}

func (a *mqlAwsAppmeshMesh) virtualNodes() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	svc := conn.AppMesh(region)
	ctx := context.Background()

	meshName := a.Name.Data
	res := []any{}
	paginator := appmesh.NewListVirtualNodesPaginator(svc, &appmesh.ListVirtualNodesInput{
		MeshName: &meshName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, vn := range page.VirtualNodes {
			mqlVn, err := CreateResource(a.MqlRuntime, "aws.appmesh.virtualNode",
				map[string]*llx.RawData{
					"__id":     llx.StringDataPtr(vn.Arn),
					"arn":      llx.StringDataPtr(vn.Arn),
					"name":     llx.StringDataPtr(vn.VirtualNodeName),
					"meshName": llx.StringData(meshName),
					"region":   llx.StringData(region),
					"status":   llx.StringData(""),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlVn)
		}
	}
	return res, nil
}

func (a *mqlAwsAppmeshVirtualService) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsAppmeshVirtualNode) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsAppmeshVirtualNode) backends() (int64, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	svc := conn.AppMesh(region)
	ctx := context.Background()

	meshName := a.MeshName.Data
	nodeName := a.Name.Data
	resp, err := svc.DescribeVirtualNode(ctx, &appmesh.DescribeVirtualNodeInput{
		MeshName:        &meshName,
		VirtualNodeName: &nodeName,
	})
	if err != nil {
		return 0, err
	}
	if resp.VirtualNode == nil || resp.VirtualNode.Spec == nil {
		return 0, nil
	}
	return int64(len(resp.VirtualNode.Spec.Backends)), nil
}

func (a *mqlAwsAppmeshVirtualNode) serviceDiscovery() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	svc := conn.AppMesh(region)
	ctx := context.Background()

	meshName := a.MeshName.Data
	nodeName := a.Name.Data
	resp, err := svc.DescribeVirtualNode(ctx, &appmesh.DescribeVirtualNodeInput{
		MeshName:        &meshName,
		VirtualNodeName: &nodeName,
	})
	if err != nil {
		return nil, err
	}
	if resp.VirtualNode == nil || resp.VirtualNode.Spec == nil || resp.VirtualNode.Spec.ServiceDiscovery == nil {
		return map[string]any{}, nil
	}
	sd := resp.VirtualNode.Spec.ServiceDiscovery
	result := map[string]any{}
	switch s := sd.(type) {
	case *appmesh_types.ServiceDiscoveryMemberDns:
		result["type"] = "dns"
		result["hostname"] = convert.ToValue(s.Value.Hostname)
		result["responseType"] = string(s.Value.ResponseType)
		result["ipPreference"] = string(s.Value.IpPreference)
	case *appmesh_types.ServiceDiscoveryMemberAwsCloudMap:
		result["type"] = "awsCloudMap"
		result["namespaceName"] = convert.ToValue(s.Value.NamespaceName)
		result["serviceName"] = convert.ToValue(s.Value.ServiceName)
	}
	return result, nil
}
