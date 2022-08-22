package os

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/providers"
	k8s_provider "go.mondoo.io/mondoo/motor/providers/k8s"
	"go.mondoo.io/mondoo/motor/providers/k8s/resources"
	"go.mondoo.io/mondoo/resources/packs/core"
	"go.mondoo.io/mondoo/resources/packs/core/certificates"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacauthorizationv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func k8sProvider(t providers.Transport) (k8s_provider.KubernetesProvider, error) {
	at, ok := t.(k8s_provider.KubernetesProvider)
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
	return core.StrMapToInterface(objM.GetAnnotations()), nil
}

func k8sLabels(lumiResource *lumi.Resource) (interface{}, error) {
	objM, err := k8sMetaObject(lumiResource)
	if err != nil {
		return nil, err
	}
	return core.StrMapToInterface(objM.GetLabels()), nil
}

func (k *lumiK8s) id() (string, error) {
	return "k8s", nil
}

func (k *lumiK8s) GetServerVersion() (interface{}, error) {
	kt, err := k8sProvider(k.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(kt.ServerVersion())
}

func (k *lumiK8s) GetApiResources() ([]interface{}, error) {
	kt, err := k8sProvider(k.MotorRuntime.Motor.Provider)
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
			"shortNames", core.StrSliceToInterface(entry.Resource.ShortNames),
			"categories", core.StrSliceToInterface(entry.Resource.Categories),
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
	kt, err := k8sProvider(r.Motor.Provider)
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
			"id", objIdFromK8sObj(obj, objT),
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		return k.MotorRuntime.CreateResource("k8s.namespace",
			"id", objIdFromK8sObj(obj, objT),
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		podSpec, err := resources.GetPodSpec(resource)
		if err != nil {
			return nil, err
		}

		podSpecDict, err := core.JsonToDict(podSpec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.pod",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"labels", core.StrMapToInterface(obj.GetLabels()),
			"annotations", core.StrMapToInterface(obj.GetAnnotations()),
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		podSpec, err := resources.GetPodSpec(resource)
		if err != nil {
			return nil, err
		}

		podSpecDict, err := core.JsonToDict(podSpec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.deployment",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"podSpec", podSpecDict,
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		podSpec, err := resources.GetPodSpec(resource)
		if err != nil {
			return nil, err
		}

		podSpecDict, err := core.JsonToDict(podSpec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.daemonset",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"podSpec", podSpecDict,
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		podSpec, err := resources.GetPodSpec(resource)
		if err != nil {
			return nil, err
		}

		podSpecDict, err := core.JsonToDict(podSpec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.statefulset",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"podSpec", podSpecDict,
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		podSpec, err := resources.GetPodSpec(resource)
		if err != nil {
			return nil, err
		}

		podSpecDict, err := core.JsonToDict(podSpec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.replicaset",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"podSpec", podSpecDict,
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		podSpec, err := resources.GetPodSpec(resource)
		if err != nil {
			return nil, err
		}

		podSpecDict, err := core.JsonToDict(podSpec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.job",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"podSpec", podSpecDict,
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		podSpec, err := resources.GetPodSpec(resource)
		if err != nil {
			return nil, err
		}

		podSpecDict, err := core.JsonToDict(podSpec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.cronjob",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"podSpec", podSpecDict,
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		s, ok := resource.(*corev1.Secret)
		if !ok {
			return nil, errors.New("not a k8s secret")
		}

		r, err := k.MotorRuntime.CreateResource("k8s.secret",
			"id", objIdFromK8sObj(obj, objT),
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		psp, ok := resource.(*policyv1beta1.PodSecurityPolicy)
		if !ok {
			return nil, errors.New("not a k8s podsecuritypolicy")
		}

		spec, err := core.JsonToDict(psp.Spec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.podsecuritypolicy",
			"id", objIdFromK8sObj(obj, objT),
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		srv, ok := resource.(*corev1.Service)
		if !ok {
			return nil, errors.New("not a k8s service")
		}

		spec, err := core.JsonToDict(srv.Spec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.service",
			"id", objIdFromK8sObj(obj, objT),
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		cm, ok := resource.(*corev1.ConfigMap)
		if !ok {
			return nil, errors.New("not a k8s configmap")
		}

		r, err := k.MotorRuntime.CreateResource("k8s.configmap",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"data", core.StrMapToInterface(cm.Data),
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		networkPolicies, ok := resource.(*networkingv1.NetworkPolicy)
		if !ok {
			return nil, errors.New("not a k8s networkpolicy")
		}

		spec, err := core.JsonToDict(networkPolicies.Spec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.networkpolicy",
			"id", objIdFromK8sObj(obj, objT),
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		serviceAccount, ok := resource.(*corev1.ServiceAccount)
		if !ok {
			return nil, errors.New("not a k8s serviceaccount")
		}

		secrets, err := core.JsonToDictSlice(serviceAccount.Secrets)
		if err != nil {
			return nil, err
		}

		imagePullSecrets, err := core.JsonToDictSlice(serviceAccount.ImagePullSecrets)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.serviceaccount",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"secrets", secrets,
			"imagePullSecrets", imagePullSecrets,
			"automountServiceAccountToken", core.ToBool(serviceAccount.AutomountServiceAccountToken),
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		clusterRole, ok := resource.(*rbacauthorizationv1.ClusterRole)
		if !ok {
			return nil, errors.New("not a k8s clusterrole")
		}

		rules, err := core.JsonToDictSlice(clusterRole.Rules)
		if err != nil {
			return nil, err
		}

		aggregationRule, err := core.JsonToDict(clusterRole.AggregationRule)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.rbac.clusterrole",
			"id", objIdFromK8sObj(obj, objT),
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		clusterRole, ok := resource.(*rbacauthorizationv1.Role)
		if !ok {
			return nil, errors.New("not a k8s role")
		}

		rules, err := core.JsonToDictSlice(clusterRole.Rules)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.rbac.role",
			"id", objIdFromK8sObj(obj, objT),
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		clusterRoleBinding, ok := resource.(*rbacauthorizationv1.ClusterRoleBinding)
		if !ok {
			return nil, errors.New("not a k8s clusterrolebinding")
		}

		subjects, err := core.JsonToDictSlice(clusterRoleBinding.Subjects)
		if err != nil {
			return nil, err
		}

		roleRef, err := core.JsonToDict(clusterRoleBinding.RoleRef)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.rbac.clusterrolebinding",
			"id", objIdFromK8sObj(obj, objT),
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

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		roleBinding, ok := resource.(*rbacauthorizationv1.RoleBinding)
		if !ok {
			return nil, errors.New("not a k8s rolebinding")
		}

		subjects, err := core.JsonToDictSlice(roleBinding.Subjects)
		if err != nil {
			return nil, err
		}

		roleRef, err := core.JsonToDict(roleBinding.RoleRef)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.rbac.rolebinding",
			"id", objIdFromK8sObj(obj, objT),
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
	kt, err := k8sProvider(k.MotorRuntime.Motor.Provider)
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

			manifest, err := core.JsonToDict(resource)
			if err != nil {
				log.Error().Err(err).Msg("couldn't convert resource to json dict")
				return nil, err
			}

			r, err := k.MotorRuntime.CreateResource("k8s.customresource",
				"id", objIdFromK8sObj(obj, objT),
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
	return k.Id()
}

func (k *lumiK8sNode) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sNode) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sNamespace) id() (string, error) {
	return k.Id()
}

func (k *lumiK8sCustomresource) id() (string, error) {
	return k.Id()
}

func (k *lumiK8sCustomresource) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sCustomresource) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sPod) id() (string, error) {
	return k.Id()
}

func (p *lumiK8sPod) init(args *lumi.Args) (*lumi.Args, K8sPod, error) {
	return initNamespacedResource[K8sPod](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Pods() })
}

func (k *lumiK8sPod) GetInitContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, InitContainerType)
}

func (k *lumiK8sPod) GetContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, ContainerContainerType)
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
	rawSpec, err := k.PodSpec()
	if err != nil {
		return nil, err
	}

	podSpec, ok := rawSpec.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid pod spec information")
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
	return k.Id()
}

func (p *lumiK8sDeployment) init(args *lumi.Args) (*lumi.Args, K8sDeployment, error) {
	return initNamespacedResource[K8sDeployment](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Deployments() })
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

func (k *lumiK8sDeployment) GetInitContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, InitContainerType)
}

func (k *lumiK8sDeployment) GetContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, ContainerContainerType)
}

func (k *lumiK8sDaemonset) id() (string, error) {
	return k.Id()
}

func (p *lumiK8sDaemonset) init(args *lumi.Args) (*lumi.Args, K8sDaemonset, error) {
	return initNamespacedResource[K8sDaemonset](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Daemonsets() })
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

func (k *lumiK8sDaemonset) GetInitContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, InitContainerType)
}

