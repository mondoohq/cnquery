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

// effectivePullPolicy returns the image pull policy that applies to a container,
// resolving the Kubernetes default when the field is unset: Always for a
// "latest" (or untagged) image, IfNotPresent otherwise. An unparseable image
// reference defaults to IfNotPresent so it is never mistaken for an Always pull.
func effectivePullPolicy(c corev1.Container) corev1.PullPolicy {
	if c.ImagePullPolicy != "" {
		return c.ImagePullPolicy
	}
	if ref := parseImageRef(c.Image); ref.ok && ref.latest {
		return corev1.PullAlways
	}
	return corev1.PullIfNotPresent
}

// specUsesAlwaysImagePullPolicy reports whether every container pulls its image
// on each start. Always forces the kubelet to contact the registry (re-pulling
// and re-authorizing) rather than trusting a node-cached layer.
func specUsesAlwaysImagePullPolicy(spec *corev1.PodSpec) bool {
	containers := imageContainers(spec)
	if len(containers) == 0 {
		return false
	}
	for _, c := range containers {
		if effectivePullPolicy(c) != corev1.PullAlways {
			return false
		}
	}
	return true
}

// specRisksStaleImage reports whether any container can run a node-cached image
// that no longer matches its reference: a mutable (non-digest) tag combined with
// a pull policy other than Always. The node may serve a stale or attacker-seeded
// image for that tag. Digest-pinned images are always exact and never at risk.
func specRisksStaleImage(spec *corev1.PodSpec) bool {
	for _, c := range imageContainers(spec) {
		ref := parseImageRef(c.Image)
		if ref.ok && ref.digested {
			continue
		}
		if effectivePullPolicy(c) != corev1.PullAlways {
			return true
		}
	}
	return false
}

// specContainerImages returns the deduplicated image references of all regular
// and init containers.
func specContainerImages(spec *corev1.PodSpec) []any {
	return dedupImages(spec, func(parsedImageRef) bool { return true })
}

// specUnpinnedImages returns the image references that are not pinned by digest.
func specUnpinnedImages(spec *corev1.PodSpec) []any {
	return dedupImages(spec, func(ref parsedImageRef) bool { return !(ref.ok && ref.digested) })
}

func dedupImages(spec *corev1.PodSpec, include func(parsedImageRef) bool) []any {
	seen := map[string]struct{}{}
	out := []any{}
	for _, c := range imageContainers(spec) {
		if c.Image == "" || !include(parseImageRef(c.Image)) {
			continue
		}
		if _, ok := seen[c.Image]; ok {
			continue
		}
		seen[c.Image] = struct{}{}
		out = append(out, c.Image)
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

// ---- pull-policy and image-list per-kind wiring ----

func (k *mqlK8sPod) usesAlwaysImagePullPolicy() (bool, error) {
	spec, err := k.podSpecTyped()
	return boolFromSpec(spec, err, specUsesAlwaysImagePullPolicy)
}

func (k *mqlK8sPod) risksStaleImage() (bool, error) {
	spec, err := k.podSpecTyped()
	return boolFromSpec(spec, err, specRisksStaleImage)
}

func (k *mqlK8sPod) containerImages() ([]any, error) {
	spec, err := k.podSpecTyped()
	return stringsFromSpec(spec, err, specContainerImages)
}

func (k *mqlK8sPod) unpinnedImages() ([]any, error) {
	spec, err := k.podSpecTyped()
	return stringsFromSpec(spec, err, specUnpinnedImages)
}

func (k *mqlK8sDeployment) usesAlwaysImagePullPolicy() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesAlwaysImagePullPolicy)
}

func (k *mqlK8sDeployment) risksStaleImage() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specRisksStaleImage)
}

func (k *mqlK8sDeployment) containerImages() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specContainerImages)
}

func (k *mqlK8sDeployment) unpinnedImages() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specUnpinnedImages)
}

func (k *mqlK8sDaemonset) usesAlwaysImagePullPolicy() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesAlwaysImagePullPolicy)
}

func (k *mqlK8sDaemonset) risksStaleImage() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specRisksStaleImage)
}

func (k *mqlK8sDaemonset) containerImages() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specContainerImages)
}

func (k *mqlK8sDaemonset) unpinnedImages() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specUnpinnedImages)
}

func (k *mqlK8sStatefulset) usesAlwaysImagePullPolicy() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesAlwaysImagePullPolicy)
}

func (k *mqlK8sStatefulset) risksStaleImage() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specRisksStaleImage)
}

func (k *mqlK8sStatefulset) containerImages() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specContainerImages)
}

func (k *mqlK8sStatefulset) unpinnedImages() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specUnpinnedImages)
}

func (k *mqlK8sReplicaset) usesAlwaysImagePullPolicy() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesAlwaysImagePullPolicy)
}

func (k *mqlK8sReplicaset) risksStaleImage() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specRisksStaleImage)
}

func (k *mqlK8sReplicaset) containerImages() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specContainerImages)
}

func (k *mqlK8sReplicaset) unpinnedImages() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specUnpinnedImages)
}

func (k *mqlK8sJob) usesAlwaysImagePullPolicy() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesAlwaysImagePullPolicy)
}

func (k *mqlK8sJob) risksStaleImage() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specRisksStaleImage)
}

func (k *mqlK8sJob) containerImages() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specContainerImages)
}

func (k *mqlK8sJob) unpinnedImages() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specUnpinnedImages)
}

func (k *mqlK8sCronjob) usesAlwaysImagePullPolicy() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specUsesAlwaysImagePullPolicy)
}

func (k *mqlK8sCronjob) risksStaleImage() (bool, error) {
	spec, err := k.securitySpec()
	return boolFromSpec(spec, err, specRisksStaleImage)
}

func (k *mqlK8sCronjob) containerImages() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specContainerImages)
}

func (k *mqlK8sCronjob) unpinnedImages() ([]any, error) {
	spec, err := k.securitySpec()
	return stringsFromSpec(spec, err, specUnpinnedImages)
}
