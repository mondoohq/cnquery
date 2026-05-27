// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeStringList(t *testing.T) {
	in := []string{"a=1", "b=2", "c,with,commas"}
	enc := EncodeStringList(in)
	assert.NotEmpty(t, enc)
	assert.Equal(t, in, decodeStringList(enc))

	assert.Empty(t, EncodeStringList(nil))
	assert.Nil(t, decodeStringList(""))
	assert.Nil(t, decodeStringList("not json"))
}

func TestParseRenderOptions(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		ro := parseRenderOptions(map[string]string{})
		assert.Equal(t, "default", ro.Namespace, "namespace defaults to default")
		assert.False(t, ro.IsUpgrade)
		assert.Empty(t, ro.ValueFiles)
	})

	t.Run("populated", func(t *testing.T) {
		ro := parseRenderOptions(map[string]string{
			OptionValues:      EncodeStringList([]string{"prod.yaml"}),
			OptionSet:         EncodeStringList([]string{"replicas=5"}),
			OptionAPIVersions: EncodeStringList([]string{"batch/v1", "policy/v1"}),
			OptionNamespace:   "team-a",
			OptionReleaseName: "rel",
			OptionKubeVersion: "1.29.0",
			OptionIsUpgrade:   "true",
		})
		assert.Equal(t, []string{"prod.yaml"}, ro.ValueFiles)
		assert.Equal(t, []string{"replicas=5"}, ro.Values)
		assert.Equal(t, []string{"batch/v1", "policy/v1"}, ro.APIVersions)
		assert.Equal(t, "team-a", ro.Namespace)
		assert.Equal(t, "rel", ro.ReleaseName)
		assert.Equal(t, "1.29.0", ro.KubeVersion)
		assert.True(t, ro.IsUpgrade)
	})
}