func (k *lumiK8sDaemonset) GetContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, ContainerContainerType)
}

func (k *lumiK8sStatefulset) id() (string, error) {
	return k.Id()
}

func (p *lumiK8sStatefulset) init(args *lumi.Args) (*lumi.Args, K8sStatefulset, error) {
	return initNamespacedResource[K8sStatefulset](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Statefulsets() })
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

func (k *lumiK8sStatefulset) GetInitContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, InitContainerType)
}

func (k *lumiK8sStatefulset) GetContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, ContainerContainerType)
}

func (k *lumiK8sReplicaset) id() (string, error) {
	return k.Id()
}

func (p *lumiK8sReplicaset) init(args *lumi.Args) (*lumi.Args, K8sReplicaset, error) {
	return initNamespacedResource[K8sReplicaset](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Replicasets() })
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

func (k *lumiK8sReplicaset) GetInitContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, InitContainerType)
}

func (k *lumiK8sReplicaset) GetContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, ContainerContainerType)
}

func (k *lumiK8sJob) id() (string, error) {
	return k.Id()
}

func (p *lumiK8sJob) init(args *lumi.Args) (*lumi.Args, K8sJob, error) {
	return initNamespacedResource[K8sJob](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Jobs() })
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

func (k *lumiK8sJob) GetInitContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, InitContainerType)
}

