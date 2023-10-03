// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inventory

import "strings"

func DeprecatedV8CompatAssets(assets []*Asset) {
	for i := range assets {
		DeprecatedV8CompatAsset(assets[i])
	}
}

func DeprecatedV8CompatAsset(asset *Asset) {
	if asset == nil {
		return
	}

	if asset.Platform != nil {
		asset.Platform.DeprecatedV8Kind = Kind2DeprecatedV8Kind(asset.Platform.Kind)
	}

	for i := range asset.Connections {
		conn := asset.Connections[i]
		conn.Kind = Kind2DeprecatedV8Kind(asset.KindString)
	}

	// FIXME: Remove this and solve it at its core
	var ids []string
	for _, id := range asset.PlatformIds {
		if strings.HasPrefix(id, "//") {
			ids = append(ids, id)
		}
	}
	asset.PlatformIds = ids
}

func Kind2DeprecatedV8Kind(kind string) DeprecatedV8_Kind {
	switch kind {
	case "virtual-machine-image":
		return DeprecatedV8_Kind_KIND_VIRTUAL_MACHINE
	case "container-image":
		return DeprecatedV8_Kind_KIND_CONTAINER_IMAGE
	case "code":
		return DeprecatedV8_Kind_KIND_CODE
	case "package":
		return DeprecatedV8_Kind_KIND_PACKAGE
	case "virtual-machine":
		return DeprecatedV8_Kind_KIND_VIRTUAL_MACHINE
	case "container":
		return DeprecatedV8_Kind_KIND_CONTAINER
	case "process":
		return DeprecatedV8_Kind_KIND_PROCESS
	case "api":
		return DeprecatedV8_Kind_KIND_API
	case "bare-metal":
		return DeprecatedV8_Kind_KIND_BARE_METAL
	case "network":
		return DeprecatedV8_Kind_KIND_NETWORK
	case "k8s-object":
		return DeprecatedV8_Kind_KIND_K8S_OBJECT
	case "aws_object":
		return DeprecatedV8_Kind_KIND_AWS_OBJECT
	case "gcp-object":
		return DeprecatedV8_Kind_KIND_GCP_OBJECT
	case "azure-object":
		return DeprecatedV8_Kind_KIND_AZURE_OBJECT
	default:
		return DeprecatedV8_Kind_KIND_UNKNOWN
	}
}
