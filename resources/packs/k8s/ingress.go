package k8s

import (
	"errors"
	"fmt"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *mqlK8s) GetIngresses() ([]interface{}, error) {
	return k8sResourceToMql(k.MotorRuntime, "ingresses", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		ingress, ok := resource.(*networkingv1.Ingress)
		if !ok {
			return nil, errors.New("not a k8s ingress")
		}

		objId := objIdFromK8sObj(obj, objT)

		rules, err := buildRules(ingress, objId, k.MotorRuntime)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.ingress",
			"id", objId,
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
		r.MqlResource().Cache.Store("_resource", &resources.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *mqlK8sIngress) id() (string, error) {
	return k.Id()
}

func (p *mqlK8sIngress) init(args *resources.Args) (*resources.Args, K8sIngress, error) {
	return initNamespacedResource[K8sIngress](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Ingresses() })
}

func (k *mqlK8sIngress) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.MqlResource())
}

func (k *mqlK8sIngress) GetLabels() (interface{}, error) {
	return k8sLabels(k.MqlResource())
}

func buildRules(ingress *networkingv1.Ingress, objId string, motorRuntime *resources.Runtime) ([]interface{}, error) {
	k8sIngressRules := []interface{}{}

	for i, rule := range ingress.Spec.Rules {
		paths := []interface{}{}
		ruleId := fmt.Sprintf("%s/rule%d", objId, i)

		if rule.HTTP != nil {
			for i, path := range rule.HTTP.Paths {
				pathId := fmt.Sprintf("%s/path%d", ruleId, i)
				ingresshttprulepath, err := buildIngressHttpRulePaths(path, pathId, motorRuntime)
				if err != nil {
					return nil, err
				}
				paths = append(paths, ingresshttprulepath)
			}
		}

		ingressRule, err := motorRuntime.CreateResource("k8s.ingressrule",
			"id", ruleId,
			"host", rule.Host,
			"httpPaths", paths,
		)
		if err != nil {
			return nil, fmt.Errorf("error creating k8s.ingressrule: %s", err)
		}

		k8sIngressRules = append(k8sIngressRules, ingressRule)
	}

	return k8sIngressRules, nil
}

func buildIngressHttpRulePaths(path networkingv1.HTTPIngressPath, id string, motorRuntime *resources.Runtime) (resources.ResourceType, error) {
	pathType := ""

	if path.PathType != nil {
		pathType = string(*path.PathType)
	}

	ingressbackend, err := buildIngressBackend(path.Backend, id, motorRuntime)
	if err != nil {
		return nil, err
	}

	ingresshttprulepath, err := motorRuntime.CreateResource("k8s.ingresshttprulepath",
		"id", id,
		"path", path.Path,
		"pathType", pathType,
		"backend", ingressbackend,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating k8s.ingresshttprulepath: %s", err)
	}

	return ingresshttprulepath, nil
}

func buildIngressBackend(networkingIngressBackend networkingv1.IngressBackend, id string, motorRuntime *resources.Runtime) (resources.ResourceType, error) {
	ingressservicebackend, err := buildIngressServiceBackend(networkingIngressBackend.Service, id, motorRuntime)
	if err != nil {
		return nil, err
	}

	ingressresourceref, err := buildIngressResourceRefBackend(networkingIngressBackend.Resource, id, motorRuntime)
	if err != nil {
		return nil, err
	}

	backendId := id
	if networkingIngressBackend.Service != nil {
		backendId = backendId + "/service"
	}

	if networkingIngressBackend.Resource != nil {
		backendId = backendId + "/resourceRef"
	}

	ingressbackend, err := motorRuntime.CreateResource("k8s.ingressbackend",
		"id", backendId,
		"service", ingressservicebackend,
		"resourceRef", ingressresourceref,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating k8s.ingressbackend: %s", err)
	}

	return ingressbackend, nil
}

func buildIngressServiceBackend(networkingIngressServiceBackend *networkingv1.IngressServiceBackend, id string, motorRuntime *resources.Runtime) (resources.ResourceType, error) {
	ingressServiceBackendName := ""
	portName := ""
	var portNumber int64
	if networkingIngressServiceBackend != nil {
		ingressServiceBackendName = networkingIngressServiceBackend.Name
		portName = networkingIngressServiceBackend.Port.Name
		portNumber = int64(networkingIngressServiceBackend.Port.Number)
	}

	svcId := fmt.Sprintf("%s/%s-%s-%d", id, ingressServiceBackendName, portName, portNumber)
	ingressservicebackend, err := motorRuntime.CreateResource("k8s.ingressservicebackend",
		"id", svcId,
		"name", ingressServiceBackendName,
		"portName", portName,
		"portNumber", portNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating k8s.ingresservicebackend: %s", err)
	}
	return ingressservicebackend, nil
}

func buildIngressResourceRefBackend(corev1ResourceRef *corev1.TypedLocalObjectReference, id string, motorRuntime *resources.Runtime) (resources.ResourceType, error) {
	apiGroup := ""
	kind := ""
	name := ""
	if corev1ResourceRef != nil {
		if corev1ResourceRef.APIGroup != nil {
			apiGroup = *corev1ResourceRef.APIGroup
		}
		kind = corev1ResourceRef.Kind
		name = corev1ResourceRef.Name
	}

	resRefId := fmt.Sprintf("%s/%s-%s-%s", id, apiGroup, kind, name)
	ingressresourceref, err := motorRuntime.CreateResource("k8s.ingressresourceref",
		"id", resRefId,
		"apiGroup", apiGroup,
		"kind", kind,
		"name", name,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating k8s.ingressresourceref: %s", err)
	}
	return ingressresourceref, nil
}

func (k *mqlK8sIngressrule) id() (string, error) {
	return k.Id()
}

func (k *mqlK8sIngresshttprulepath) id() (string, error) {
	return k.Id()
}

func (k *mqlK8sIngressbackend) id() (string, error) {
	return k.Id()
}

func (k *mqlK8sIngressservicebackend) id() (string, error) {
	return k.Id()
}

func (k *mqlK8sIngressresourceref) id() (string, error) {
	return k.Id()
}
