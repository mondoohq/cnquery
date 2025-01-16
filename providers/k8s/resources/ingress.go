// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"sync"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/types"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sIngressInternal struct {
	lock  sync.Mutex
	obj   *networkingv1.Ingress
	objId string
}

func (k *mqlK8s) ingresses() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, gvkString(networkingv1.SchemeGroupVersion.WithKind("ingresses")), getNamespaceScope(k.MqlRuntime), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		ingress, ok := resource.(*networkingv1.Ingress)
		if !ok {
			return nil, errors.New("not a k8s ingress")
		}

		objId := objIdFromK8sObj(obj, objT)

		rules, err := buildRules(ingress, objId, k.MqlRuntime)
		if err != nil {
			return nil, err
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.ingress", map[string]*llx.RawData{
			"id":              llx.StringData(objId),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"namespace":       llx.StringData(obj.GetNamespace()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
			"rules":           llx.ArrayData(rules, types.Resource("k8s.ingressrule")),
		})
		if err != nil {
			return nil, err
		}
		r.(*mqlK8sIngress).obj = ingress
		r.(*mqlK8sIngress).objId = objId
		return r, nil
	})
}

func (k *mqlK8sIngress) tls() ([]interface{}, error) {
	o, err := CreateResource(k.MqlRuntime, "k8s", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	k8s := o.(*mqlK8s)

	tls, err := getTLS(k.obj, k.objId, k.MqlRuntime, k8s.GetSecrets)
	if err != nil {
		return nil, err
	}

	return tls, nil
}

func (k *mqlK8sIngress) manifest() (map[string]interface{}, error) {
	manifest, err := convert.JsonToDict(k.obj)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (k *mqlK8sIngress) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sIngress(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sIngress](runtime, args, func(k *mqlK8s) *plugin.TValue[[]interface{}] { return k.GetIngresses() })
}

func (k *mqlK8sIngress) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sIngress) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}

func buildRules(ingress *networkingv1.Ingress, objId string, runtime *plugin.Runtime) ([]interface{}, error) {
	k8sIngressRules := []interface{}{}

	for i, rule := range ingress.Spec.Rules {
		paths := []interface{}{}
		ruleId := fmt.Sprintf("%s/rule%d", objId, i)

		if rule.HTTP != nil {
			for i, path := range rule.HTTP.Paths {
				pathId := fmt.Sprintf("%s/path%d", ruleId, i)
				ingresshttprulepath, err := buildIngressHttpRulePaths(path, pathId, runtime)
				if err != nil {
					return nil, err
				}
				paths = append(paths, ingresshttprulepath)
			}
		}

		ingressRule, err := CreateResource(runtime, "k8s.ingressrule", map[string]*llx.RawData{
			"id":        llx.StringData(ruleId),
			"host":      llx.StringData(rule.Host),
			"httpPaths": llx.ArrayData(paths, types.Resource("k8s.ingresshttprulepath")),
		})
		if err != nil {
			return nil, fmt.Errorf("error creating k8s.ingressrule: %s", err)
		}

		k8sIngressRules = append(k8sIngressRules, ingressRule)
	}

	return k8sIngressRules, nil
}

