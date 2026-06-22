// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/x509"
	"encoding/pem"
	"time"

	corev1 "k8s.io/api/core/v1"
)

func (k *mqlK8sSecret) isServiceAccountToken() (bool, error) {
	return k.Type.Data == string(corev1.SecretTypeServiceAccountToken), nil
}

func (k *mqlK8sSecret) isImagePullSecret() (bool, error) {
	return k.Type.Data == string(corev1.SecretTypeDockercfg) ||
		k.Type.Data == string(corev1.SecretTypeDockerConfigJson), nil
}

// isUnused reports whether nothing in the cluster references this Secret: no pod
// consumes it (usedBy is empty) and no ServiceAccount in its namespace lists it
// among its tokens or image-pull secrets. The ServiceAccount check keeps mounted
// service-account-token secrets from being flagged as orphaned.
func (k *mqlK8sSecret) isUnused() (bool, error) {
	usedBy := k.GetUsedBy()
	if usedBy.Error != nil {
		return false, usedBy.Error
	}
	if len(usedBy.Data) > 0 {
		return false, nil
	}

	cluster, err := k8sCluster(k.MqlRuntime)
	if err != nil {
		return false, err
	}
	sas := cluster.GetServiceaccounts()
	if sas.Error != nil {
		return false, sas.Error
	}

	name := k.Name.Data
	namespace := k.Namespace.Data
	for i := range sas.Data {
		sa, ok := sas.Data[i].(*mqlK8sServiceaccount)
		if !ok || sa.obj == nil || sa.obj.Namespace != namespace {
			continue
		}
		for _, ref := range sa.obj.Secrets {
			if ref.Name == name {
				return false, nil
			}
		}
		for _, ref := range sa.obj.ImagePullSecrets {
			if ref.Name == name {
				return false, nil
			}
		}
	}
	return true, nil
}

// tlsCertificates parses the certificates in the secret's tls.crt chain. It
// returns nil for non-TLS secrets and skips any PEM block that is not a
// parseable certificate rather than failing the whole query.
func (k *mqlK8sSecret) tlsCertificates() []*x509.Certificate {
	if k.obj == nil || k.obj.Type != corev1.SecretTypeTLS {
		return nil
	}
	raw, ok := k.obj.Data["tls.crt"]
	if !ok {
		return nil
	}

	var certs []*x509.Certificate
	rest := raw
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
			certs = append(certs, cert)
		}
	}
	return certs
}

func (k *mqlK8sSecret) hasExpiredCertificate() (bool, error) {
	now := time.Now()
	for _, cert := range k.tlsCertificates() {
		if now.After(cert.NotAfter) {
			return true, nil
		}
	}
	return false, nil
}

// certificateExpiry returns the earliest notAfter across the secret's
// certificates, the binding constraint for the chain. It is null when the
// secret holds no certificate.
func (k *mqlK8sSecret) certificateExpiry() (*time.Time, error) {
	certs := k.tlsCertificates()
	if len(certs) == 0 {
		return nil, nil
	}
	earliest := certs[0].NotAfter
	for _, cert := range certs[1:] {
		if cert.NotAfter.Before(earliest) {
			earliest = cert.NotAfter
		}
	}
	return &earliest, nil
}
