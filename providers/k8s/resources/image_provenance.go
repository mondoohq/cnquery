// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

// This file wires image-provenance rollups onto every workload kind, the
// supply-chain view of which images a workload runs:
//
//   usesImageDigest()  every container image is pinned by digest (immutable)
//   usesLatestTag()    any container image uses the mutable "latest" tag
//                      (explicitly or implicitly, when no tag is set)
//   imageRegistries()  the distinct registry domains the images come from
//
// Only regular and init containers are considered; ephemeral (debug) containers
// are transient and commonly run ad-hoc "latest" tooling, so counting them
// would create noise. Images that fail to parse are treated as unpinned (they
// cannot satisfy usesImageDigest) and contribute no registry.

import (
	"github.com/distribution/reference"
	corev1 "k8s.io/api/core/v1"
)

type parsedImageRef struct {
	registry string
	digested bool
	latest   bool
	ok       bool
}

func parseImageRef(image string) parsedImageRef {
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return parsedImageRef{}
	}
	ref := parsedImageRef{registry: reference.Domain(named), ok: true}
	if _, isDigested := named.(reference.Digested); isDigested {
		ref.digested = true
		return ref
	}
	// No digest: resolve the tag, defaulting to "latest" when none is set.
	if tagged, ok := reference.TagNameOnly(named).(reference.Tagged); ok {
		ref.latest = tagged.Tag() == "latest"
	}
	return ref
}

func imageContainers(spec *corev1.PodSpec) []corev1.Container {
	if spec == nil {
		return nil
	}
	out := make([]corev1.Container, 0, len(spec.Containers)+len(spec.InitContainers))
	out = append(out, spec.Containers...)
	out = append(out, spec.InitContainers...)
	return out
}

func specUsesImageDigest(spec *corev1.PodSpec) bool {
	containers := imageContainers(spec)
	if len(containers) == 0 {
		return false
	}
	for _, c := range containers {
		ref := parseImageRef(c.Image)
		if !ref.ok || !ref.digested {
			return false
		}
	}
	return true
}

func specUsesLatestTag(spec *corev1.PodSpec) bool {
	for _, c := range imageContainers(spec) {
		if ref := parseImageRef(c.Image); ref.ok && ref.latest {
			return true
		}
	}
	return false
}

func specImageRegistries(spec *corev1.PodSpec) []any {
	seen := map[string]struct{}{}
	out := []any{}
	for _, c := range imageContainers(spec) {
		ref := parseImageRef(c.Image)
		if !ref.ok || ref.registry == "" {
			continue
		}
		if _, ok := seen[ref.registry]; ok {
			continue
		}
		seen[ref.registry] = struct{}{}
		out = append(out, ref.registry)
	}
	return out
}

// ---- per-kind wiring ----

func (k *mqlK8sPod) usesImageDigest() (bool, error) {
	spec, err := k.podSpecTyped()
	return boolFromSpec(spec, err, specUsesImageDigest)
}

func (k *mqlK8sPod) usesLatestTag() (bool, error) {
	spec, err := k.podSpecTyped()
	return boolFromSpec(spec, err, specUsesLatestTag)
}

func (k *mqlK8sPod) imageRegistries() ([]any, error) {
	spec, err := k.podSpecTyped()
	return stringsFromSpec(spec, err, specImageRegistries)
}

func (k *mqlK8sDeployment) usesImageDigest() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesImageDigest)
}

func (k *mqlK8sDeployment) usesLatestTag() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesLatestTag)
}

func (k *mqlK8sDeployment) imageRegistries() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specImageRegistries)
}

func (k *mqlK8sDaemonset) usesImageDigest() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesImageDigest)
}

func (k *mqlK8sDaemonset) usesLatestTag() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesLatestTag)
}

func (k *mqlK8sDaemonset) imageRegistries() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specImageRegistries)
}

func (k *mqlK8sStatefulset) usesImageDigest() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesImageDigest)
}

func (k *mqlK8sStatefulset) usesLatestTag() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesLatestTag)
}

func (k *mqlK8sStatefulset) imageRegistries() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specImageRegistries)
}

func (k *mqlK8sReplicaset) usesImageDigest() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesImageDigest)
}

func (k *mqlK8sReplicaset) usesLatestTag() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesLatestTag)
}

func (k *mqlK8sReplicaset) imageRegistries() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specImageRegistries)
}

func (k *mqlK8sJob) usesImageDigest() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesImageDigest)
}

func (k *mqlK8sJob) usesLatestTag() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesLatestTag)
}

func (k *mqlK8sJob) imageRegistries() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specImageRegistries)
}

func (k *mqlK8sCronjob) usesImageDigest() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesImageDigest)
}

func (k *mqlK8sCronjob) usesLatestTag() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesLatestTag)
}

func (k *mqlK8sCronjob) imageRegistries() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specImageRegistries)
}
