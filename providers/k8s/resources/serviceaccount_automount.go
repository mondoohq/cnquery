// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

// Resolving whether an API token is mounted needs both the pod spec and the
// ServiceAccount it runs as. Kubernetes applies the pod-level setting when it
// is present, otherwise the ServiceAccount's, otherwise its default of true.
//
// Unlike imagePullSecrets, which the ServiceAccount admission controller copies
// into the pod spec, the automount decision is never written back to the spec:
// the controller simply omits the projected token volume. Reading the spec
// alone therefore reports "token mounted" for every workload hardened by
// setting automountServiceAccountToken: false on the ServiceAccount, which is
// the recommended way to disable it for a whole namespace.

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	corev1 "k8s.io/api/core/v1"
)

// automountFromSpecAndAccount applies the Kubernetes precedence rule: the
// pod-level setting wins, then the ServiceAccount's, and an unset value at both
// levels means the default, true.
func automountFromSpecAndAccount(podLevel, accountLevel *bool) bool {
	if podLevel != nil {
		return *podLevel
	}
	if accountLevel != nil {
		return *accountLevel
	}
	return true
}

// podServiceAccountName returns the ServiceAccount a pod spec runs as, falling
// back to the deprecated field and then to "default", which is what Kubernetes
// assigns when the spec names none.
func podServiceAccountName(spec *corev1.PodSpec) string {
	if spec == nil {
		return "default"
	}
	if spec.ServiceAccountName != "" {
		return spec.ServiceAccountName
	}
	if spec.DeprecatedServiceAccount != "" {
		return spec.DeprecatedServiceAccount
	}
	return "default"
}

// serviceAccountAutomount returns the named ServiceAccount's automount setting,
// or nil when the account cannot be resolved. It reads the already-fetched
// ServiceAccount collection rather than looking each account up individually,
// so it costs no extra API calls.
func serviceAccountAutomount(runtime *plugin.Runtime, namespace, name string) *bool {
	index, err := namespacedByName[*mqlK8sServiceaccount](
		runtime, func(x *mqlK8s) *plugin.TValue[[]any] { return x.GetServiceaccounts() })
	if err != nil {
		// A scan without permission to list ServiceAccounts cannot tell what the
		// account asks for, so the Kubernetes default applies.
		log.Debug().Err(err).Msg("cannot list service accounts to resolve automountServiceAccountToken")
		return nil
	}
	sa, ok := index[namespace+"/"+name]
	if !ok || sa == nil {
		return nil
	}
	v := sa.GetAutomountServiceAccountToken()
	if v.Error != nil {
		log.Debug().Err(v.Error).Str("serviceaccount", name).Msg("cannot read automountServiceAccountToken")
		return nil
	}
	value := v.Data
	return &value
}

// effectiveAutomountServiceAccountToken reports whether Kubernetes mounts an API
// token for a workload running under the given pod spec.
func effectiveAutomountServiceAccountToken(runtime *plugin.Runtime, spec *corev1.PodSpec, namespace string) bool {
	var podLevel *bool
	if spec != nil {
		podLevel = spec.AutomountServiceAccountToken
	}
	if podLevel != nil {
		return *podLevel
	}
	return automountFromSpecAndAccount(nil, serviceAccountAutomount(runtime, namespace, podServiceAccountName(spec)))
}
