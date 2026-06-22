// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
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
