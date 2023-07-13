package resourceclient

import (
	"context"
	"errors"
	"fmt"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
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

// IsNotFound returns a boolean indicating whether the error is a not found error.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var e *find.NotFoundError
	return errors.As(err, &e)
}
