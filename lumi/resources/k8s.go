package resources

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/cosmo/resources"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/transports"
	k8s_transport "go.mondoo.io/mondoo/motor/transports/k8s"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func k8stransport(t transports.Transport) (*k8s_transport.Transport, error) {
	at, ok := t.(*k8s_transport.Transport)
	if !ok {
		return nil, errors.New("k8s resource is not supported on this transport")
	}
	return at, nil
}

func (k *lumiK8s) id() (string, error) {
	return "k8s", nil
}

func (k *lumiK8s) GetServerVersion() (interface{}, error) {
	kt, err := k8stransport(k.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	return jsonToDict(kt.ServerVersion())
}

func (k *lumiK8s) GetApiResources() ([]interface{}, error) {
	kt, err := k8stransport(k.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	resources, err := kt.SupportedResources()
	if err != nil {
		return nil, err
	}

	// convert to lumi resources
	list := resources.Resources()
	resp := []interface{}{}
	for i := range list {
		entry := list[i]

		lumiK8SResource, err := k.Runtime.CreateResource("k8s.apiresource",
			"name", entry.Resource.Name,
			"singularName", entry.Resource.SingularName,
			"namespaced", entry.Resource.Namespaced,
			"group", entry.Resource.Group,
			"version", entry.Resource.Version,
			"kind", entry.Resource.Kind,
			"shortNames", strSliceToInterface(entry.Resource.ShortNames),
			"categories", strSliceToInterface(entry.Resource.Categories),
		)
		if err != nil {
			return nil, err
		}
		resp = append(resp, lumiK8SResource)
	}

	return resp, nil
}

type resourceConvertFn func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error)

func k8sResourceToLumi(r *lumi.Runtime, kind string, fn resourceConvertFn) ([]interface{}, error) {
	kt, err := k8stransport(r.Motor.Transport)
	if err != nil {
		return nil, err
	}

	result, err := kt.Resources(kind, "")
	if err != nil {
		return nil, err
	}

	resp := []interface{}{}
	for i := range result.RootResources {
		resource := result.RootResources[i]

		obj, err := meta.Accessor(resource)
		if err != nil {
			log.Error().Err(err).Msg("could not access object attributes")
			return nil, err
		}
		objT, err := meta.TypeAccessor(resource)
		if err != nil {
			log.Error().Err(err).Msg("could not access object attributes")
			return nil, err
		}

		lumiK8sResource, err := fn(kind, resource, obj, objT)
		if err != nil {
			return nil, err
		}

		resp = append(resp, lumiK8sResource)
	}

	return resp, nil
}

func (k *lumiK8s) GetNodes() ([]interface{}, error) {
	return k8sResourceToLumi(k.Runtime, "nodes.v1.", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		return k.Runtime.CreateResource("k8s.node",
			"uid", string(obj.GetUID()),
			"name", obj.GetName(),
			"kind", objT.GetKind(),
		)
	})
}

func (k *lumiK8s) GetNamespaces() ([]interface{}, error) {
	return k8sResourceToLumi(k.Runtime, "namespaces", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		return k.Runtime.CreateResource("k8s.namespace",
			"uid", string(obj.GetUID()),
			"name", obj.GetName(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
		)
	})
}

func (k *lumiK8s) GetPods() ([]interface{}, error) {
	return k8sResourceToLumi(k.Runtime, "pods.v1.", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		podSpec, err := jsonToDict(resources.GetPodSpec(resource))
		if err != nil {
			return nil, err
		}

		return k.Runtime.CreateResource("k8s.pod",
			"uid", string(obj.GetUID()),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"labels", strMapToInterface(obj.GetLabels()),
			"annotations", strMapToInterface(obj.GetAnnotations()),
			"apiVersion", objT.GetAPIVersion(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"podSpec", podSpec,
			"manifest", manifest,
		)
	})
}

func (k *lumiK8s) GetDeployments() ([]interface{}, error) {
	return k8sResourceToLumi(k.Runtime, "deployments", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		return k.Runtime.CreateResource("k8s.deployment",
			"uid", string(obj.GetUID()),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
		)
	})
}

func (k *lumiK8s) GetDaemonsets() ([]interface{}, error) {
	return k8sResourceToLumi(k.Runtime, "daemonsets", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		return k.Runtime.CreateResource("k8s.daemonset",
			"uid", string(obj.GetUID()),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
		)
	})
}

func (k *lumiK8s) GetJobs() ([]interface{}, error) {
	return k8sResourceToLumi(k.Runtime, "jobs", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		return k.Runtime.CreateResource("k8s.job",
			"uid", string(obj.GetUID()),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
		)
	})
}

func (k *lumiK8s) GetCronjobs() ([]interface{}, error) {
	return k8sResourceToLumi(k.Runtime, "cronjobs", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		return k.Runtime.CreateResource("k8s.cronjob",
			"uid", string(obj.GetUID()),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
		)
	})
}

func (k *lumiK8sApiresource) id() (string, error) {
	return k.Name()
}

func (k *lumiK8sNode) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sNamespace) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sPod) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sPod) GetContainers() ([]interface{}, error) {
	uid, err := k.Uid()
	if err != nil {
		return nil, err
	}

	kt, err := k8stransport(k.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	result, err := kt.Resources("pods.v1.", "")
	if err != nil {
		return nil, err
	}

	obj, err := resources.FindByUid(result.AllResources, uid)
	if err != nil {
		return nil, err
	}

	resp := []interface{}{}
	containers := resources.GetContainers(obj)
	for i := range containers {

		c := containers[i]

		secContext, err := jsonToDict(c.SecurityContext)
		if err != nil {
			return nil, err
		}

		resources, err := jsonToDict(c.Resources)
		if err != nil {
			return nil, err
		}

		volumeMounts, err := jsonToDictSlice(c.VolumeMounts)
		if err != nil {
			return nil, err
		}

		volumeDevices, err := jsonToDictSlice(c.VolumeDevices)
		if err != nil {
			return nil, err
		}

		livenessProbe, err := jsonToDict(c.LivenessProbe)
		if err != nil {
			return nil, err
		}

		readinessProbe, err := jsonToDict(c.ReadinessProbe)
		if err != nil {
			return nil, err
		}

		lumiContainer, err := k.Runtime.CreateResource("k8s.container",
			"uid", uid+"/"+c.Name, // container names are unique within a pod
			"name", c.Name,
			"image", c.Image,
			"command", strSliceToInterface(c.Command),
			"args", strSliceToInterface(c.Args),
			"resources", resources,
			"volumeMounts", volumeMounts,
			"volumeDevices", volumeDevices,
			"livenessProbe", livenessProbe,
			"readinessProbe", readinessProbe,
			"imagePullPolicy", string(c.ImagePullPolicy),
			"securityContext", secContext,
			"workingDir", c.WorkingDir,
			"tty", c.TTY,
		)
		if err != nil {
			return nil, err
		}
		resp = append(resp, lumiContainer)
	}
	return resp, nil
}

func (k *lumiK8sPod) GetNamespace() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (k *lumiK8sPod) GetNode() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (k *lumiK8sContainer) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sDeployment) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sDeployment) GetNamespace() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (k *lumiK8sDaemonset) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sDaemonset) GetNamespace() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (k *lumiK8sJob) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sJob) GetNamespace() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (k *lumiK8sCronjob) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sCronjob) GetNamespace() (interface{}, error) {
	return nil, errors.New("not implemented")
}