func (k *lumiK8sJob) GetContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, ContainerContainerType)
}

func (k *lumiK8sCronjob) id() (string, error) {
	return k.Id()
}

func (p *lumiK8sCronjob) init(args *lumi.Args) (*lumi.Args, K8sCronjob, error) {
	return initNamespacedResource[K8sCronjob](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Cronjobs() })
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

func (k *lumiK8sCronjob) GetInitContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, InitContainerType)
}

func (k *lumiK8sCronjob) GetContainers() ([]interface{}, error) {
	return getContainers(k, k.MotorRuntime, ContainerContainerType)
}

func (k *lumiK8sSecret) id() (string, error) {
	return k.Id()
}

func (p *lumiK8sSecret) init(args *lumi.Args) (*lumi.Args, K8sSecret, error) {
	return initNamespacedResource[K8sSecret](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Secrets() })
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

	return core.CertificatesToLumiCertificates(k.MotorRuntime, certs)
}

func (k *lumiK8sPodsecuritypolicy) id() (string, error) {
	return k.Id()
}

func (k *lumiK8sPodsecuritypolicy) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sPodsecuritypolicy) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sConfigmap) id() (string, error) {
	return k.Id()
}

func (p *lumiK8sConfigmap) init(args *lumi.Args) (*lumi.Args, K8sConfigmap, error) {
	return initNamespacedResource[K8sConfigmap](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Configmaps() })
}

func (k *lumiK8sConfigmap) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sConfigmap) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sService) id() (string, error) {
	return k.Id()
}

func (p *lumiK8sService) init(args *lumi.Args) (*lumi.Args, K8sService, error) {
	return initNamespacedResource[K8sService](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Services() })
}

func (k *lumiK8sService) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sService) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sNetworkpolicy) id() (string, error) {
	return k.Id()
}

func (p *lumiK8sNetworkpolicy) init(args *lumi.Args) (*lumi.Args, K8sNetworkpolicy, error) {
	return initNamespacedResource[K8sNetworkpolicy](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.NetworkPolicies() })
}

func (k *lumiK8sNetworkpolicy) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sNetworkpolicy) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sServiceaccount) id() (string, error) {
	return k.Id()
}

