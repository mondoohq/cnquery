// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"net/url"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// PlatformIdVaultServer prefixes the identifier of a Vault server asset. The
// host is part of the identifier because a cluster name is not unique across
// deployments and is empty on a server that was never initialized.
const PlatformIdVaultServer = "//platformid.api.mondoo.app/runtime/vault/host/"

// RootNamespaceLabel stands in for the root namespace in the technology path.
// The asset URL tree has a fixed depth, so the segment is always present and an
// unnamed root namespace needs a label rather than an empty string.
const RootNamespaceLabel = "root"

// NewVaultServerPlatform describes a Vault server asset. The namespace is part
// of the technology path so Enterprise namespaces group under their server
// rather than collapsing into one bucket.
func NewVaultServerPlatform(host, namespace string) *inventory.Platform {
	label := namespace
	if label == "" {
		label = RootNamespaceLabel
	}
	segments := []string{"saas", "vault", "host", host, "namespace", label}

	pf := &inventory.Platform{TechnologyUrlSegments: segments}
	PlatformByName("vault").Apply(pf)
	return pf
}

// NewVaultServerIdentifier builds the platform ID of a Vault server asset. Both
// segments are escaped, since a namespace carries slashes and would otherwise
// produce an identifier that reads as a deeper path.
func NewVaultServerIdentifier(host, namespace string) string {
	id := PlatformIdVaultServer + url.PathEscape(host)
	if namespace != "" {
		id += "/namespace/" + url.PathEscape(namespace)
	}
	return id
}
