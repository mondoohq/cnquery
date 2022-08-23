package k8s

import (
	"bytes"
	"errors"

	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/core"
	"go.mondoo.io/mondoo/resources/packs/core/certificates"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *mqlK8s) GetSecrets() ([]interface{}, error) {
	return k8sResourceToMql(k.MotorRuntime, "secrets.v1.", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
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
		r.MqlResource().Cache.Store("_resource", &resources.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *mqlK8sSecret) id() (string, error) {
	return k.Id()
}

func (p *mqlK8sSecret) init(args *resources.Args) (*resources.Args, K8sSecret, error) {
	return initNamespacedResource[K8sSecret](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Secrets() })
}

func (k *mqlK8sSecret) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.MqlResource())
}

func (k *mqlK8sSecret) GetLabels() (interface{}, error) {
	return k8sLabels(k.MqlResource())
}

func (k *mqlK8sSecret) GetCertificates() (interface{}, error) {
	entry, ok := k.MqlResource().Cache.Load("_resource")
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

	return core.CertificatesToMqlCertificates(k.MotorRuntime, certs)
}