func (p *lumiK8sServiceaccount) init(args *lumi.Args) (*lumi.Args, K8sServiceaccount, error) {
	return initNamespacedResource[K8sServiceaccount](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Serviceaccounts() })
}

func (k *lumiK8sServiceaccount) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sServiceaccount) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sRbacClusterrole) id() (string, error) {
	return k.Id()
}

func (p *lumiK8sRbacClusterrole) init(args *lumi.Args) (*lumi.Args, K8sRbacClusterrole, error) {
	return initResource[K8sRbacClusterrole](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Clusterroles() })
}

func (k *lumiK8sRbacClusterrole) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sRbacClusterrole) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sRbacRole) id() (string, error) {
	return k.Id()
}

func (p *lumiK8sRbacRole) init(args *lumi.Args) (*lumi.Args, K8sRbacRole, error) {
	return initNamespacedResource[K8sRbacRole](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Roles() })
}

func (k *lumiK8sRbacRole) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sRbacRole) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sRbacClusterrolebinding) id() (string, error) {
	return k.Id()
}

func (p *lumiK8sRbacClusterrolebinding) init(args *lumi.Args) (*lumi.Args, K8sRbacClusterrolebinding, error) {
	return initResource[K8sRbacClusterrolebinding](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Clusterrolebindings() })
}

func (k *lumiK8sRbacClusterrolebinding) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sRbacClusterrolebinding) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func (k *lumiK8sRbacRolebinding) id() (string, error) {
	return k.Id()
}

func (p *lumiK8sRbacRolebinding) init(args *lumi.Args) (*lumi.Args, K8sRbacRolebinding, error) {
	return initNamespacedResource[K8sRbacRolebinding](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Rolebindings() })
}

func (k *lumiK8sRbacRolebinding) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.LumiResource())
}

func (k *lumiK8sRbacRolebinding) GetLabels() (interface{}, error) {
	return k8sLabels(k.LumiResource())
}

func getPlatformIdentifierElements(transport providers.Transport) (string, string, error) {
	kt, err := k8sProvider(transport)
	if err != nil {
		return "", "", err
	}

	identifier, err := kt.PlatformIdentifier()
	if err != nil {
		return "", "", err
	}

	var identifierName string
	var identifierNamespace string
	splitIdentifier := strings.Split(identifier, "/")
	arrayLength := len(splitIdentifier)
	if arrayLength >= 1 {
		identifierName = splitIdentifier[arrayLength-1]
	}
	if arrayLength >= 4 {
		identifierNamespace = splitIdentifier[arrayLength-4]
	}

	return identifierName, identifierNamespace, nil
}

type K8sNamespacedObject interface {
	K8sObject
	Namespace() (string, error)
}

type K8sObject interface {
	Id() (string, error)
	Kind() (string, error)
	Name() (string, error)
	Manifest() (interface{}, error)
}

func objId(o K8sNamespacedObject) (string, error) {
	kind, err := o.Kind()
	if err != nil {
		return "", err
	}

	name, err := o.Name()
	if err != nil {
		return "", err
	}

	namespace, err := o.Namespace()
	if err != nil {
		return "", err
	}

	return objIdFromFields(kind, namespace, name), nil
}

func objIdFromK8sObj(o metav1.Object, objT metav1.Type) string {
	return objIdFromFields(objT.GetKind(), o.GetNamespace(), o.GetName())
}

func objIdFromFields(kind, namespace, name string) string {
	// Kind is usually capitalized. Make it all lower case for readability
	return fmt.Sprintf("%s:%s:%s", strings.ToLower(kind), namespace, name)
}

