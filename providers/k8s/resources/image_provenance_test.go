// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestImageProvenanceRollups_Fixtures(t *testing.T) {
	k8s := workloadSecurityK8s(t)

	t.Run("digest-pinned workload", func(t *testing.T) {
		d := deploymentByName(t, k8s, "digest-pinned")
		assert.True(t, d.GetUsesImageDigest().Data, "usesImageDigest")
		assert.False(t, d.GetUsesLatestTag().Data, "usesLatestTag")
		assert.ElementsMatch(t, []any{"docker.io"}, d.GetImageRegistries().Data, "imageRegistries")
	})

	t.Run("latest-tag workload from multiple registries", func(t *testing.T) {
		d := deploymentByName(t, k8s, "latest-image")
		assert.False(t, d.GetUsesImageDigest().Data, "usesImageDigest")
		assert.True(t, d.GetUsesLatestTag().Data, "usesLatestTag")
		assert.ElementsMatch(t, []any{"docker.io", "gcr.io"}, d.GetImageRegistries().Data, "imageRegistries")
	})

	t.Run("pinned-tag workload is neither digest nor latest", func(t *testing.T) {
		d := deploymentByName(t, k8s, "hardened") // nginx:1.25
		assert.False(t, d.GetUsesImageDigest().Data, "usesImageDigest")
		assert.False(t, d.GetUsesLatestTag().Data, "usesLatestTag")
	})
}

func TestParseImageRef(t *testing.T) {
	cases := []struct {
		image    string
		registry string
		digested bool
		latest   bool
		ok       bool
	}{
		{"nginx", "docker.io", false, true, true},                                  // implicit latest
		{"nginx:latest", "docker.io", false, true, true},                           // explicit latest
		{"nginx:1.25", "docker.io", false, false, true},                            // pinned tag
		{"gcr.io/proj/app:v2", "gcr.io", false, false, true},                       // private registry, tag
		{"registry.local:5000/app:1.0", "registry.local:5000", false, false, true}, // registry with port
		{"nginx@sha256:" + zeros64, "docker.io", true, false, true},                // digest
		{"UPPER CASE !! invalid", "", false, false, false},                         // unparseable
	}
	for _, tc := range cases {
		t.Run(tc.image, func(t *testing.T) {
			ref := parseImageRef(tc.image)
			assert.Equal(t, tc.ok, ref.ok, "ok")
			assert.Equal(t, tc.registry, ref.registry, "registry")
			assert.Equal(t, tc.digested, ref.digested, "digested")
			assert.Equal(t, tc.latest, ref.latest, "latest")
		})
	}
}

const zeros64 = "0000000000000000000000000000000000000000000000000000000000000000"

func TestImageProvenanceRollups_Helpers(t *testing.T) {
	t.Run("nil and empty specs", func(t *testing.T) {
		assert.False(t, specUsesImageDigest(nil), "digest nil")
		assert.False(t, specUsesLatestTag(nil), "latest nil")
		assert.Empty(t, specImageRegistries(nil), "registries nil")
		assert.False(t, specUsesImageDigest(&corev1.PodSpec{}), "digest empty")
	})

	t.Run("digest requires every container", func(t *testing.T) {
		spec := &corev1.PodSpec{Containers: []corev1.Container{
			{Name: "a", Image: "nginx@sha256:" + zeros64},
			{Name: "b", Image: "redis:7"},
		}}
		assert.False(t, specUsesImageDigest(spec), "one tagged container breaks digest pinning")
	})

	t.Run("init container latest tag is detected", func(t *testing.T) {
		spec := &corev1.PodSpec{
			InitContainers: []corev1.Container{{Name: "init", Image: "busybox"}},
			Containers:     []corev1.Container{{Name: "a", Image: "nginx:1.25"}},
		}
		assert.True(t, specUsesLatestTag(spec), "init busybox is implicit latest")
	})
}

