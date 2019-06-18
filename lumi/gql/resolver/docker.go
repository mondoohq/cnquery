package resolver

import (
	"context"

	"github.com/pkg/errors"
	"go.mondoo.io/mondoo/lumi/gql"

	"github.com/docker/docker/client"

	docker_types "github.com/docker/docker/api/types"
	"go.mondoo.io/mondoo/motor/motoros/local"
)

func getDockerClient() (*client.Client, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	cli.NegotiateAPIVersion(context.Background())
	return cli, nil
}

func (r *queryResolver) Docker(ctx context.Context) (*gql.Docker, error) {
	return &gql.Docker{}, nil
}

type dockerResolver struct{ *Resolver }

func (r *dockerResolver) Images(ctx context.Context, obj *gql.Docker) ([]gql.DockerImage, error) {
	_, ok := r.Runtime.Motor.Transport.(*local.LocalTransport)
	if !ok {
		return nil, errors.New("docker is not support for this transport")
	}

	cl, err := getDockerClient()
	if err != nil {
		return nil, err
	}

	dImages, err := cl.ImageList(context.Background(), docker_types.ImageListOptions{})
	if err != nil {
		return nil, err
	}

	imgs := make([]gql.DockerImage, len(dImages))
	for i := range dImages {
		dImg := dImages[i]
		imgs[i] = gql.DockerImage{
			ID:          dImg.ID,
			Size:        dImg.Size,
			Virtualsize: dImg.VirtualSize,
			Tags:        dImg.RepoTags,
		}

		labels := []gql.KeyValue{}
		for k := range dImg.Labels {
			key := k
			value := dImg.Labels[key]
			labels = append(labels, gql.KeyValue{
				Key:   &key,
				Value: &value,
			})
		}
		imgs[i].Labels = labels
	}

	return imgs, nil
}

func (r *dockerResolver) Container(ctx context.Context, obj *gql.Docker) ([]gql.DockerContainer, error) {
	_, ok := r.Runtime.Motor.Transport.(*local.LocalTransport)
	if !ok {
		return nil, errors.New("docker is not support for this transport")
	}

	cl, err := getDockerClient()
	if err != nil {
		return nil, err
	}

	dContainers, err := cl.ContainerList(context.Background(), docker_types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	container := make([]gql.DockerContainer, len(dContainers))
	for i := range dContainers {
		dContainer := dContainers[i]
		container[i] = gql.DockerContainer{
			ID:      dContainer.ID,
			Command: dContainer.Command,
			Image:   dContainer.Image,
			Imageid: dContainer.ImageID,
			Names:   dContainer.Names,
			State:   dContainer.State,
			Status:  dContainer.Status,
		}

		labels := []gql.KeyValue{}
		for k := range dContainer.Labels {
			key := k
			value := dContainer.Labels[key]
			labels = append(labels, gql.KeyValue{
				Key:   &key,
				Value: &value,
			})
		}
		container[i].Labels = labels
	}

	return container, nil
}
