// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resourceclient

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
)

func VmInfo(vm *object.VirtualMachine) (*mo.VirtualMachine, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultAPITimeout)
	defer cancel()
	var props mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

func VmProperties(vm *mo.VirtualMachine) (map[string]interface{}, error) {
	return PropertiesToDict(vm)
}

func AdvancedSettings(vm *object.VirtualMachine) (map[string]interface{}, error) {
	vmInfo, err := VmInfo(vm)
	if err != nil {
		return nil, err
	}

	advancedProps := map[string]interface{}{}
	for i := range vmInfo.Config.ExtraConfig {
		prop := vmInfo.Config.ExtraConfig[i]
		key := prop.GetOptionValue().Key
		value := fmt.Sprintf("%v", prop.GetOptionValue().Value)
		advancedProps[key] = value
	}
	return advancedProps, nil
}

func (c *Client) ListVirtualMachines(dc *object.Datacenter) ([]*object.VirtualMachine, error) {
	finder := find.NewFinder(c.Client.Client, true)
	finder.SetDatacenter(dc)
	res, err := finder.VirtualMachineList(context.Background(), "*")
	if err != nil && IsNotFound(err) {
		return []*object.VirtualMachine{}, nil
	} else if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) VirtualMachineByInventoryPath(path string) (*object.VirtualMachine, error) {
	finder := find.NewFinder(c.Client.Client, true)
	return finder.VirtualMachine(context.Background(), path)
}

func (c *Client) VirtualMachineByMoid(moid types.ManagedObjectReference) (*object.VirtualMachine, error) {
	finder := find.NewFinder(c.Client.Client, true)
	ref, err := finder.ObjectReference(context.Background(), moid)
	if err != nil {
		return nil, err
	}

	switch ref.(type) {
	case *object.VirtualMachine:
		return ref.(*object.VirtualMachine), nil
	}
	return nil, errors.New("reference is not a valid virtual machine")
}

// GetVmTags retrieves tags for a virtual machine using the vSphere vAPI
// If the vAPI is not available, it will return an empty array instead of an error.
// This maintains backward compatibility with vSphere environments that don't use tags.
func (c *Client) GetVmTags(ctx context.Context, vmRef types.ManagedObjectReference, conf *inventory.Config) []string {
	// Create vAPI REST client
	restClient := rest.NewClient(c.Client.Client)

	// Get credentials from connection config
	creds, err := vault.GetPassword(conf.Credentials)
	if err != nil {
		// If credential retrieval fails, return empty array
		return []string{}
	}

	userInfo := url.UserPassword(creds.User, string(creds.Secret))
	err = restClient.Login(ctx, userInfo)
	if err != nil {
		return []string{}
	}

	tagManager := tags.NewManager(restClient)

	// Get attached tags for the VM
	attachedTags, err := tagManager.GetAttachedTags(ctx, vmRef)
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

// IsNotFound returns a boolean indicating whether the error is a not found error.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var e *find.NotFoundError
	return errors.As(err, &e)
}
