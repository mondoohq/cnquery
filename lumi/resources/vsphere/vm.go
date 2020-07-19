package vsphere

import (
	"context"
	"fmt"

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

	dataProps := map[string]interface{}{}
	dataProps["PowerState"] = string(props.Runtime.PowerState)
	dataProps["ConnectionState"] = string(props.Runtime.ConnectionState)
	return dataProps, nil
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
