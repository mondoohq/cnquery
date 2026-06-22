// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

// This file exposes runtime truth from a pod's status, as opposed to what its
// spec declares. The kubelet records the image it actually resolved and pulled
// in status.containerStatuses[].imageID, which carries a digest even when the
// spec used a mutable tag — letting us report what is really running and detect
// drift between the (mutable) spec reference and the running image.
//
// These live on k8s.pod because status is a property of the running object;
// controllers describe desired state and carry no container statuses.

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// imageDigest extracts the "sha256:..." digest from a status imageID, which may
// take forms like "docker-pullable://repo@sha256:..", "repo@sha256:..", or a
// bare "sha256:..". Returns "" when no digest is present.
func imageDigest(imageID string) string {
	if i := strings.Index(imageID, "@sha256:"); i >= 0 {
		return imageID[i+1:]
	}
	if i := strings.Index(imageID, "sha256:"); i >= 0 {
		return imageID[i:]
	}
	return ""
}

// runtimeContainerStatuses returns the regular and init container statuses, the
// steady-state containers (ephemeral debug containers are excluded, matching the
// image-provenance scope).
func runtimeContainerStatuses(pod *corev1.Pod) []corev1.ContainerStatus {
	out := make([]corev1.ContainerStatus, 0, len(pod.Status.ContainerStatuses)+len(pod.Status.InitContainerStatuses))
	out = append(out, pod.Status.ContainerStatuses...)
	out = append(out, pod.Status.InitContainerStatuses...)
	return out
}

func (k *mqlK8sPod) runningImageDigests() ([]any, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	out := []any{}
	for _, cs := range runtimeContainerStatuses(pod) {
		d := imageDigest(cs.ImageID)
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	return out, nil
}

// hasImageDigestDrift reports whether any container runs an unpinned image: the
// spec references a mutable tag (not a digest) yet the kubelet resolved it to a
// concrete digest. A restart or reschedule could silently pull a different image
// for that tag. Digest-pinned spec references can never drift.
func (k *mqlK8sPod) hasImageDigestDrift() (bool, error) {
	pod, err := k.getPod()
	if err != nil {
		return false, err
	}

	runningDigest := map[string]string{}
	for _, cs := range runtimeContainerStatuses(pod) {
		runningDigest[cs.Name] = imageDigest(cs.ImageID)
	}

	specContainers := make([]corev1.Container, 0, len(pod.Spec.Containers)+len(pod.Spec.InitContainers))
	specContainers = append(specContainers, pod.Spec.Containers...)
	specContainers = append(specContainers, pod.Spec.InitContainers...)

	for _, c := range specContainers {
		if ref := parseImageRef(c.Image); ref.ok && ref.digested {
			continue
		}
		if runningDigest[c.Name] != "" {
			return true, nil
		}
	}
	return false, nil
}
