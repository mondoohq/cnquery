// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"net/url"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// PlatformIdMinioDeployment prefixes the identifier of a MinIO deployment
// asset. The deployment ID that MinIO assigns itself is preferred, because it
// survives a move to a new address; the host is the fallback for a deployment
// that does not report one.
const PlatformIdMinioDeployment = "//platformid.api.mondoo.app/runtime/minio/deployment/"

// PlatformIdMinioHost prefixes the identifier of a MinIO deployment reached by
// address, for deployments that report no deployment ID.
const PlatformIdMinioHost = "//platformid.api.mondoo.app/runtime/minio/host/"

// NewMinioPlatform describes a MinIO deployment asset.
func NewMinioPlatform(host string) *inventory.Platform {
	segments := []string{"saas", "minio", "host", host}

	pf := &inventory.Platform{TechnologyUrlSegments: segments}
	PlatformByName("minio").Apply(pf)
	return pf
}

// NewMinioIdentifier builds the platform ID of a MinIO deployment asset. The
// segment is escaped, since a host carries a colon and a deployment ID is
// server-supplied.
func NewMinioIdentifier(deploymentID, host string) string {
	if deploymentID != "" {
		return PlatformIdMinioDeployment + url.PathEscape(deploymentID)
	}
	return PlatformIdMinioHost + url.PathEscape(host)
}
