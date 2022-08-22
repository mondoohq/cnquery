package os

import (
	"github.com/google/go-containerregistry/pkg/name"
	"go.mondoo.io/mondoo/resources"
)

func NewMqlContainerImage(runtime *resources.Runtime, containerImageName string) (interface{}, error) {
	ref, err := name.ParseReference(containerImageName)
	if err != nil {
		return nil, err
	}

	identifierType := ""
	switch ref.(type) {
	case name.Tag:
		identifierType = "tag"
	case name.Digest:
		identifierType = "digest"
	}

	// "index.docker.io/library/coredns:latest"
	// name: "index.docker.io/library/coredns:latest"
	// identifier: latest
	// identifierType: tag
	r, err := runtime.CreateResource("container.image",
		"name", ref.Name(),
		"identifier", ref.Identifier(),
		"identifierType", identifierType,
	)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (k *mqlContainerImage) id() (string, error) {
	return k.Name()
}

func (k *mqlContainerImage) GetRepository() (interface{}, error) {
	containerImageName, err := k.Name()
	if err != nil {
		return nil, err
	}

	ref, err := name.ParseReference(containerImageName)
	if err != nil {
		return nil, err
	}

	return newLumiContainerRepository(k.MotorRuntime, ref.Context())
}

func newLumiContainerRepository(runtime *resources.Runtime, repo name.Repository) (interface{}, error) {
	r, err := runtime.CreateResource("container.repository",
		"name", repo.RepositoryStr(),
		"scheme", repo.Scheme(),
		"fullName", repo.Name(),
		"registry", repo.RegistryStr(),
	)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (k *mqlContainerRepository) id() (string, error) {
	return k.FullName()
}
