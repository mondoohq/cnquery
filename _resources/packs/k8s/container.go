package k8s

import (
	"errors"
	"fmt"

	k8s_resources "go.mondoo.com/cnquery/motor/providers/k8s/resources"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/resources/packs/os"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

type ContainerType string

var (
	EphemeralContainerType ContainerType = "ephemeral"
	InitContainerType      ContainerType = "init"
	ContainerContainerType ContainerType = "container"
)

func getContainers(
	o K8sNamespacedObject, mqlRuntime *resources.Runtime, containerType ContainerType,
) ([]interface{}, error) {
	var containersFunc func(runtime.Object) ([]corev1.Container, error)
	resourceType := ""
	switch containerType {
	case EphemeralContainerType:
		containersFunc = k8s_resources.GetEphemeralContainers
		resourceType = "k8s.ephemeralContainer"
	case InitContainerType:
		containersFunc = k8s_resources.GetInitContainers
		resourceType = "k8s.initContainer"
	case ContainerContainerType:
		containersFunc = k8s_resources.GetContainers
		resourceType = "k8s.container"
	default:
		return nil, fmt.Errorf("unknown container type %s", containerType)
	}

	id, err := objId(o)
	if err != nil {
		return nil, err
	}

	// At this point we already have the cached manifest. We can parse it to retrieve the
	// containers for the resource.
	manifestRaw, err := o.Manifest()
	if err != nil {
		return nil, err
	}

	manifest, ok := manifestRaw.(map[string]interface{})
	if !ok {
		return nil, errors.New("expected manifest to be an object with keys")
	}

	unstr := unstructured.Unstructured{Object: manifest}
	obj := k8s_resources.ConvertToK8sObject(unstr)

	resp := []interface{}{}
	containers, err := containersFunc(obj)
	if err != nil {
		return nil, err
	}
	for i := range containers {

		c := containers[i]

		secContext, err := core.JsonToDict(c.SecurityContext)
		if err != nil {
			return nil, err
		}

		resources, err := core.JsonToDict(c.Resources)
		if err != nil {
			return nil, err
		}

		volumeMounts, err := core.JsonToDictSlice(c.VolumeMounts)
		if err != nil {
			return nil, err
		}

		volumeDevices, err := core.JsonToDictSlice(c.VolumeDevices)
		if err != nil {
			return nil, err
		}

		env, err := core.JsonToDictSlice(c.Env)
		if err != nil {
			return nil, err
		}

		envFrom, err := core.JsonToDictSlice(c.EnvFrom)
		if err != nil {
			return nil, err
		}

		args := []interface{}{
			"uid", id + "/" + c.Name, // container names are unique within a resource
			"name", c.Name,
			"imageName", c.Image,
			"image", c.Image, // deprecated, will be replaced with the containerImage going forward
			"command", core.StrSliceToInterface(c.Command),
			"args", core.StrSliceToInterface(c.Args),
			"volumeMounts", volumeMounts,
			"volumeDevices", volumeDevices,
			"imagePullPolicy", string(c.ImagePullPolicy),
			"securityContext", secContext,
			"workingDir", c.WorkingDir,
			"tty", c.TTY,
			"env", env,
			"envFrom", envFrom,
		}

		if containerType != EphemeralContainerType {
			args = append(args, "resources", resources)
		}

		if containerType == ContainerContainerType {
			livenessProbe, err := core.JsonToDict(c.LivenessProbe)
			if err != nil {
				return nil, err
			}

			readinessProbe, err := core.JsonToDict(c.ReadinessProbe)
			if err != nil {
				return nil, err
			}

			args = append(args, "livenessProbe", livenessProbe, "readinessProbe", readinessProbe)
		}

		mqlContainer, err := mqlRuntime.CreateResource(resourceType, args...)
		if err != nil {
			return nil, err
		}
		resp = append(resp, mqlContainer)
	}
	return resp, nil
}

func (k *mqlK8sEphemeralContainer) id() (string, error) {
	return k.Uid()
}

func (k *mqlK8sEphemeralContainer) GetContainerImage() (interface{}, error) {
	containerImageName, err := k.ImageName()
	if err != nil {
		return nil, err
	}

	return os.NewMqlContainerImage(k.MotorRuntime, containerImageName)
}

func (k *mqlK8sInitContainer) id() (string, error) {
	return k.Uid()
}

func (k *mqlK8sInitContainer) GetContainerImage() (interface{}, error) {
	containerImageName, err := k.ImageName()
	if err != nil {
		return nil, err
	}

	return os.NewMqlContainerImage(k.MotorRuntime, containerImageName)
}

func (k *mqlK8sContainer) id() (string, error) {
	return k.Uid()
}

func (k *mqlK8sContainer) GetContainerImage() (interface{}, error) {
	containerImageName, err := k.ImageName()
	if err != nil {
		return nil, err
	}

	return os.NewMqlContainerImage(k.MotorRuntime, containerImageName)
}