func initNamespacedResource[T K8sNamespacedObject](
	args *lumi.Args, runtime *lumi.Runtime, r func(k8s K8s) ([]interface{}, error),
) (*lumi.Args, T, error) {
	// pass-through if all args are already provided
	if len(*args) > 2 {
		return args, *new(T), nil
	}

	// get platform identifier infos
	identifierName, identifierNamespace, err := getPlatformIdentifierElements(runtime.Motor.Provider)
	if err != nil {
		return args, *new(T), nil
	}

	// search for existing resources if id or name/namespace is provided
	obj, err := runtime.CreateResource("k8s")
	if err != nil {
		return args, *new(T), err
	}
	k8sResource := obj.(K8s)

	nsResources, err := r(k8sResource)
	if err != nil {
		return args, *new(T), err
	}

	var matchFn func(nsR T) bool

	var idRaw string
	if _, ok := (*args)["id"]; ok {
		idRaw = (*args)["id"].(string)
	}

	if idRaw != "" {
		matchFn = func(nsR T) bool {
			id, _ := nsR.Id()
			return id == idRaw
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
	if nameRaw == "" {
		nameRaw = identifierName
		namespaceRaw = identifierNamespace
	}
	if nameRaw != "" {
		matchFn = func(nsR T) bool {
			name, _ := nsR.Name()
			namespace, _ := nsR.Namespace()
			return name == nameRaw && namespace == namespaceRaw
		}
	}

	for i := range nsResources {
		nsR := nsResources[i].(T)
		if matchFn(nsR) {
			return args, nsR, nil
		}
	}

	return args, *new(T), fmt.Errorf("not found")
}

func initResource[T K8sObject](
	args *lumi.Args, runtime *lumi.Runtime, r func(k8s K8s) ([]interface{}, error),
) (*lumi.Args, T, error) {
	// pass-through if all args are already provided
	if len(*args) > 1 {
		return args, *new(T), nil
	}

	// get platform identifier infos
	identifierName, _, err := getPlatformIdentifierElements(runtime.Motor.Provider)
	if err != nil {
		return args, *new(T), nil
	}

	// search for existing resources if id or name is provided
	obj, err := runtime.CreateResource("k8s")
	if err != nil {
		return nil, *new(T), err
	}
	k8sResource := obj.(K8s)

	resources, err := r(k8sResource)
	if err != nil {
		return nil, *new(T), err
	}

	var matchFn func(entry T) bool

	idRaw := (*args)["id"]
	if idRaw != nil {
		matchFn = func(entry T) bool {
			id, _ := entry.Id()
			if id == idRaw.(string) {
				return true
			}
			return false
		}
	}

	var nameRaw string
	if _, ok := (*args)["name"]; ok {
		nameRaw = (*args)["name"].(string)
	}
	if nameRaw == "" {
		nameRaw = identifierName
	}
	if nameRaw != "" {
		matchFn = func(nsR T) bool {
			name, _ := nsR.Name()
			return name == nameRaw
		}
	}

	for i := range resources {
		entry := resources[i].(T)
		if matchFn(entry) {
			return nil, entry, nil
		}
	}

	return nil, *new(T), fmt.Errorf("not found")
}

type ContainerType string

var (
	InitContainerType      ContainerType = "init"
	ContainerContainerType ContainerType = "container"
)

func getContainers(
	o K8sNamespacedObject, lumiRuntime *lumi.Runtime, containerType ContainerType,
) ([]interface{}, error) {
	var containersFunc func(runtime.Object) ([]corev1.Container, error)
	resourceType := ""
	switch containerType {
	case InitContainerType:
		containersFunc = resources.GetInitContainers
		resourceType = "k8s.initContainer"
	case ContainerContainerType:
		containersFunc = resources.GetContainers
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
	obj := resources.ConvertToK8sObject(unstr)

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

		args := []interface{}{
			"uid", id + "/" + c.Name, // container names are unique within a resource
			"name", c.Name,
			"imageName", c.Image,
			"image", c.Image, // deprecated, will be replaced with the containerImage going forward
			"command", core.StrSliceToInterface(c.Command),
			"args", core.StrSliceToInterface(c.Args),
			"resources", resources,
			"volumeMounts", volumeMounts,
			"volumeDevices", volumeDevices,
			"imagePullPolicy", string(c.ImagePullPolicy),
			"securityContext", secContext,
			"workingDir", c.WorkingDir,
			"tty", c.TTY,
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

		lumiContainer, err := lumiRuntime.CreateResource(resourceType, args...)
		if err != nil {
			return nil, err
		}
		resp = append(resp, lumiContainer)
	}
	return resp, nil
}
