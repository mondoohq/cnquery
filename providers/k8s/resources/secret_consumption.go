// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

// This file wires Secret-consumption rollups onto every workload kind, the
// "what secrets can a compromised pod reach?" view that complements the
// security rollups. k8s.secret.usedBy() already answers the reverse (which pods
// reference a Secret); these answer it from the workload side:
//
//   usesSecretsAsEnv()    a Secret is injected as an environment variable
//   mountsSecretVolumes() a Secret is mounted as a volume
//   consumedSecrets()     names of every Secret the workload references
//
// Environment-variable injection is called out separately because it is the
// higher-risk path: env values leak into process listings, crash dumps, and
// child processes, where a mounted file does not.

import (
	corev1 "k8s.io/api/core/v1"
)

func specUsesSecretsAsEnv(spec *corev1.PodSpec) bool {
	for _, c := range allWorkloadContainers(spec) {
		for _, e := range c.Env {
			if e.ValueFrom != nil && e.ValueFrom.SecretKeyRef != nil {
				return true
			}
		}
		for _, ef := range c.EnvFrom {
			if ef.SecretRef != nil {
				return true
			}
		}
	}
	return false
}

func specMountsSecretVolumes(spec *corev1.PodSpec) bool {
	if spec == nil {
		return false
	}
	for i := range spec.Volumes {
		v := spec.Volumes[i]
		if v.Secret != nil {
			return true
		}
		if v.Projected != nil {
			for _, src := range v.Projected.Sources {
				if src.Secret != nil {
					return true
				}
			}
		}
	}
	return false
}

// specConsumedSecrets returns the deduplicated names of every Secret the
// workload references: via environment variables, mounted (or projected)
// volumes, and image-pull secrets.
func specConsumedSecrets(spec *corev1.PodSpec) []any {
	seen := map[string]struct{}{}
	out := []any{}
	add := func(name string) {
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}

	for _, c := range allWorkloadContainers(spec) {
		for _, e := range c.Env {
			if e.ValueFrom != nil && e.ValueFrom.SecretKeyRef != nil {
				add(e.ValueFrom.SecretKeyRef.Name)
			}
		}
		for _, ef := range c.EnvFrom {
			if ef.SecretRef != nil {
				add(ef.SecretRef.Name)
			}
		}
	}

	if spec != nil {
		for i := range spec.Volumes {
			v := spec.Volumes[i]
			if v.Secret != nil {
				add(v.Secret.SecretName)
			}
			if v.Projected != nil {
				for _, src := range v.Projected.Sources {
					if src.Secret != nil {
						add(src.Secret.Name)
					}
				}
			}
		}
		for _, ips := range spec.ImagePullSecrets {
			add(ips.Name)
		}
	}

	return out
}

// ---- per-kind wiring ----

func (k *mqlK8sPod) usesSecretsAsEnv() (bool, error) {
	spec, err := k.podSpecTyped()
	return boolFromSpec(spec, err, specUsesSecretsAsEnv)
}

func (k *mqlK8sPod) mountsSecretVolumes() (bool, error) {
	spec, err := k.podSpecTyped()
	return boolFromSpec(spec, err, specMountsSecretVolumes)
}

func (k *mqlK8sPod) consumedSecrets() ([]any, error) {
	spec, err := k.podSpecTyped()
	return stringsFromSpec(spec, err, specConsumedSecrets)
}

func (k *mqlK8sDeployment) usesSecretsAsEnv() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesSecretsAsEnv)
}

func (k *mqlK8sDeployment) mountsSecretVolumes() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specMountsSecretVolumes)
}

func (k *mqlK8sDeployment) consumedSecrets() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specConsumedSecrets)
}

func (k *mqlK8sDaemonset) usesSecretsAsEnv() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesSecretsAsEnv)
}

func (k *mqlK8sDaemonset) mountsSecretVolumes() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specMountsSecretVolumes)
}

func (k *mqlK8sDaemonset) consumedSecrets() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specConsumedSecrets)
}

func (k *mqlK8sStatefulset) usesSecretsAsEnv() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesSecretsAsEnv)
}

func (k *mqlK8sStatefulset) mountsSecretVolumes() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specMountsSecretVolumes)
}

func (k *mqlK8sStatefulset) consumedSecrets() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specConsumedSecrets)
}

func (k *mqlK8sReplicaset) usesSecretsAsEnv() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesSecretsAsEnv)
}

func (k *mqlK8sReplicaset) mountsSecretVolumes() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specMountsSecretVolumes)
}

func (k *mqlK8sReplicaset) consumedSecrets() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specConsumedSecrets)
}

func (k *mqlK8sJob) usesSecretsAsEnv() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesSecretsAsEnv)
}

func (k *mqlK8sJob) mountsSecretVolumes() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specMountsSecretVolumes)
}

func (k *mqlK8sJob) consumedSecrets() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specConsumedSecrets)
}

func (k *mqlK8sCronjob) usesSecretsAsEnv() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesSecretsAsEnv)
}

func (k *mqlK8sCronjob) mountsSecretVolumes() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specMountsSecretVolumes)
}

func (k *mqlK8sCronjob) consumedSecrets() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specConsumedSecrets)
}
