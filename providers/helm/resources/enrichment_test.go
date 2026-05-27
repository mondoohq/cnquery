// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/lint/support"
)

func TestParseHookAnnotations(t *testing.T) {
	t.Run("non-hook", func(t *testing.T) {
		isHook, types, weight, del := parseHookAnnotations(map[string]any{"app": "x"})
		assert.False(t, isHook)
		assert.Empty(t, types)
		assert.Zero(t, weight)
		assert.Empty(t, del)
	})

	t.Run("full hook", func(t *testing.T) {
		isHook, types, weight, del := parseHookAnnotations(map[string]any{
			"helm.sh/hook":               "pre-install,post-install",
			"helm.sh/hook-weight":        "5",
			"helm.sh/hook-delete-policy": "before-hook-creation, hook-succeeded",
		})
		assert.True(t, isHook)
		assert.Equal(t, []any{"pre-install", "post-install"}, types)
		assert.Equal(t, int64(5), weight)
		assert.Equal(t, []any{"before-hook-creation", "hook-succeeded"}, del)
	})

	t.Run("hook with non-numeric weight", func(t *testing.T) {
		isHook, _, weight, _ := parseHookAnnotations(map[string]any{
			"helm.sh/hook":        "test",
			"helm.sh/hook-weight": "not-a-number",
		})
		assert.True(t, isHook)
		assert.Zero(t, weight)
	})
}

func TestMergeMaps(t *testing.T) {
	dst := map[string]any{
		"a": 1,
		"nested": map[string]any{
			"keep":     "yes",
			"override": "old",
		},
	}
	src := map[string]any{
		"b": 2,
		"nested": map[string]any{
			"override": "new",
		},
	}
	out := mergeMaps(dst, src)

	assert.Equal(t, 1, out["a"])
	assert.Equal(t, 2, out["b"])
	nested := out["nested"].(map[string]any)
	assert.Equal(t, "yes", nested["keep"], "untouched nested keys survive")
	assert.Equal(t, "new", nested["override"], "src wins on conflict")
	// dst must not be mutated.
	assert.Equal(t, "old", dst["nested"].(map[string]any)["override"])
}

func TestTemplateUsesLookup(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{"plain", "kind: ConfigMap\ndata:\n  x: {{ .Values.x }}", false},
		{"lookup in action", `data: {{ (lookup "v1" "ConfigMap" "" "x") }}`, true},
		{"lookup in if", `{{ if lookup "v1" "Secret" .Release.Namespace "s" }}yes{{ end }}`, true},
		{"word lookup in string is not a call", `data: "we lookup things"`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, templateUsesLookup(tt.name, tt.raw))
		})
	}
}

func TestLintSeverityName(t *testing.T) {
	assert.Equal(t, "info", lintSeverityName(support.InfoSev))
	assert.Equal(t, "warning", lintSeverityName(support.WarningSev))
	assert.Equal(t, "error", lintSeverityName(support.ErrorSev))
	assert.Equal(t, "unknown", lintSeverityName(support.UnknownSev))
}
