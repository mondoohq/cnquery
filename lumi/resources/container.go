package resources

import (
	"github.com/google/go-containerregistry/pkg/name"
	"go.mondoo.io/mondoo/lumi"
)

func newLumiContainerImage(runtime *lumi.Runtime, containerImageName string) (interface{}, error) {
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

func (k *lumiContainerImage) id() (string, error) {
	return k.Name()
}

func (k *lumiContainerImage) GetRepository() (interface{}, error) {
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

func newLumiContainerRepository(runtime *lumi.Runtime, repo name.Repository) (interface{}, error) {
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

func (k *lumiContainerRepository) id() (string, error) {
	return k.FullName()
}
