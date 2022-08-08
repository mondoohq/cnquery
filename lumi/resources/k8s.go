package resources

import (
	"bytes"
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/certificates"
	"go.mondoo.io/mondoo/motor/providers"
	k8s_transport "go.mondoo.io/mondoo/motor/providers/k8s"
	"go.mondoo.io/mondoo/motor/providers/k8s/resources"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacauthorizationv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func k8stransport(t providers.Transport) (k8s_transport.Transport, error) {
	at, ok := t.(k8s_transport.Transport)
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
	kt, err := k8stransport(k.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	return jsonToDict(kt.ServerVersion())
}

func (k *lumiK8s) GetApiResources() ([]interface{}, error) {
	kt, err := k8stransport(k.MotorRuntime.Motor.Transport)
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

		lumiK8SResource, err := k.MotorRuntime.CreateResource("k8s.apiresource",
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

	result, err := kt.Resources(kind, "", "")
	if err != nil {
		return nil, err
	}

	resp := []interface{}{}
	for i := range result.Resources {
		resource := result.Resources[i]

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
	return k8sResourceToLumi(k.MotorRuntime, "nodes.v1.", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		r, err := k.MotorRuntime.CreateResource("k8s.node",
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
	return k8sResourceToLumi(k.MotorRuntime, "namespaces", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		return k.MotorRuntime.CreateResource("k8s.namespace",
			"uid", string(obj.GetUID()),
			"name", obj.GetName(),
			"created", &ts.Time,
			"manifest", manifest,
		)
	})
}

func (k *lumiK8s) GetPods() ([]interface{}, error) {
	return k8sResourceToLumi(k.MotorRuntime, "pods.v1.", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		podSpec, err := resources.GetPodSpec(resource)
		if err != nil {
			return nil, err
		}

		podSpecDict, err := jsonToDict(podSpec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.pod",
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"labels", strMapToInterface(obj.GetLabels()),
			"annotations", strMapToInterface(obj.GetAnnotations()),
			"apiVersion", objT.GetAPIVersion(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"podSpec", podSpecDict,
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
	return k8sResourceToLumi(k.MotorRuntime, "deployments", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.deployment",
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
	return k8sResourceToLumi(k.MotorRuntime, "daemonsets", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.daemonset",
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

func (k *lumiK8s) GetStatefulsets() ([]interface{}, error) {
	return k8sResourceToLumi(k.MotorRuntime, "statefulsets", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.statefulset",
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

func (k *lumiK8s) GetReplicasets() ([]interface{}, error) {
	return k8sResourceToLumi(k.MotorRuntime, "replicasets", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.replicaset",
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
	return k8sResourceToLumi(k.MotorRuntime, "jobs", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.job",
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
	return k8sResourceToLumi(k.MotorRuntime, "cronjobs", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.cronjob",
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
	return k8sResourceToLumi(k.MotorRuntime, "secrets.v1.", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		s, ok := resource.(*corev1.Secret)
		if !ok {
			return nil, errors.New("not a k8s secret")
		}

		r, err := k.MotorRuntime.CreateResource("k8s.secret",
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

func (k *lumiK8s) GetPodSecurityPolicies() ([]interface{}, error) {
	return k8sResourceToLumi(k.MotorRuntime, "podsecuritypolicies", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		psp, ok := resource.(*policyv1beta1.PodSecurityPolicy)
		if !ok {
			return nil, errors.New("not a k8s podsecuritypolicy")
		}

		spec, err := jsonToDict(psp.Spec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.podsecuritypolicy",
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"spec", spec,
		)
		if err != nil {
			return nil, err
		}
		r.LumiResource().Cache.Store("_resource", &lumi.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *lumiK8s) GetServices() ([]interface{}, error) {
	return k8sResourceToLumi(k.MotorRuntime, "services", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		srv, ok := resource.(*corev1.Service)
		if !ok {
			return nil, errors.New("not a k8s service")
		}

		spec, err := jsonToDict(srv.Spec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.service",
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"spec", spec,
		)
		if err != nil {
			return nil, err
		}
		r.LumiResource().Cache.Store("_resource", &lumi.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *lumiK8s) GetConfigmaps() ([]interface{}, error) {
	return k8sResourceToLumi(k.MotorRuntime, "configmaps", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		cm, ok := resource.(*corev1.ConfigMap)
		if !ok {
			return nil, errors.New("not a k8s configmap")
		}

		r, err := k.MotorRuntime.CreateResource("k8s.configmap",
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"data", strMapToInterface(cm.Data),
		)
		if err != nil {
			return nil, err
		}
		r.LumiResource().Cache.Store("_resource", &lumi.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *lumiK8s) GetNetworkPolicies() ([]interface{}, error) {
	return k8sResourceToLumi(k.MotorRuntime, "networkpolicies", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		networkPolicies, ok := resource.(*networkingv1.NetworkPolicy)
		if !ok {
			return nil, errors.New("not a k8s networkpolicy")
		}

		spec, err := jsonToDict(networkPolicies.Spec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.networkpolicy",
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"spec", spec,
		)
		if err != nil {
			return nil, err
		}
		r.LumiResource().Cache.Store("_resource", &lumi.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *lumiK8s) GetServiceaccounts() ([]interface{}, error) {
	return k8sResourceToLumi(k.MotorRuntime, "serviceaccounts", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		serviceAccount, ok := resource.(*corev1.ServiceAccount)
		if !ok {
			return nil, errors.New("not a k8s serviceaccount")
		}

		secrets, err := jsonToDictSlice(serviceAccount.Secrets)
		if err != nil {
			return nil, err
		}

		imagePullSecrets, err := jsonToDictSlice(serviceAccount.ImagePullSecrets)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.serviceaccount",
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"secrets", secrets,
			"imagePullSecrets", imagePullSecrets,
			"automountServiceAccountToken", toBool(serviceAccount.AutomountServiceAccountToken),
		)
		if err != nil {
			return nil, err
		}
		r.LumiResource().Cache.Store("_resource", &lumi.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *lumiK8s) GetClusterroles() ([]interface{}, error) {
	return k8sResourceToLumi(k.MotorRuntime, "clusterroles", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		clusterRole, ok := resource.(*rbacauthorizationv1.ClusterRole)
		if !ok {
			return nil, errors.New("not a k8s clusterrole")
		}

		rules, err := jsonToDictSlice(clusterRole.Rules)
		if err != nil {
			return nil, err
		}

		aggregationRule, err := jsonToDict(clusterRole.AggregationRule)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.rbac.clusterrole",
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"rules", rules,
			"aggregationRule", aggregationRule,
		)
		if err != nil {
			return nil, err
		}
		r.LumiResource().Cache.Store("_resource", &lumi.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *lumiK8s) GetRoles() ([]interface{}, error) {
	return k8sResourceToLumi(k.MotorRuntime, "roles", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		clusterRole, ok := resource.(*rbacauthorizationv1.Role)
		if !ok {
			return nil, errors.New("not a k8s role")
		}

		rules, err := jsonToDictSlice(clusterRole.Rules)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.rbac.role",
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"rules", rules,
		)
		if err != nil {
			return nil, err
		}
		r.LumiResource().Cache.Store("_resource", &lumi.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *lumiK8s) GetClusterrolebindings() ([]interface{}, error) {
	return k8sResourceToLumi(k.MotorRuntime, "clusterrolebindings", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		clusterRoleBinding, ok := resource.(*rbacauthorizationv1.ClusterRoleBinding)
		if !ok {
			return nil, errors.New("not a k8s clusterrolebinding")
		}

		subjects, err := jsonToDictSlice(clusterRoleBinding.Subjects)
		if err != nil {
			return nil, err
		}

		roleRef, err := jsonToDict(clusterRoleBinding.RoleRef)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.rbac.clusterrolebinding",
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"subjects", subjects,
			"roleRef", roleRef,
		)
		if err != nil {
			return nil, err
		}
		r.LumiResource().Cache.Store("_resource", &lumi.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *lumiK8s) GetRolebindings() ([]interface{}, error) {
	return k8sResourceToLumi(k.MotorRuntime, "rolebinding", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := jsonToDict(resource)
		if err != nil {
			return nil, err
		}

		roleBinding, ok := resource.(*rbacauthorizationv1.RoleBinding)
		if !ok {
			return nil, errors.New("not a k8s rolebinding")
		}

		subjects, err := jsonToDictSlice(roleBinding.Subjects)
		if err != nil {
			return nil, err
		}

		roleRef, err := jsonToDict(roleBinding.RoleRef)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.rbac.rolebinding",
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"subjects", subjects,
			"roleRef", roleRef,
		)
		if err != nil {
			return nil, err
		}
		r.LumiResource().Cache.Store("_resource", &lumi.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *lumiK8s) GetCustomresources() ([]interface{}, error) {
	kt, err := k8stransport(k.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	result, err := kt.Resources("CustomResourceDefinition", "", "")
	if err != nil {
		return nil, err
	}

	resp := []interface{}{}
	for i := range result.Resources {
		resource := result.Resources[i]

		// resource.
		crd, err := meta.Accessor(resource)
		if err != nil {
			log.Error().Err(err).Msg("could not access object attributes")
			return nil, err
		}

		lumiResources, err := k8sResourceToLumi(k.MotorRuntime, crd.GetName(), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
			ts := obj.GetCreationTimestamp()

			manifest, err := jsonToDict(resource)
			if err != nil {
				log.Error().Err(err).Msg("couldn't convert resource to json dict")
				return nil, err
			}

			r, err := k.MotorRuntime.CreateResource("k8s.customresource",
				"uid", string(obj.GetUID()),
				"resourceVersion", obj.GetResourceVersion(),
				"name", obj.GetName(),
				"namespace", obj.GetNamespace(),
				"kind", objT.GetKind(),
				"created", &ts.Time,
				"manifest", manifest,
			)
			if err != nil {
				log.Error().Err(err).Msg("couldn't create resource")
				return nil, err
			}
			r.LumiResource().Cache.Store("_resource", &lumi.CacheEntry{Data: resp})
			return r, nil
		})
		resp = append(resp, lumiResources...)
	}
	return resp, nil
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

func (k *lumiK8sCustomresource) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sCustomresource) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sCustomresource) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sPod) id() (string, error) {
	return k.Uid()
}

func (p *lumiK8sPod) init(args *lumi.Args) (*lumi.Args, K8sPod, error) {
	// pass-through if all args are already provided
	if len(*args) > 2 {
		return args, nil, nil
	}

	// get platform identifier infos
	identifierUid, identifierName, identifierNamespace, err := getPlatformIdentifierElements(p.MotorRuntime.Motor.Transport)
	if err != nil {
		return args, nil, nil
	}

	obj, err := p.MotorRuntime.CreateResource("k8s")
	if err != nil {
		return nil, nil, err
	}
	k8sResource := obj.(K8s)

	pods, err := k8sResource.Pods()
	if err != nil {
		return nil, nil, err
	}

	var matchFn func(configMap K8sPod) bool

	var uidRaw string
	if len(*args) == 0 {
		uidRaw = identifierUid
	} else if _, ok := (*args)["uid"]; ok {
		uidRaw = (*args)["uid"].(string)
	}

	if uidRaw != "" {
		matchFn = func(configMap K8sPod) bool {
			uid, _ := configMap.Uid()
			if uid == uidRaw {
				return true
			}
			return false
		}
	}

	var nameRaw string
	var namespaceRaw string
	if _, ok := (*args)["name"]; ok {
		nameRaw = (*args)["name"].(string)
	}
	if _, ok := (*args)["namespace"]; ok {
		namespaceRaw = (*args)["namespace"].(string)
	}
	if nameRaw == "" && namespaceRaw == "" {
		nameRaw = identifierName
		namespaceRaw = identifierNamespace
	}
	if nameRaw != "" && namespaceRaw != "" {
		matchFn = func(configMap K8sPod) bool {
			name, _ := configMap.Name()
			namespace, _ := configMap.Namespace()
			if name == nameRaw && namespace == namespaceRaw {
				return true
			}
			return false
		}
	}

	for i := range pods {
		configMap := pods[i].(K8sPod)
		if matchFn(configMap) {
			return nil, configMap, nil
		}
	}

	return args, nil, nil
}

func (k *lumiK8sPod) GetInitContainers() ([]interface{}, error) {
	uid, err := k.Uid()
	if err != nil {
		return nil, err
	}

	// At this point we already have the cached Pod manifest. We can parse it to retrieve the
	// containers for the pod.
	manifest, err := k.Manifest()
	if err != nil {
		return nil, err
	}
	unstr := unstructured.Unstructured{Object: manifest}
	obj := resources.ConvertToK8sObject(unstr)

	resp := []interface{}{}
	containers, err := resources.GetInitContainers(obj)
	if err != nil {
		return nil, err
	}
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

		lumiContainer, err := k.MotorRuntime.CreateResource("k8s.initContainer",
			"uid", uid+"/"+c.Name, // container names are unique within a pod
			"name", c.Name,
			"imageName", c.Image,
			"image", c.Image, // deprecated, will be replaced with the containerImage going forward
			"command", strSliceToInterface(c.Command),
			"args", strSliceToInterface(c.Args),
			"resources", resources,
			"volumeMounts", volumeMounts,
			"volumeDevices", volumeDevices,
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

func (k *lumiK8sPod) GetContainers() ([]interface{}, error) {
	uid, err := k.Uid()
	if err != nil {
		return nil, err
	}

	// At this point we already have the cached Pod manifest. We can parse it to retrieve the
	// containers for the pod.
	manifest, err := k.Manifest()
	if err != nil {
		return nil, err
	}
	unstr := unstructured.Unstructured{Object: manifest}
	obj := resources.ConvertToK8sObject(unstr)

	resp := []interface{}{}
	containers, err := resources.GetContainers(obj)
	if err != nil {
		return nil, err
	}
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

		lumiContainer, err := k.MotorRuntime.CreateResource("k8s.container",
			"uid", uid+"/"+c.Name, // container names are unique within a pod
			"name", c.Name,
			"imageName", c.Image,
			"image", c.Image, // deprecated, will be replaced with the containerImage going forward
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

func (k *lumiK8sPod) GetNode() (K8sNode, error) {
	podSpec, err := k.PodSpec()
	if err != nil {
		return nil, err
	}

	obj, err := k.MotorRuntime.CreateResource("k8s")
	if err != nil {
		return nil, err
	}
	k8sResource := obj.(K8s)

	nodes, err := k8sResource.Nodes()
	if err != nil {
		return nil, err
	}

	matchFn := func(node K8sNode) bool {
		name, _ := node.Name()
		if name == podSpec["nodeName"] {
			return true
		}
		return false
	}

	for i := range nodes {
		node := nodes[i].(K8sNode)
		if matchFn(node) {
			return node, nil
		}
	}

	return nil, nil
}

func (k *lumiK8sInitContainer) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sInitContainer) GetContainerImage() (interface{}, error) {
	containerImageName, err := k.ImageName()
	if err != nil {
		return nil, err
	}

	return newLumiContainerImage(k.MotorRuntime, containerImageName)
}

func (k *lumiK8sContainer) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sContainer) GetContainerImage() (interface{}, error) {
	containerImageName, err := k.ImageName()
	if err != nil {
		return nil, err
	}

	return newLumiContainerImage(k.MotorRuntime, containerImageName)
}

func (k *lumiK8sDeployment) id() (string, error) {
	return k.Uid()
}

func (p *lumiK8sDeployment) init(args *lumi.Args) (*lumi.Args, K8sDeployment, error) {
	// pass-through if all args are already provided
	if len(*args) == 0 || len(*args) > 2 {
		return args, nil, nil
	}

	// search for existing resources if uid or name/namespace is provided
	obj, err := p.MotorRuntime.CreateResource("k8s")
	if err != nil {
		return nil, nil, err
	}
	k8sResource := obj.(K8s)

	secrets, err := k8sResource.Deployments()
	if err != nil {
		return nil, nil, err
	}

	var matchFn func(configMap K8sDeployment) bool

	uidRaw := (*args)["uid"]
	if uidRaw != nil {
		matchFn = func(configMap K8sDeployment) bool {
			uid, _ := configMap.Uid()
			if uid == uidRaw.(string) {
				return true
			}
			return false
		}
	}

	nameRaw := (*args)["name"]
	namespaceRaw := (*args)["namespace"]
	if nameRaw != nil && namespaceRaw != nil {
		matchFn = func(configMap K8sDeployment) bool {
			name, _ := configMap.Name()
			namespace, _ := configMap.Namespace()
			if name == nameRaw.(string) && namespace == namespaceRaw.(string) {
				return true
			}
			return false
		}
	}

	for i := range secrets {
		configMap := secrets[i].(K8sDeployment)
		if matchFn(configMap) {
			return nil, configMap, nil
		}
	}

	return args, nil, nil
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

func (p *lumiK8sDaemonset) init(args *lumi.Args) (*lumi.Args, K8sDaemonset, error) {
	// pass-through if all args are already provided
	if len(*args) == 0 || len(*args) > 2 {
		return args, nil, nil
	}

	// search for existing resources if uid or name/namespace is provided
	obj, err := p.MotorRuntime.CreateResource("k8s")
	if err != nil {
		return nil, nil, err
	}
	k8sResource := obj.(K8s)

	secrets, err := k8sResource.Daemonsets()
	if err != nil {
		return nil, nil, err
	}

	var matchFn func(configMap K8sDaemonset) bool

	uidRaw := (*args)["uid"]
	if uidRaw != nil {
		matchFn = func(configMap K8sDaemonset) bool {
			uid, _ := configMap.Uid()
			if uid == uidRaw.(string) {
				return true
			}
			return false
		}
	}

	nameRaw := (*args)["name"]
	namespaceRaw := (*args)["namespace"]
	if nameRaw != nil && namespaceRaw != nil {
		matchFn = func(configMap K8sDaemonset) bool {
			name, _ := configMap.Name()
			namespace, _ := configMap.Namespace()
			if name == nameRaw.(string) && namespace == namespaceRaw.(string) {
				return true
			}
			return false
		}
	}

	for i := range secrets {
		configMap := secrets[i].(K8sDaemonset)
		if matchFn(configMap) {
			return nil, configMap, nil
		}
	}

	return args, nil, nil
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

func (k *lumiK8sStatefulset) id() (string, error) {
	return k.Uid()
}

func (p *lumiK8sStatefulset) init(args *lumi.Args) (*lumi.Args, K8sStatefulset, error) {
	// pass-through if all args are already provided
	if len(*args) > 2 {
		return args, nil, nil
	}

	// get platform identifier infos
	identifierUid, identifierName, identifierNamespace, err := getPlatformIdentifierElements(p.MotorRuntime.Motor.Transport)
	if err != nil {
		return args, nil, nil
	}

	// search for existing resources if uid or name/namespace is provided
	obj, err := p.MotorRuntime.CreateResource("k8s")
	if err != nil {
		return nil, nil, err
	}
	k8sResource := obj.(K8s)

	statefulSets, err := k8sResource.Statefulsets()
	if err != nil {
		return nil, nil, err
	}

	var matchFn func(statefulset K8sStatefulset) bool

	var uidRaw string
	if len(*args) == 0 {
		uidRaw = identifierUid
	} else if _, ok := (*args)["uid"]; ok {
		uidRaw = (*args)["uid"].(string)
	}

	if uidRaw != "" {
		matchFn = func(statefulset K8sStatefulset) bool {
			uid, _ := statefulset.Uid()
			return uid == uidRaw
		}
	}

	var nameRaw string
	var namespaceRaw string
	if _, ok := (*args)["name"]; ok {
		nameRaw = (*args)["name"].(string)
	}
	if _, ok := (*args)["namespace"]; ok {
		namespaceRaw = (*args)["namespace"].(string)
	}
	if nameRaw == "" && namespaceRaw == "" {
		nameRaw = identifierName
		namespaceRaw = identifierNamespace
	}
	if nameRaw != "" && namespaceRaw != "" {
		matchFn = func(statefulset K8sStatefulset) bool {
			name, _ := statefulset.Name()
			namespace, _ := statefulset.Namespace()
			return name == nameRaw && namespace == namespaceRaw
		}
	}

	for i := range statefulSets {
		statefulset := statefulSets[i].(K8sStatefulset)
		if matchFn(statefulset) {
			return nil, statefulset, nil
		}
	}

	return args, nil, nil
}

func (k *lumiK8sStatefulset) GetNamespace() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (k *lumiK8sStatefulset) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sStatefulset) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sReplicaset) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sReplicaset) GetNamespace() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (k *lumiK8sReplicaset) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sReplicaset) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sJob) id() (string, error) {
	return k.Uid()
}

func (p *lumiK8sJob) init(args *lumi.Args) (*lumi.Args, K8sJob, error) {
	// pass-through if all args are already provided
	if len(*args) == 0 || len(*args) > 2 {
		return args, nil, nil
	}

	// search for existing resources if uid or name/namespace is provided
	obj, err := p.MotorRuntime.CreateResource("k8s")
	if err != nil {
		return nil, nil, err
	}
	k8sResource := obj.(K8s)

	secrets, err := k8sResource.Jobs()
	if err != nil {
		return nil, nil, err
	}

	var matchFn func(configMap K8sJob) bool

	uidRaw := (*args)["uid"]
	if uidRaw != nil {
		matchFn = func(configMap K8sJob) bool {
			uid, _ := configMap.Uid()
			if uid == uidRaw.(string) {
				return true
			}
			return false
		}
	}

	nameRaw := (*args)["name"]
	namespaceRaw := (*args)["namespace"]
	if nameRaw != nil && namespaceRaw != nil {
		matchFn = func(configMap K8sJob) bool {
			name, _ := configMap.Name()
			namespace, _ := configMap.Namespace()
			if name == nameRaw.(string) && namespace == namespaceRaw.(string) {
				return true
			}
			return false
		}
	}

	for i := range secrets {
		configMap := secrets[i].(K8sJob)
		if matchFn(configMap) {
			return nil, configMap, nil
		}
	}

	return args, nil, nil
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

func (p *lumiK8sCronjob) init(args *lumi.Args) (*lumi.Args, K8sCronjob, error) {
	// pass-through if all args are already provided
	if len(*args) > 2 {
		return args, nil, nil
	}

	// get platform identifier infos
	identifierUid, identifierName, identifierNamespace, err := getPlatformIdentifierElements(p.MotorRuntime.Motor.Transport)
	if err != nil {
		return args, nil, nil
	}

	// search for existing resources if uid or name/namespace is provided
	obj, err := p.MotorRuntime.CreateResource("k8s")
	if err != nil {
		return nil, nil, err
	}
	k8sResource := obj.(K8s)

	cronJobs, err := k8sResource.Cronjobs()
	if err != nil {
		return nil, nil, err
	}

	var matchFn func(configMap K8sCronjob) bool

	var uidRaw string
	if len(*args) == 0 {
		uidRaw = identifierUid
	} else if _, ok := (*args)["uid"]; ok {
		uidRaw = (*args)["uid"].(string)
	}

	if uidRaw != "" {
		matchFn = func(configMap K8sCronjob) bool {
			uid, _ := configMap.Uid()
			if uid == uidRaw {
				return true
			}
			return false
		}
	}

	var nameRaw string
	var namespaceRaw string
	if _, ok := (*args)["name"]; ok {
		nameRaw = (*args)["name"].(string)
	}
	if _, ok := (*args)["namespace"]; ok {
		namespaceRaw = (*args)["namespace"].(string)
	}
	if nameRaw == "" && namespaceRaw == "" {
		nameRaw = identifierName
		namespaceRaw = identifierNamespace
	}
	if nameRaw != "" && namespaceRaw != "" {
		matchFn = func(configMap K8sCronjob) bool {
			name, _ := configMap.Name()
			namespace, _ := configMap.Namespace()
			if name == nameRaw && namespace == namespaceRaw {
				return true
			}
			return false
		}
	}

	for i := range cronJobs {
		configMap := cronJobs[i].(K8sCronjob)
		if matchFn(configMap) {
			return nil, configMap, nil
		}
	}

	return args, nil, nil
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

func (p *lumiK8sSecret) init(args *lumi.Args) (*lumi.Args, K8sSecret, error) {
	// pass-through if all args are already provided
	if len(*args) == 0 || len(*args) > 2 {
		return args, nil, nil
	}

	// search for existing resources if uid or name/namespace is provided
	obj, err := p.MotorRuntime.CreateResource("k8s")
	if err != nil {
		return nil, nil, err
	}
	k8sResource := obj.(K8s)

	secrets, err := k8sResource.Secrets()
	if err != nil {
		return nil, nil, err
	}

	var matchFn func(configMap K8sSecret) bool

	uidRaw := (*args)["uid"]
	if uidRaw != nil {
		matchFn = func(configMap K8sSecret) bool {
			uid, _ := configMap.Uid()
			if uid == uidRaw.(string) {
				return true
			}
			return false
		}
	}

	nameRaw := (*args)["name"]
	namespaceRaw := (*args)["namespace"]
	if nameRaw != nil && namespaceRaw != nil {
		matchFn = func(configMap K8sSecret) bool {
			name, _ := configMap.Name()
			namespace, _ := configMap.Namespace()
			if name == nameRaw.(string) && namespace == namespaceRaw.(string) {
				return true
			}
			return false
		}
	}

	for i := range secrets {
		configMap := secrets[i].(K8sSecret)
		if matchFn(configMap) {
			return nil, configMap, nil
		}
	}

	return args, nil, nil
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

	return certificatesToLumiCertificates(k.MotorRuntime, certs)
}

func (k *lumiK8sPodsecuritypolicy) id() (string, error) {
	return k.Uid()
}

func (k *lumiK8sPodsecuritypolicy) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sPodsecuritypolicy) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sConfigmap) id() (string, error) {
	return k.Uid()
}

func (p *lumiK8sConfigmap) init(args *lumi.Args) (*lumi.Args, K8sConfigmap, error) {
	// pass-through if all args are already provided
	if len(*args) == 0 || len(*args) > 2 {
		return args, nil, nil
	}

	// search for existing resources if uid or name/namespace is provided
	obj, err := p.MotorRuntime.CreateResource("k8s")
	if err != nil {
		return nil, nil, err
	}
	k8sResource := obj.(K8s)

	configMaps, err := k8sResource.Configmaps()
	if err != nil {
		return nil, nil, err
	}

	var matchFn func(configMap K8sConfigmap) bool

	uidRaw := (*args)["uid"]
	if uidRaw != nil {
		matchFn = func(configMap K8sConfigmap) bool {
			uid, _ := configMap.Uid()
			if uid == uidRaw.(string) {
				return true
			}
			return false
		}
	}

	nameRaw := (*args)["name"]
	namespaceRaw := (*args)["namespace"]
	if nameRaw != nil && namespaceRaw != nil {
		matchFn = func(configMap K8sConfigmap) bool {
			name, _ := configMap.Name()
			namespace, _ := configMap.Namespace()
			if name == nameRaw.(string) && namespace == namespaceRaw.(string) {
				return true
			}
			return false
		}
	}

	for i := range configMaps {
		configMap := configMaps[i].(K8sConfigmap)
		if matchFn(configMap) {
			return nil, configMap, nil
		}
	}

	return args, nil, nil
}

func (k *lumiK8sConfigmap) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sConfigmap) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sService) id() (string, error) {
	return k.Uid()
}

func (p *lumiK8sService) init(args *lumi.Args) (*lumi.Args, K8sService, error) {
	// pass-through if all args are already provided
	if len(*args) == 0 || len(*args) > 2 {
		return args, nil, nil
	}

	// search for existing resources if uid or name/namespace is provided
	obj, err := p.MotorRuntime.CreateResource("k8s")
	if err != nil {
		return nil, nil, err
	}
	k8sResource := obj.(K8s)

	services, err := k8sResource.Services()
	if err != nil {
		return nil, nil, err
	}

	var matchFn func(entry K8sService) bool

	uidRaw := (*args)["uid"]
	if uidRaw != nil {
		matchFn = func(service K8sService) bool {
			uid, _ := service.Uid()
			if uid == uidRaw.(string) {
				return true
			}
			return false
		}
	}

	nameRaw := (*args)["name"]
	namespaceRaw := (*args)["namespace"]
	if nameRaw != nil && namespaceRaw != nil {
		matchFn = func(entry K8sService) bool {
			name, _ := entry.Name()
			namespace, _ := entry.Namespace()
			if name == nameRaw.(string) && namespace == namespaceRaw.(string) {
				return true
			}
			return false
		}
	}

	for i := range services {
		service := services[i].(K8sService)
		if matchFn(service) {
			return nil, service, nil
		}
	}

	return args, nil, nil
}

func (k *lumiK8sService) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sService) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sNetworkpolicy) id() (string, error) {
	return k.Uid()
}

func (p *lumiK8sNetworkpolicy) init(args *lumi.Args) (*lumi.Args, K8sNetworkpolicy, error) {
	// pass-through if all args are already provided
	if len(*args) == 0 || len(*args) > 2 {
		return args, nil, nil
	}

	// search for existing resources if uid or name/namespace is provided
	obj, err := p.MotorRuntime.CreateResource("k8s")
	if err != nil {
		return nil, nil, err
	}
	k8sResource := obj.(K8s)

	policies, err := k8sResource.NetworkPolicies()
	if err != nil {
		return nil, nil, err
	}

	var matchFn func(entry K8sNetworkpolicy) bool

	uidRaw := (*args)["uid"]
	if uidRaw != nil {
		matchFn = func(service K8sNetworkpolicy) bool {
			uid, _ := service.Uid()
			if uid == uidRaw.(string) {
				return true
			}
			return false
		}
	}

	nameRaw := (*args)["name"]
	namespaceRaw := (*args)["namespace"]
	if nameRaw != nil && namespaceRaw != nil {
		matchFn = func(entry K8sNetworkpolicy) bool {
			name, _ := entry.Name()
			namespace, _ := entry.Namespace()
			if name == nameRaw.(string) && namespace == namespaceRaw.(string) {
				return true
			}
			return false
		}
	}

	for i := range policies {
		policy := policies[i].(K8sService)
		if matchFn(policy) {
			return nil, policy, nil
		}
	}

	return args, nil, nil
}

func (k *lumiK8sNetworkpolicy) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sNetworkpolicy) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sServiceaccount) id() (string, error) {
	return k.Uid()
}

func (p *lumiK8sServiceaccount) init(args *lumi.Args) (*lumi.Args, K8sServiceaccount, error) {
	// pass-through if all args are already provided
	if len(*args) == 0 || len(*args) > 2 {
		return args, nil, nil
	}

	// search for existing resources if uid or name/namespace is provided
	obj, err := p.MotorRuntime.CreateResource("k8s")
	if err != nil {
		return nil, nil, err
	}
	k8sResource := obj.(K8s)

	serviceAccounts, err := k8sResource.Serviceaccounts()
	if err != nil {
		return nil, nil, err
	}

	var matchFn func(entry K8sServiceaccount) bool

	uidRaw := (*args)["uid"]
	if uidRaw != nil {
		matchFn = func(service K8sServiceaccount) bool {
			uid, _ := service.Uid()
			if uid == uidRaw.(string) {
				return true
			}
			return false
		}
	}

	nameRaw := (*args)["name"]
	namespaceRaw := (*args)["namespace"]
	if nameRaw != nil && namespaceRaw != nil {
		matchFn = func(entry K8sServiceaccount) bool {
			name, _ := entry.Name()
			namespace, _ := entry.Namespace()
			if name == nameRaw.(string) && namespace == namespaceRaw.(string) {
				return true
			}
			return false
		}
	}

	for i := range serviceAccounts {
		entry := serviceAccounts[i].(K8sServiceaccount)
		if matchFn(entry) {
			return nil, entry, nil
		}
	}

	return args, nil, nil
}

func (k *lumiK8sServiceaccount) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sServiceaccount) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sRbacClusterrole) id() (string, error) {
	return k.Uid()
}

func (p *lumiK8sRbacClusterrole) init(args *lumi.Args) (*lumi.Args, K8sRbacClusterrole, error) {
	// pass-through if all args are already provided
	if len(*args) == 0 || len(*args) > 2 {
		return args, nil, nil
	}

	// search for existing resources if uid or name/namespace is provided
	obj, err := p.MotorRuntime.CreateResource("k8s")
	if err != nil {
		return nil, nil, err
	}
	k8sResource := obj.(K8s)

	clusterRoles, err := k8sResource.Clusterroles()
	if err != nil {
		return nil, nil, err
	}

	var matchFn func(entry K8sRbacClusterrole) bool

	uidRaw := (*args)["uid"]
	if uidRaw != nil {
		matchFn = func(service K8sRbacClusterrole) bool {
			uid, _ := service.Uid()
			if uid == uidRaw.(string) {
				return true
			}
			return false
		}
	}

	nameRaw := (*args)["name"]
	if nameRaw != nil {
		matchFn = func(entry K8sRbacClusterrole) bool {
			name, _ := entry.Name()
			if name == nameRaw.(string) {
				return true
			}
			return false
		}
	}

	for i := range clusterRoles {
		entry := clusterRoles[i].(K8sRbacClusterrole)
		if matchFn(entry) {
			return nil, entry, nil
		}
	}

	return args, nil, nil
}

func (k *lumiK8sRbacClusterrole) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sRbacClusterrole) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sRbacRole) id() (string, error) {
	return k.Uid()
}

func (p *lumiK8sRbacRole) init(args *lumi.Args) (*lumi.Args, K8sRbacRole, error) {
	// pass-through if all args are already provided
	if len(*args) == 0 || len(*args) > 2 {
		return args, nil, nil
	}

	// search for existing resources if uid or name/namespace is provided
	obj, err := p.MotorRuntime.CreateResource("k8s")
	if err != nil {
		return nil, nil, err
	}
	k8sResource := obj.(K8s)

	roles, err := k8sResource.Roles()
	if err != nil {
		return nil, nil, err
	}

	var matchFn func(entry K8sRbacRole) bool

	uidRaw := (*args)["uid"]
	if uidRaw != nil {
		matchFn = func(service K8sRbacRole) bool {
			uid, _ := service.Uid()
			if uid == uidRaw.(string) {
				return true
			}
			return false
		}
	}

	nameRaw := (*args)["name"]
	namespaceRaw := (*args)["namespace"]
	if nameRaw != nil && namespaceRaw != nil {
		matchFn = func(entry K8sRbacRole) bool {
			name, _ := entry.Name()
			namespace, _ := entry.Namespace()
			if name == nameRaw.(string) && namespace == namespaceRaw.(string) {
				return true
			}
			return false
		}
	}

	for i := range roles {
		entry := roles[i].(K8sRbacRole)
		if matchFn(entry) {
			return nil, entry, nil
		}
	}

	return args, nil, nil
}

func (k *lumiK8sRbacRole) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sRbacRole) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sRbacClusterrolebinding) id() (string, error) {
	return k.Uid()
}

func (p *lumiK8sRbacClusterrolebinding) init(args *lumi.Args) (*lumi.Args, K8sRbacClusterrolebinding, error) {
	// pass-through if all args are already provided
	if len(*args) == 0 || len(*args) > 2 {
		return args, nil, nil
	}

	// search for existing resources if uid or name/namespace is provided
	obj, err := p.MotorRuntime.CreateResource("k8s")
	if err != nil {
		return nil, nil, err
	}
	k8sResource := obj.(K8s)

	roleBindings, err := k8sResource.Clusterrolebindings()
	if err != nil {
		return nil, nil, err
	}

	var matchFn func(entry K8sRbacClusterrolebinding) bool

	uidRaw := (*args)["uid"]
	if uidRaw != nil {
		matchFn = func(service K8sRbacClusterrolebinding) bool {
			uid, _ := service.Uid()
			if uid == uidRaw.(string) {
				return true
			}
			return false
		}
	}

	nameRaw := (*args)["name"]
	if nameRaw != nil {
		matchFn = func(entry K8sRbacClusterrolebinding) bool {
			name, _ := entry.Name()
			if name == nameRaw.(string) {
				return true
			}
			return false
		}
	}

	for i := range roleBindings {
		entry := roleBindings[i].(K8sRbacClusterrolebinding)
		if matchFn(entry) {
			return nil, entry, nil
		}
	}

	return args, nil, nil
}

func (k *lumiK8sRbacClusterrolebinding) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sRbacClusterrolebinding) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sRbacRolebinding) id() (string, error) {
	return k.Uid()
}

func (p *lumiK8sRbacRolebinding) init(args *lumi.Args) (*lumi.Args, K8sRbacRolebinding, error) {
	// pass-through if all args are already provided
	if len(*args) == 0 || len(*args) > 2 {
		return args, nil, nil
	}

	// search for existing resources if uid or name/namespace is provided
	obj, err := p.MotorRuntime.CreateResource("k8s")
	if err != nil {
		return nil, nil, err
	}
	k8sResource := obj.(K8s)

	roleBindings, err := k8sResource.Rolebindings()
	if err != nil {
		return nil, nil, err
	}

	var matchFn func(entry K8sRbacRolebinding) bool

	uidRaw := (*args)["uid"]
	if uidRaw != nil {
		matchFn = func(service K8sRbacRolebinding) bool {
			uid, _ := service.Uid()
			if uid == uidRaw.(string) {
				return true
			}
			return false
		}
	}

	nameRaw := (*args)["name"]
	namespaceRaw := (*args)["namespace"]
	if nameRaw != nil && namespaceRaw != nil {
		matchFn = func(entry K8sRbacRolebinding) bool {
			name, _ := entry.Name()
			namespace, _ := entry.Namespace()
			if name == nameRaw.(string) && namespace == namespaceRaw.(string) {
				return true
			}
			return false
		}
	}

	for i := range roleBindings {
		entry := roleBindings[i].(K8sRbacRolebinding)
		if matchFn(entry) {
			return nil, entry, nil
		}
	}

	return args, nil, nil
}

func (k *lumiK8sRbacRolebinding) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sRbacRolebinding) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func getPlatformIdentifierElements(transport providers.Transport) (string, string, string, error) {
	kt, err := k8stransport(transport)
	if err != nil {
		return "", "", "", err
	}

	identifier, err := kt.PlatformIdentifier()
	if err != nil {
		return "", "", "", err
	}

	var identifierUid string
	var identifierName string
	var identifierNamespace string
	splitIdentifier := strings.Split(identifier, "/")
	arrayLength := len(splitIdentifier)
	if arrayLength >= 1 {
		identifierUid = splitIdentifier[arrayLength-1]
	}
	if arrayLength >= 3 {
		identifierName = splitIdentifier[arrayLength-3]
	}
	if arrayLength >= 6 {
		identifierNamespace = splitIdentifier[arrayLength-6]
	}

	return identifierUid, identifierName, identifierNamespace, nil
}
