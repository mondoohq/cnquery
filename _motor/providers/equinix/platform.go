// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package equinix

func (t *Provider) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/equinix/projects/" + t.projectId, nil // TODO: this is not specific enough
}
