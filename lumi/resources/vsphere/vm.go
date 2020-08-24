package vsphere

import (
	"context"
	"errors"
	"fmt"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
)

func vmProperties(vm *object.VirtualMachine) (*mo.VirtualMachine, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultAPITimeout)
	defer cancel()
	var props mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

func VmProperties(vm *object.VirtualMachine) (map[string]interface{}, error) {
	props, err := vmProperties(vm)
	if err != nil {
		return nil, err
	}
	return PropertiesToDict(props)
}

func AdvancedSettings(vm *object.VirtualMachine) (map[string]interface{}, error) {
	props, err := vmProperties(vm)
	if err != nil {
		return nil, err
	}

	advancedProps := map[string]interface{}{}
	for i := range props.Config.ExtraConfig {
		prop := props.Config.ExtraConfig[i]
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

func (c *Client) VirtualMachine(path string) (*object.VirtualMachine, error) {
	finder := find.NewFinder(c.Client.Client, true)
	return finder.VirtualMachine(context.Background(), path)
}

// IsNotFound returns a boolean indicating whether the error is a not found error.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var e *find.NotFoundError
	return errors.As(err, &e)
}
