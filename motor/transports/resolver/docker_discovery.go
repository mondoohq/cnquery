package resolver

import (
	"context"
	"os"
	"strings"

	dopts "github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
)

// parseDockerCLI is doing a small part from client.FromEnv(c)
// but it parses the DOCKER_HOST like the docker cli and not the docker go lib
// DO NOT ASK why docker maintains two implementations
func parseDockerCLIHost(c *client.Client) error {
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		parsedHost, err := dopts.ParseHost(false, host)
		if err != nil {
			return err
		}

		if err := client.WithHost(parsedHost)(c); err != nil {
			return err
		}
	}
	return nil
}

func FromDockerEnv(c *client.Client) error {
	err := client.FromEnv(c)

	// we ignore the parse error since we are going to re-parse it anyway
	if err != nil && !strings.Contains(err.Error(), "unable to parse docker host") {
		return err
	}

	// The docker go client works different than the docker cli
	// therefore we mimic the approach from the docker cli to make it easier for users
	return parseDockerCLIHost(c)
}

func dockerClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(FromDockerEnv)
	if err != nil {
		return nil, err
	}
	cli.NegotiateAPIVersion(context.Background())
	return cli, nil
}

// TODO: this implementation needs to be merged with motorcloud/docker
func NewDockerEngineDiscovery() (*dockerEngineDiscovery, error) {
	dc, err := dockerClient()
	if err != nil {
		return nil, err
	}

	return &dockerEngineDiscovery{
		Client: dc,
	}, nil
}

type dockerEngineDiscovery struct {
	Client *client.Client
}

func (e *dockerEngineDiscovery) client() (*client.Client, error) {
	if e.Client != nil {
		return e.Client, nil
	}
	return nil, errors.New("docker client not initialized")
}

func (e *dockerEngineDiscovery) ContainerList() ([]string, error) {
	dc, err := e.client()
	if err != nil {
		return []string{}, err
	}

	containers, err := dc.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return []string{}, err
	}

	containerShas := []string{}
	for i := range containers {
		containerShas = append(containerShas, containers[i].ID)
	}

	return containerShas, nil
}

// be aware that images are prefixed with sha256:, while containers are not
func (e *dockerEngineDiscovery) ImageList() ([]string, error) {
	dc, err := e.client()
	if err != nil {
		return []string{}, err
	}

	images, err := dc.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		return []string{}, err
	}

	imagesShas := []string{}
	for i := range images {
		imagesShas = append(imagesShas, images[i].ID)
	}

	return imagesShas, nil
}

type ContainerInfo struct {
	ID      string
	Running bool
	Labels  map[string]string
	Arch    string
}

// will resolve name and id to a container id
func (e *dockerEngineDiscovery) ContainerInfo(name string) (ContainerInfo, error) {
	ci := ContainerInfo{}
	dc, err := e.client()
	if err != nil {
		return ci, err
	}

	cdata, err := dc.ContainerInspect(context.Background(), name)
	if err != nil {
		return ci, err
	}

	ci.ID = cdata.ID
	ci.Running = cdata.State.Running

	// fetch docker specific metadata
	labels := map[string]string{}
	labels["mondoo.app/instance"] = cdata.ID
	// labels["mondoo.app/image-id"] = cdata.ImageID
	labels["docker.io/image-name"] = cdata.Image
	labels["docker.io/names"] = name

	ci.Labels = labels

	return ci, nil
}

type ImageInfo struct {
	ID     string
	Name   string
	Labels map[string]string
	Arch   string
}

func (e *dockerEngineDiscovery) ImageInfo(name string) (ImageInfo, error) {
	ii := ImageInfo{}
	dc, err := e.client()
	if err != nil {
		return ii, err
	}

	res, _, err := dc.ImageInspectWithRaw(context.Background(), name)
	if err != nil {
		return ii, err
	}

	switch res.Architecture {
	case "amd64":
		ii.Arch = "x86_64"
	}

	labels := map[string]string{}
	labels["mondoo.app/image-id"] = res.ID
	labels["docker.io/tags"] = strings.Join(res.RepoTags, ",")
	labels["docker.io/digests"] = strings.Join(res.RepoDigests, ",")

	ii.Name = ShortContainerImageID(res.ID)
	ii.ID = res.ID
	ii.Labels = labels
	return ii, nil
}

func ShortContainerImageID(id string) string {
	id = strings.Replace(id, "sha256:", "", -1)
	if len(id) > 12 {
		return id[0:12]
	}
	return id
}
