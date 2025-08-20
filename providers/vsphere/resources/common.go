// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"net/url"

	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/types"
	vmwaretypes "github.com/vmware/govmomi/vim25/types"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
)

// extractTagKeys extracts tag keys from vmware Tag slice
func extractTagKeys(tags []vmwaretypes.Tag) []string {
	tagKeys := make([]string, len(tags))
	for i, tag := range tags {
		tagKeys[i] = tag.Key
	}
	return tagKeys
}

// GetTags retrieves tags for a host system using the vSphere vAPI
// If the vAPI is not available, it will return an empty array instead of an error.
// This maintains backward compatibility with vSphere environments that don't use tags.
func GetTags(ctx context.Context, ref types.ManagedObjectReference, client *vim25.Client, conf *inventory.Config) []string {
	// Create vAPI REST client
	restClient := rest.NewClient(client)

	// Get credentials from connection config
	creds, err := vault.GetPassword(conf.Credentials)
	if err != nil {
		return []string{}
	}

	userInfo := url.UserPassword(creds.User, string(creds.Secret))
	err = restClient.Login(ctx, userInfo)
	if err != nil {
		return []string{}
	}

	tagManager := tags.NewManager(restClient)

	// Get attached tags for the host
	attachedTags, err := tagManager.GetAttachedTags(ctx, ref)
	if err != nil {
		return []string{}
	}

	// Convert tags to string format: "category:tag"
	tagStrings := make([]string, len(attachedTags))
	for i, tag := range attachedTags {
		// Get category information
		category, err := tagManager.GetCategory(ctx, tag.CategoryID)
		if err != nil {
			// If we can't get category, just use tag name
			tagStrings[i] = tag.Name
			continue
		}
		tagStrings[i] = fmt.Sprintf("%s:%s", category.Name, tag.Name)
	}

	return tagStrings
}
