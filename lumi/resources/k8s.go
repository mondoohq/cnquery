package resources

import (
	"bytes"
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/certificates"
	"go.mondoo.io/mondoo/motor/transports"
	k8s_transport "go.mondoo.io/mondoo/motor/transports/k8s"
	"go.mondoo.io/mondoo/motor/transports/k8s/resources"
	corev1 "k8s.io/api/core/v1"
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

func k8sMetaObject(lumiResource *lumi.Resource) (metav1.Object, error) {
	entry, ok := lumiResource.Cache.Load("_resource")
	if !ok {
		return nil, errors.New("cannot get resource from cache")
	}

	obj, ok := entry.Data.(runtime.Object)
	if !ok {
		return nil, errors.New("cannot get resource from cache")
	}

	return meta.Accessor(obj)
}

func k8sAnnotations(lumiResource *lumi.Resource) (interface{}, error) {
	objM, err := k8sMetaObject(lumiResource)
	if err != nil {
		return nil, err
	}
	return mapTagsToLumiMapTags(objM.GetAnnotations()), nil
}

func k8sLabels(lumiResource *lumi.Resource) (interface{}, error) {
	objM, err := k8sMetaObject(lumiResource)
	if err != nil {
		return nil, err
	}
	return mapTagsToLumiMapTags(objM.GetLabels()), nil
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

	resources, err := kt.SupportedResourceTypes()
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
		r, err := k.Runtime.CreateResource("k8s.node",
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"kind", objT.GetKind(),
		)
		if err != nil {
			return nil, err
		}
		r.LumiResource().Cache.Store("_resource", &lumi.CacheEntry{Data: resource})
		return r, nil
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

		r, err := k.Runtime.CreateResource("k8s.pod",
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
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
		if err != nil {
			return nil, err
		}
		r.LumiResource().Cache.Store("_resource", &lumi.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *lumiK8s) GetDeployments() ([]interface{}, error) {
	return k8sResourceToLumi(k.Runtime, "deployments", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		r, err := k.Runtime.CreateResource("k8s.deployment",
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
		)
		if err != nil {
			return nil, err
		}
		r.LumiResource().Cache.Store("_resource", &lumi.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *lumiK8s) GetDaemonsets() ([]interface{}, error) {
	return k8sResourceToLumi(k.Runtime, "daemonsets", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		r, err := k.Runtime.CreateResource("k8s.daemonset",
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
		)
		if err != nil {
			return nil, err
		}
		r.LumiResource().Cache.Store("_resource", &lumi.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *lumiK8s) GetJobs() ([]interface{}, error) {
	return k8sResourceToLumi(k.Runtime, "jobs", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		r, err := k.Runtime.CreateResource("k8s.job",
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
		)
		if err != nil {
			return nil, err
		}
		r.LumiResource().Cache.Store("_resource", &lumi.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *lumiK8s) GetCronjobs() ([]interface{}, error) {
	return k8sResourceToLumi(k.Runtime, "cronjobs", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		r, err := k.Runtime.CreateResource("k8s.cronjob",
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
		)
		if err != nil {
			return nil, err
		}
		r.LumiResource().Cache.Store("_resource", &lumi.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *lumiK8s) GetSecrets() ([]interface{}, error) {
	return k8sResourceToLumi(k.Runtime, "secrets.v1.", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		s, ok := resource.(*corev1.Secret)
		if !ok {
			return nil, errors.New("not a k8s secret")
		}

		r, err := k.Runtime.CreateResource("k8s.secret",
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"type", string(s.Type),
		)
		if err != nil {
			return nil, err
		}
		r.LumiResource().Cache.Store("_resource", &lumi.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *lumiK8sApiresource) id() (string, error) {
	return k.Name()
}

func (k *lumiK8sNode) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sNode) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sNode) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
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

func (k *lumiK8sPod) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sPod) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
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

func (k *lumiK8sDeployment) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sDeployment) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sDaemonset) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sDaemonset) GetNamespace() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (k *lumiK8sDaemonset) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sDaemonset) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sJob) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sJob) GetNamespace() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (k *lumiK8sJob) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sJob) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sCronjob) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sCronjob) GetNamespace() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (k *lumiK8sCronjob) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sCronjob) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sSecret) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sSecret) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sSecret) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sSecret) GetCertificates() (interface{}, error) {
	entry, ok := k.LumiResource().Cache.Load("_resource")
	if !ok {
		return nil, errors.New("cannot get resource from cache")
	}

	secret, ok := entry.Data.(*corev1.Secret)
	if !ok {
		return nil, errors.New("cannot get resource from cache")
	}

	if secret.Type != corev1.SecretTypeTLS {
		// this is not an error, it just does not contain a certificate
		return nil, nil
	}

	certRawData, ok := secret.Data["tls.crt"]
	if !ok {
		return nil, errors.New("could not find the 'tls.crt' key")
	}
	certs, err := certificates.ParseCertFromPEM(bytes.NewReader(certRawData))
	if err != nil {
		return nil, err
	}

	return certificatesToLumiCertificates(k.Runtime, certs)
}
