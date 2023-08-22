// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

func MondooAzureInstanceID(instanceID string) string {
	return "//platformid.api.mondoo.app/runtime/azure" + instanceID
}
