package resolver

import (
	"context"
	"net/http"
	"strconv"

	"github.com/pkg/errors"

	"go.mondoo.io/mondoo/lumi/gql"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/storage/v1"
)

func (r *queryResolver) Gcp(ctx context.Context) (*gql.GoogleCloudPlatform, error) {
	return &gql.GoogleCloudPlatform{
		Compute: &gql.GcpCompute{},
		Storage: &gql.GcpStorage{},
	}, nil
}

type googleCloudPlatformResolver struct{ *Resolver }

func (r *googleCloudPlatformResolver) Projects(ctx context.Context, obj *gql.GoogleCloudPlatform) ([]*gql.GcpProject, error) {
	client, err := gcpClient(compute.CloudPlatformScope)
	resSrv, err := cloudresourcemanager.New(client)
	if err != nil {
		return nil, err
	}

	projectsResp, err := resSrv.Projects.List().Do()
	if err != nil {
		return nil, err
	}

	projects := make([]*gql.GcpProject, len(projectsResp.Projects))
	for i := range projectsResp.Projects {
		gcpProject := projectsResp.Projects[i]
		projects[i] = &gql.GcpProject{
			ID:     gcpProject.ProjectId,
			Name:   gcpProject.Name,
			Number: strconv.FormatInt(gcpProject.ProjectNumber, 10),
		}
	}
	return projects, nil
}

type gcpComputeResolver struct{ *Resolver }

func (r *gcpComputeResolver) Zones(ctx context.Context, obj *gql.GcpCompute, projectFilter *string) ([]*gql.GcpZone, error) {
	if projectFilter == nil {
		return nil, errors.New("project need to be provided")
	}
	project := *projectFilter

	client, err := gcpClient(compute.CloudPlatformScope)
	svc, err := compute.New(client)
	if err != nil {
		return nil, err
	}

	gpcZones, err := svc.Zones.List(project).Do()
	if err != nil {
		return nil, err
	}

	zones := make([]*gql.GcpZone, len(gpcZones.Items))
	for i := range gpcZones.Items {
		gcpZone := gpcZones.Items[i]
		zones[i] = &gql.GcpZone{
			Name:   gcpZone.Name,
			Region: gcpZone.Region,
			Status: gcpZone.Status,
		}
	}
	return zones, nil

}

func (r *gcpComputeResolver) Instances(ctx context.Context, obj *gql.GcpCompute, projectFilter *string, zoneFilter *string) ([]*gql.GcpComputeInstance, error) {
	if projectFilter == nil {
		return nil, errors.New("project need to be provided")
	}

	if zoneFilter == nil {
		return nil, errors.New("zone need to be provided")
	}

	client, err := gcpClient(compute.ComputeScope)
	if err != nil {
		return nil, err
	}

	project := *projectFilter
	zone := *zoneFilter

	svc, err := compute.New(client)
	if err != nil {
		return nil, err
	}

	il, err := svc.Instances.List(project, zone).Do()
	if err != nil {
		return nil, err
	}

	instances := make([]*gql.GcpComputeInstance, len(il.Items))
	for i := range il.Items {
		instance := il.Items[i]

		instances[i] = &gql.GcpComputeInstance{
			ID:     strconv.FormatUint(instance.Id, 10),
			Kind:   instance.Kind,
			Name:   instance.Name,
			Status: instance.Status,
			Zone:   instance.Zone,
		}

		labels := []*gql.KeyValue{}
		for k := range instance.Labels {
			key := k
			value := instance.Labels[key]
			labels = append(labels, &gql.KeyValue{
				Key:   &key,
				Value: &value,
			})
		}
		instances[i].Labels = labels
	}

	return instances, nil
}

type gcpStorageResolver struct{ *Resolver }

func (r *gcpStorageResolver) Buckets(ctx context.Context, obj *gql.GcpStorage, projectFilter *string) ([]*gql.GcpStorageBucket, error) {
	client, err := gcpClient(storage.DevstorageReadOnlyScope)
	if err != nil {
		return nil, err
	}

	project := "project"

	storageSvc, err := storage.New(client)
	if err != nil {
		return nil, err
	}

	bl, err := storageSvc.Buckets.List(project).Do()
	if err != nil {
		return nil, err
	}

	buckets := make([]*gql.GcpStorageBucket, len(bl.Items))
	for i := range bl.Items {
		bucket := bl.Items[i]

		buckets[i] = &gql.GcpStorageBucket{
			ID:           bucket.Id,
			Name:         bucket.Name,
			Location:     bucket.Location,
			Storageclass: bucket.StorageClass,
		}

		labels := []*gql.KeyValue{}
		for k := range bucket.Labels {
			key := k
			value := bucket.Labels[key]
			labels = append(labels, &gql.KeyValue{
				Key:   &key,
				Value: &value,
			})
		}
		buckets[i].Labels = labels
	}

	return buckets, nil
}

func gcpClient(scope ...string) (*http.Client, error) {
	ctx := context.Background()
	return google.DefaultClient(ctx, scope...)
}
