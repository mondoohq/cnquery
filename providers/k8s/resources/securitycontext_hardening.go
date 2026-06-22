// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	corev1 "k8s.io/api/core/v1"
)

// The Pod Security Standards baseline helpers in pod_security_standards.go
// (specBaselineAppArmor, specBaselineProcMount, specBaselineSysctls) report
// compliance. These expose the inverse as standalone predicates so a workload
// can be audited for a single hardening dimension without computing the full
// podSecurityStandard level.

// specUsesUnconfinedAppArmor reports whether any container (or the pod) runs
// with an explicitly Unconfined AppArmor profile.
func specUsesUnconfinedAppArmor(spec *corev1.PodSpec) bool {
	return !specBaselineAppArmor(spec)
}

// specUsesUnmaskedProcMount reports whether any container sets procMount to a
// value other than Default (i.e. Unmasked), which exposes masked /proc paths.
func specUsesUnmaskedProcMount(spec *corev1.PodSpec) bool {
	return !specBaselineProcMount(spec)
}

// specHasUnsafeSysctls reports whether the pod requests any sysctl outside the
// kubelet's safe set.
func specHasUnsafeSysctls(spec *corev1.PodSpec) bool {
	return !specBaselineSysctls(spec)
}

func (k *mqlK8sPod) usesUnconfinedAppArmor() (bool, error) {
	spec, err := k.podSpecTyped()
	return boolFromSpec(spec, err, specUsesUnconfinedAppArmor)
}

func (k *mqlK8sPod) usesUnmaskedProcMount() (bool, error) {
	spec, err := k.podSpecTyped()
	return boolFromSpec(spec, err, specUsesUnmaskedProcMount)
}

func (k *mqlK8sPod) hasUnsafeSysctls() (bool, error) {
	spec, err := k.podSpecTyped()
	return boolFromSpec(spec, err, specHasUnsafeSysctls)
}

func (k *mqlK8sDeployment) usesUnconfinedAppArmor() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesUnconfinedAppArmor)
}

func (k *mqlK8sDeployment) usesUnmaskedProcMount() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesUnmaskedProcMount)
}

func (k *mqlK8sDeployment) hasUnsafeSysctls() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasUnsafeSysctls)
}

func (k *mqlK8sDaemonset) usesUnconfinedAppArmor() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesUnconfinedAppArmor)
}

func (k *mqlK8sDaemonset) usesUnmaskedProcMount() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesUnmaskedProcMount)
}

func (k *mqlK8sDaemonset) hasUnsafeSysctls() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasUnsafeSysctls)
}

func (k *mqlK8sStatefulset) usesUnconfinedAppArmor() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesUnconfinedAppArmor)
}

func (k *mqlK8sStatefulset) usesUnmaskedProcMount() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesUnmaskedProcMount)
}

func (k *mqlK8sStatefulset) hasUnsafeSysctls() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasUnsafeSysctls)
}

func (k *mqlK8sReplicaset) usesUnconfinedAppArmor() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesUnconfinedAppArmor)
}

func (k *mqlK8sReplicaset) usesUnmaskedProcMount() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesUnmaskedProcMount)
}

func (k *mqlK8sReplicaset) hasUnsafeSysctls() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasUnsafeSysctls)
}

func (k *mqlK8sJob) usesUnconfinedAppArmor() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesUnconfinedAppArmor)
}

func (k *mqlK8sJob) usesUnmaskedProcMount() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesUnmaskedProcMount)
}

func (k *mqlK8sJob) hasUnsafeSysctls() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasUnsafeSysctls)
}

func (k *mqlK8sCronjob) usesUnconfinedAppArmor() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesUnconfinedAppArmor)
}

func (k *mqlK8sCronjob) usesUnmaskedProcMount() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesUnmaskedProcMount)
}

func (k *mqlK8sCronjob) hasUnsafeSysctls() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasUnsafeSysctls)
}
