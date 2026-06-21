// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

// This file wires container resource-constraint rollup predicates onto every
// workload kind. Unbounded containers are a denial-of-service / noisy-neighbor
// risk, so these complement the security rollups in workload_security.go with
// an availability view:
//
//   hasCpuLimit / hasMemoryLimit  every container sets that limit
//   hasResourceLimits             every container sets both CPU and memory limits
//   hasResourceRequests           every container sets both CPU and memory requests
//
// Only regular and init containers are considered. Ephemeral containers are
// excluded because they are transient debug aids, not part of the steady-state
// workload (and the Kubernetes API disallows setting resources on them anyway),
// so counting them would make every workload with a debug container report false.

import (
	corev1 "k8s.io/api/core/v1"
)

func resourceContainers(spec *corev1.PodSpec) []corev1.Container {
	if spec == nil {
		return nil
	}
	out := make([]corev1.Container, 0, len(spec.Containers)+len(spec.InitContainers))
	out = append(out, spec.Containers...)
	out = append(out, spec.InitContainers...)
	return out
}

// resourceQuantitySet reports whether the resource list sets a non-zero quantity
// for the named resource.
func resourceQuantitySet(list corev1.ResourceList, name corev1.ResourceName) bool {
	q, ok := list[name]
	return ok && !q.IsZero()
}

// specAllContainersHave reports whether every resource-bearing container sets
// the named quantity in its limits (when limits is true) or requests list.
func specAllContainersHave(spec *corev1.PodSpec, limits bool, name corev1.ResourceName) bool {
	containers := resourceContainers(spec)
	// A pod with no regular or init containers is treated as unconstrained.
	if len(containers) == 0 {
		return false
	}
	for _, c := range containers {
		list := c.Resources.Requests
		if limits {
			list = c.Resources.Limits
		}
		if !resourceQuantitySet(list, name) {
			return false
		}
	}
	return true
}

func specHasCPULimit(spec *corev1.PodSpec) bool {
	return specAllContainersHave(spec, true, corev1.ResourceCPU)
}

func specHasMemoryLimit(spec *corev1.PodSpec) bool {
	return specAllContainersHave(spec, true, corev1.ResourceMemory)
}

func specHasResourceLimits(spec *corev1.PodSpec) bool {
	return specHasCPULimit(spec) && specHasMemoryLimit(spec)
}

func specHasResourceRequests(spec *corev1.PodSpec) bool {
	return specAllContainersHave(spec, false, corev1.ResourceCPU) &&
		specAllContainersHave(spec, false, corev1.ResourceMemory)
}

// ---- per-kind wiring ----

func (k *mqlK8sPod) hasCpuLimit() (bool, error) {
	spec, err := k.podSpecTyped()
	return boolFromSpec(spec, err, specHasCPULimit)
}

func (k *mqlK8sPod) hasMemoryLimit() (bool, error) {
	spec, err := k.podSpecTyped()
	return boolFromSpec(spec, err, specHasMemoryLimit)
}

func (k *mqlK8sPod) hasResourceLimits() (bool, error) {
	spec, err := k.podSpecTyped()
	return boolFromSpec(spec, err, specHasResourceLimits)
}

func (k *mqlK8sPod) hasResourceRequests() (bool, error) {
	spec, err := k.podSpecTyped()
	return boolFromSpec(spec, err, specHasResourceRequests)
}

func (k *mqlK8sDeployment) hasCpuLimit() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasCPULimit)
}

func (k *mqlK8sDeployment) hasMemoryLimit() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasMemoryLimit)
}

func (k *mqlK8sDeployment) hasResourceLimits() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasResourceLimits)
}

func (k *mqlK8sDeployment) hasResourceRequests() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasResourceRequests)
}

func (k *mqlK8sDaemonset) hasCpuLimit() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasCPULimit)
}

func (k *mqlK8sDaemonset) hasMemoryLimit() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasMemoryLimit)
}

func (k *mqlK8sDaemonset) hasResourceLimits() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasResourceLimits)
}

func (k *mqlK8sDaemonset) hasResourceRequests() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasResourceRequests)
}

func (k *mqlK8sStatefulset) hasCpuLimit() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasCPULimit)
}

func (k *mqlK8sStatefulset) hasMemoryLimit() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasMemoryLimit)
}

func (k *mqlK8sStatefulset) hasResourceLimits() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasResourceLimits)
}

func (k *mqlK8sStatefulset) hasResourceRequests() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasResourceRequests)
}

func (k *mqlK8sReplicaset) hasCpuLimit() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasCPULimit)
}

func (k *mqlK8sReplicaset) hasMemoryLimit() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasMemoryLimit)
}

func (k *mqlK8sReplicaset) hasResourceLimits() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasResourceLimits)
}

func (k *mqlK8sReplicaset) hasResourceRequests() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasResourceRequests)
}

func (k *mqlK8sJob) hasCpuLimit() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasCPULimit)
}

func (k *mqlK8sJob) hasMemoryLimit() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasMemoryLimit)
}

func (k *mqlK8sJob) hasResourceLimits() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasResourceLimits)
}

func (k *mqlK8sJob) hasResourceRequests() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasResourceRequests)
}

func (k *mqlK8sCronjob) hasCpuLimit() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasCPULimit)
}

func (k *mqlK8sCronjob) hasMemoryLimit() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasMemoryLimit)
}

func (k *mqlK8sCronjob) hasResourceLimits() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasResourceLimits)
}

func (k *mqlK8sCronjob) hasResourceRequests() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specHasResourceRequests)
}