func TestImagePullProvenance_Fixtures(t *testing.T) {
	k8s := workloadSecurityK8s(t)

	t.Run("digest-pinned is never stale and lists no unpinned images", func(t *testing.T) {
		d := deploymentByName(t, k8s, "digest-pinned")
		assert.False(t, d.GetRisksStaleImage().Data, "risksStaleImage")
		assert.Empty(t, d.GetUnpinnedImages().Data, "unpinnedImages")
		assert.Len(t, d.GetContainerImages().Data, 1, "containerImages")
	})

	t.Run("latest images default to Always pull", func(t *testing.T) {
		d := deploymentByName(t, k8s, "latest-image")
		assert.True(t, d.GetUsesAlwaysImagePullPolicy().Data, "usesAlwaysImagePullPolicy")
		assert.False(t, d.GetRisksStaleImage().Data, "risksStaleImage (Always pull)")
		assert.ElementsMatch(t, []any{"nginx", "gcr.io/foo/bar:latest"}, d.GetUnpinnedImages().Data, "unpinnedImages")
	})

	t.Run("pinned tag with default policy risks staleness", func(t *testing.T) {
		d := deploymentByName(t, k8s, "hardened") // nginx:1.25, no explicit pull policy
		assert.True(t, d.GetRisksStaleImage().Data, "risksStaleImage")
		assert.False(t, d.GetUsesAlwaysImagePullPolicy().Data, "usesAlwaysImagePullPolicy")
		assert.ElementsMatch(t, []any{"nginx:1.25"}, d.GetUnpinnedImages().Data, "unpinnedImages")
	})
}

func TestImagePullProvenance_Helpers(t *testing.T) {
	digest := "nginx@sha256:" + zeros64

	t.Run("effective pull policy defaults", func(t *testing.T) {
		assert.Equal(t, corev1.PullAlways, effectivePullPolicy(corev1.Container{Image: "nginx"}), "implicit latest")
		assert.Equal(t, corev1.PullIfNotPresent, effectivePullPolicy(corev1.Container{Image: "nginx:1.25"}), "pinned tag")
		assert.Equal(t, corev1.PullNever, effectivePullPolicy(corev1.Container{Image: "nginx:1.25", ImagePullPolicy: corev1.PullNever}), "explicit wins")
	})

	t.Run("risksStaleImage", func(t *testing.T) {
		staleSpec := &corev1.PodSpec{Containers: []corev1.Container{{Image: "nginx:1.25", ImagePullPolicy: corev1.PullIfNotPresent}}}
		assert.True(t, specRisksStaleImage(staleSpec), "mutable tag + IfNotPresent")

		freshSpec := &corev1.PodSpec{Containers: []corev1.Container{{Image: "nginx:1.25", ImagePullPolicy: corev1.PullAlways}}}
		assert.False(t, specRisksStaleImage(freshSpec), "mutable tag + Always")

		digestSpec := &corev1.PodSpec{Containers: []corev1.Container{{Image: digest, ImagePullPolicy: corev1.PullIfNotPresent}}}
		assert.False(t, specRisksStaleImage(digestSpec), "digest is always exact")
	})

	t.Run("unparseable image does not count as Always pull", func(t *testing.T) {
		spec := &corev1.PodSpec{Containers: []corev1.Container{{Image: "UPPER CASE !! invalid"}}}
		assert.Equal(t, corev1.PullIfNotPresent, effectivePullPolicy(spec.Containers[0]), "unparseable defaults to IfNotPresent")
		assert.False(t, specUsesAlwaysImagePullPolicy(spec), "unparseable image is not Always")
		assert.True(t, specRisksStaleImage(spec), "unparseable image is treated as a stale risk")
	})

	t.Run("usesAlwaysImagePullPolicy requires every container", func(t *testing.T) {
		spec := &corev1.PodSpec{Containers: []corev1.Container{
			{Image: "a:1", ImagePullPolicy: corev1.PullAlways},
			{Image: "b:1", ImagePullPolicy: corev1.PullIfNotPresent},
		}}
		assert.False(t, specUsesAlwaysImagePullPolicy(spec))
		assert.False(t, specUsesAlwaysImagePullPolicy(nil))
	})

	t.Run("image lists dedup and split on pinning", func(t *testing.T) {
		spec := &corev1.PodSpec{
			InitContainers: []corev1.Container{{Image: "shared:1"}},
			Containers: []corev1.Container{
				{Image: "shared:1"},
				{Image: digest},
			},
		}
		assert.ElementsMatch(t, []any{"shared:1", digest}, specContainerImages(spec), "containerImages dedup")
		assert.ElementsMatch(t, []any{"shared:1"}, specUnpinnedImages(spec), "unpinnedImages excludes digest")
	})
}
