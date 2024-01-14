// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/k8s/connection/shared/resources"
	"go.mondoo.com/cnquery/v10/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ContainerType string

var (
	EphemeralContainerType ContainerType = "ephemeral"
	InitContainerType      ContainerType = "init"
	ContainerContainerType ContainerType = "container"
)

func getContainers(
	obj runtime.Object, meta metav1.Object, pluginRuntime *plugin.Runtime, containerType ContainerType,
) ([]interface{}, error) {
	var containersFunc func(runtime.Object) ([]corev1.Container, error)
	resourceType := ""
	switch containerType {
	case EphemeralContainerType:
		containersFunc = resources.GetEphemeralContainers
		resourceType = "k8s.ephemeralContainer"
	case InitContainerType:
		containersFunc = resources.GetInitContainers
		resourceType = "k8s.initContainer"
	case ContainerContainerType:
		containersFunc = resources.GetContainers
		resourceType = "k8s.container"
	default:
		return nil, fmt.Errorf("unknown container type %s", containerType)
	}

	id, err := objId(obj, meta)
	if err != nil {
		return nil, err
	}

	resp := []interface{}{}
	containers, err := containersFunc(obj)
	if err != nil {
		return nil, err
	}
	for i := range containers {

		c := containers[i]

		secContext, err := convert.JsonToDict(c.SecurityContext)
		if err != nil {
			return nil, err
		}

		volumeMounts, err := convert.JsonToDictSlice(c.VolumeMounts)
		if err != nil {
			return nil, err
		}

		volumeDevices, err := convert.JsonToDictSlice(c.VolumeDevices)
		if err != nil {
			return nil, err
		}

		env, err := convert.JsonToDictSlice(c.Env)
		if err != nil {
			return nil, err
		}

		envFrom, err := convert.JsonToDictSlice(c.EnvFrom)
		if err != nil {
			return nil, err
		}

		args := map[string]*llx.RawData{
			"uid":             llx.StringData(id + "/" + c.Name), // container names are unique within a resource
			"name":            llx.StringData(c.Name),
			"imageName":       llx.StringData(c.Image),
			"image":           llx.StringData(c.Image), // deprecated, will be replaced with the containerImage going forward
			"command":         llx.ArrayData(convert.SliceAnyToInterface(c.Command), types.String),
			"args":            llx.ArrayData(convert.SliceAnyToInterface(c.Args), types.String),
			"volumeMounts":    llx.ArrayData(volumeMounts, types.Dict),
			"volumeDevices":   llx.ArrayData(volumeDevices, types.Dict),
			"imagePullPolicy": llx.StringData(string(c.ImagePullPolicy)),
			"securityContext": llx.DictData(secContext),
			"workingDir":      llx.StringData(c.WorkingDir),
			"tty":             llx.BoolData(c.TTY),
			"env":             llx.ArrayData(env, types.Dict),
			"envFrom":         llx.ArrayData(envFrom, types.Dict),
		}

		if containerType != EphemeralContainerType {
			resources, err := convert.JsonToDict(c.Resources)
			if err != nil {
				return nil, err
			}

			args["resources"] = llx.DictData(resources)
		}

		if containerType == ContainerContainerType {
			livenessProbe, err := convert.JsonToDict(c.LivenessProbe)
			if err != nil {
				return nil, err
			}

			readinessProbe, err := convert.JsonToDict(c.ReadinessProbe)
			if err != nil {
				return nil, err
			}

			args["livenessProbe"] = llx.DictData(livenessProbe)
			args["readinessProbe"] = llx.DictData(readinessProbe)
		}

		mqlContainer, err := CreateResource(pluginRuntime, resourceType, args)
		if err != nil {
			return nil, err
		}
		resp = append(resp, mqlContainer)
	}
	return resp, nil
}

func (k *mqlK8sEphemeralContainer) id() (string, error) {
	return k.Uid.Data, nil
}

func (k *mqlK8sEphemeralContainer) containerImage() (plugin.Resource, error) {
	if k.ImageName.Error != nil {
		return nil, k.ImageName.Error
	}

	c, err := k.MqlRuntime.CreateSharedResource("container.image", map[string]*llx.RawData{
		"reference": llx.StringData(k.ImageName.Data),
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (k *mqlK8sInitContainer) id() (string, error) {
	return k.Uid.Data, nil
}

func (k *mqlK8sInitContainer) containerImage() (plugin.Resource, error) {
	if k.ImageName.Error != nil {
		return nil, k.ImageName.Error
	}

	c, err := k.MqlRuntime.CreateSharedResource("container.image", map[string]*llx.RawData{
		"reference": llx.StringData(k.ImageName.Data),
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (k *mqlK8sContainer) id() (string, error) {
	return k.Uid.Data, nil
}

func (k *mqlK8sContainer) containerImage() (plugin.Resource, error) {
	if k.ImageName.Error != nil {
		return nil, k.ImageName.Error
	}

	c, err := k.MqlRuntime.CreateSharedResource("container.image", map[string]*llx.RawData{
		"reference": llx.StringData(k.ImageName.Data),
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}
