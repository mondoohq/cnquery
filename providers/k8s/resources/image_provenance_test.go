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