func buildIngressHttpRulePaths(path networkingv1.HTTPIngressPath, id string, runtime *plugin.Runtime) (plugin.Resource, error) {
	pathType := ""

	if path.PathType != nil {
		pathType = string(*path.PathType)
	}

	ingressbackend, err := buildIngressBackend(path.Backend, id, runtime)
	if err != nil {
		return nil, err
	}

	ingresshttprulepath, err := CreateResource(runtime, "k8s.ingresshttprulepath", map[string]*llx.RawData{
		"id":       llx.StringData(id),
		"path":     llx.StringData(path.Path),
		"pathType": llx.StringData(pathType),
		"backend":  llx.ResourceData(ingressbackend, "k8s.ingressbackend"),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating k8s.ingresshttprulepath: %s", err)
	}

	return ingresshttprulepath, nil
}

func buildIngressBackend(networkingIngressBackend networkingv1.IngressBackend, id string, runtime *plugin.Runtime) (plugin.Resource, error) {
	ingressservicebackend, err := buildIngressServiceBackend(networkingIngressBackend.Service, id, runtime)
	if err != nil {
		return nil, err
	}

	ingressresourceref, err := buildIngressResourceRefBackend(networkingIngressBackend.Resource, id, runtime)
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
	ingressbackend, err := CreateResource(runtime, "k8s.ingressbackend", map[string]*llx.RawData{
		"id":          llx.StringData(backendId),
		"service":     llx.ResourceData(ingressservicebackend, "k8s.ingressservicebackend"),
		"resourceRef": llx.ResourceData(ingressresourceref, "k8s.ingressresourceref"),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating k8s.ingressbackend: %s", err)
	}

	return ingressbackend, nil
}

func buildIngressServiceBackend(networkingIngressServiceBackend *networkingv1.IngressServiceBackend, id string, runtime *plugin.Runtime) (plugin.Resource, error) {
	ingressServiceBackendName := ""
	portName := ""
	var portNumber int64
	if networkingIngressServiceBackend != nil {
		ingressServiceBackendName = networkingIngressServiceBackend.Name
		portName = networkingIngressServiceBackend.Port.Name
		portNumber = int64(networkingIngressServiceBackend.Port.Number)
	}

	svcId := fmt.Sprintf("%s/%s-%s-%d", id, ingressServiceBackendName, portName, portNumber)
	ingressservicebackend, err := CreateResource(runtime, "k8s.ingressservicebackend", map[string]*llx.RawData{
		"id":         llx.StringData(svcId),
		"name":       llx.StringData(ingressServiceBackendName),
		"portName":   llx.StringData(portName),
		"portNumber": llx.IntData(portNumber),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating k8s.ingresservicebackend: %s", err)
	}
	return ingressservicebackend, nil
}

func buildIngressResourceRefBackend(corev1ResourceRef *corev1.TypedLocalObjectReference, id string, runtime *plugin.Runtime) (plugin.Resource, error) {
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
	ingressresourceref, err := CreateResource(runtime, "k8s.ingressresourceref", map[string]*llx.RawData{
		"id":       llx.StringData(resRefId),
		"apiGroup": llx.StringData(apiGroup),
		"kind":     llx.StringData(kind),
		"name":     llx.StringData(name),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating k8s.ingressresourceref: %s", err)
	}
	return ingressresourceref, nil
}

func (k *mqlK8sIngressrule) id() (string, error) {
	return k.Id.Data, nil
}

func (k *mqlK8sIngresshttprulepath) id() (string, error) {
	return k.Id.Data, nil
}

func (k *mqlK8sIngressbackend) id() (string, error) {
	return k.Id.Data, nil
}

func (k *mqlK8sIngressservicebackend) id() (string, error) {
	return k.Id.Data, nil
}

func (k *mqlK8sIngressresourceref) id() (string, error) {
	return k.Id.Data, nil
}

func (k *mqlK8sIngresstls) id() (string, error) {
	return k.Id.Data, nil
}

func getTLS(ingress *networkingv1.Ingress, objId string, runtime *plugin.Runtime, getSecrets func() *plugin.TValue[[]interface{}]) ([]interface{}, error) {
	tlsData := []interface{}{}
	if len(ingress.Spec.TLS) > 0 {
		// This returns ALL Secrets found in the cluster!
		secretsInterface := getSecrets()
		if secretsInterface.Error != nil {
			return nil, fmt.Errorf("failed to fetch Secrets referenced in Ingress: %s", secretsInterface.Error)
		}

		// Build up a map of Secrets found in the same Namespace as this Ingress resource
		secrets := map[string]*mqlK8sSecret{}
		for _, secInterface := range secretsInterface.Data {
			secret, ok := secInterface.(*mqlK8sSecret)
			if !ok {
				return nil, errors.New("returned list of Secrets failed type assertion")
			}

			if ingress.Namespace != secret.Namespace.Data {
				continue
			}

			secrets[secret.Name.Data] = secret
		}

		// There is the potential for no Secret to be found or that a Secret
		// is found (can happen when scanning static manifest files or simply an Ingress
		// which references a non-existent Secret) but either improperly-formatted
		// or simply not containing TLS data. In either event just keep trying to
		// process as much as we can.
		for i, tls := range ingress.Spec.TLS {
			secret, ok := secrets[tls.SecretName]
			if !ok {
				continue
			}

			certs := secret.GetCertificates()
			if certs.Error != nil {
				return nil, errors.New("error getting certificate data from Secret")
			}
			if certs.Data == nil || len(certs.Data) == 0 {
				// no TLS data in Secret referenced
				// k8s will allow this, so we'll just follow along with this being
				// a non-critical issue and skip processing the Secret
				continue
			}

			ingressTls, err := CreateResource(runtime, "k8s.ingresstls", map[string]*llx.RawData{
				"id":           llx.StringData(fmt.Sprintf("%s-tls%d", objId, i)),
				"hosts":        llx.ArrayData(convert.SliceAnyToInterface(tls.Hosts), types.String),
				"certificates": llx.ArrayData(secret.Certificates.Data, types.Resource("network.certificate")),
			})
			if err != nil {
				return nil, fmt.Errorf("error creating k8s.ingresstls: %s", err)
			}

			tlsData = append(tlsData, ingressTls)
		}
	}

	return tlsData, nil
}
