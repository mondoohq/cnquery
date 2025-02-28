// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import "go.mondoo.com/cnquery/v11/providers/os/connection/shared"

const GCP Provider = "gcp"

type gcp struct {
	conn shared.Connection
}

func (g *gcp) Provider() Provider {
	return GCP
}

func (g *gcp) Instance() (*InstanceMetadata, error) {
	return nil, nil
}
