// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/google/go-containerregistry/pkg/name"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/os/connection"
)

func initContainerImage(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.TarConnection)
	reference := conn.Metadata.Labels["docker.io/digests"]

	ref, err := name.ParseReference(reference)
	if err != nil {
		return nil, nil, err
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
	r, err := CreateResource(runtime, "container.image", map[string]*llx.RawData{
		"reference":      llx.StringData(reference),
		"name":           llx.StringData(ref.Name()),
		"identifier":     llx.StringData(ref.Identifier()),
		"identifierType": llx.StringData(identifierType),
	})
	if err != nil {
		return nil, nil, err
	}

	return nil, r, nil
}

func (k *mqlContainerImage) id() (string, error) {
	return k.Name.Data, nil
}

func (k *mqlContainerImage) repository() (*mqlContainerRepository, error) {
	if k.Name.Error != nil {
		return nil, k.Name.Error
	}

	ref, err := name.ParseReference(k.Name.Data)
	if err != nil {
		return nil, err
	}

	return newLumiContainerRepository(k.MqlRuntime, ref.Context())
}

func newLumiContainerRepository(runtime *plugin.Runtime, repo name.Repository) (*mqlContainerRepository, error) {
	r, err := CreateResource(runtime, "container.repository", map[string]*llx.RawData{
		"name":     llx.StringData(repo.RepositoryStr()),
		"scheme":   llx.StringData(repo.Scheme()),
		"fullName": llx.StringData(repo.Name()),
		"registry": llx.StringData(repo.RegistryStr()),
	})
	if err != nil {
		return nil, err
	}

	return r.(*mqlContainerRepository), nil
}

func (k *mqlContainerRepository) id() (string, error) {
	return k.FullName.Data, nil
}
