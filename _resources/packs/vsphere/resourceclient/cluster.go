package resourceclient

import (
	"context"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
)

func (c *Client) ListClusters(dc *object.Datacenter) ([]*object.ClusterComputeResource, error) {
	finder := find.NewFinder(c.Client.Client, true)
	finder.SetDatacenter(dc)

	l, err := finder.ClusterComputeResourceList(context.Background(), "*")
	if err != nil && IsNotFound(err) {
		return []*object.ClusterComputeResource{}, nil
	} else if err != nil {
		return nil, err
	}
	return l, nil
}

func (c *Client) Cluster(path string) (*object.ClusterComputeResource, error) {
	finder := find.NewFinder(c.Client.Client, true)
	return finder.ClusterComputeResource(context.Background(), path)
}

func clusterProperties(cluster *object.ClusterComputeResource) (*mo.ClusterComputeResource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultAPITimeout)
	defer cancel()
	var props mo.ClusterComputeResource
	if err := cluster.Properties(ctx, cluster.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

func (c *Client) ClusterProperties(cluster *object.ClusterComputeResource) (map[string]interface{}, error) {
	props, err := clusterProperties(cluster)
	if err != nil {
		return nil, err
	}

	return PropertiesToDict(props)
}
