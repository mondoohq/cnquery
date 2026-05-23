// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyDirective(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{"if .Values.enabled", "if"},
		{"else if .Values.other", "if"},
		{"range .Values.items", "range"},
		{"with .Values.config", "with"},
		{"define \"mytemplate\"", "define"},
		{"block \"name\" .", "block"},
		{"template \"name\" .", "include"},
		{"include \"mychart.labels\" .", "include"},
		{"tpl .Values.template .", "tpl"},
		{".Values.replicaCount", ""},
		{"end", ""},
		{"else", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			assert.Equal(t, tt.expected, classifyDirective(tt.expr))
		})
	}
}

func TestExtractDirectivesFallback(t *testing.T) {
	t.Run("helm template with include and toYaml", func(t *testing.T) {
		raw := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Release.Name }}
  labels:
    {{- include "mychart.labels" . | nindent 4 }}
spec:
  {{- if .Values.autoscaling.enabled }}
  replicas: {{ .Values.replicaCount }}
  {{- end }}
  template:
    spec:
      containers:
        {{- range .Values.containers }}
        - name: {{ .name }}
        {{- end }}
        {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- tpl (toYaml .) $ | nindent 8 }}
        {{- end }}`

		// extractDirectivesFallback needs a runtime, but we can test the
		// classification logic directly instead
		directives := extractDirectivesFromLines(raw)

		typeCount := map[string]int{}
		for _, d := range directives {
			typeCount[d.typ]++
		}

		assert.Equal(t, 1, typeCount["include"], "should find 1 include directive")
		assert.Equal(t, 1, typeCount["if"], "should find 1 if directive")
		assert.Equal(t, 1, typeCount["range"], "should find 1 range directive")
		assert.Equal(t, 1, typeCount["with"], "should find 1 with directive")
		assert.Equal(t, 1, typeCount["tpl"], "should find 1 tpl directive")
	})

	t.Run("empty template", func(t *testing.T) {
		directives := extractDirectivesFromLines("")
		assert.Empty(t, directives)
	})

	t.Run("template with no directives", func(t *testing.T) {
		raw := `apiVersion: v1
kind: ConfigMap
metadata:
  name: static-config
data:
  key: value`
		directives := extractDirectivesFromLines(raw)
		assert.Empty(t, directives)
	})

	t.Run("multiple directives on same line", func(t *testing.T) {
		raw := `{{- if .Values.a }}{{ range .Values.b }}item{{ end }}{{ end }}`
		directives := extractDirectivesFromLines(raw)

		types := []string{}
		for _, d := range directives {
			types = append(types, d.typ)
		}
		assert.Contains(t, types, "if")
		assert.Contains(t, types, "range")
	})

	t.Run("whitespace-trimmed directives", func(t *testing.T) {
		raw := `{{- if .Values.enabled -}}`
		directives := extractDirectivesFromLines(raw)
		require.Len(t, directives, 1)
		assert.Equal(t, "if", directives[0].typ)
	})
}

// directiveInfo is a simplified version for testing without a runtime.
type directiveInfo struct {
	typ        string
	expression string
	line       int
}

// extractDirectivesFromLines extracts directive info without needing a plugin.Runtime.
func extractDirectivesFromLines(rawContent string) []directiveInfo {
	var directives []directiveInfo

	if rawContent == "" {
		return directives
	}

	lines := splitLines(rawContent)
	for i, line := range lines {
		trimmed := line
		for {
			start := indexOf(trimmed, "{{")
			if start == -1 {
				break
			}
			end := indexOf(trimmed[start:], "}}")
			if end == -1 {
				break
			}
			expr := trimDirectiveExpr(trimmed[start+2 : start+end])

			directiveType := classifyDirective(expr)
			if directiveType != "" {
				directives = append(directives, directiveInfo{
					typ:        directiveType,
					expression: expr,
					line:       i + 1,
				})
			}

			trimmed = trimmed[start+end+2:]
		}
	}

	return directives
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func trimDirectiveExpr(s string) string {
	// Trim whitespace
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	s = s[start:end]
	// Trim leading/trailing -
	if len(s) > 0 && s[0] == '-' {
		s = s[1:]
	}
	if len(s) > 0 && s[len(s)-1] == '-' {
		s = s[:len(s)-1]
	}
	// Trim again
	start, end = 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty string", "", nil},
		{"single line no newline", "hello", []string{"hello"}},
		{"single line with newline", "hello\n", []string{"hello"}},
		{"multiple lines", "a\nb\nc", []string{"a", "b", "c"}},
		{"trailing newline", "a\nb\n", []string{"a", "b"}},
		{"blank lines in middle", "a\n\nb", []string{"a", "", "b"}},
		{"only newlines", "\n\n\n", []string{"", "", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLines(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrimDirectiveExpr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain expression", "if .Values.x", "if .Values.x"},
		{"leading dash", "- if .Values.x", "if .Values.x"},
		{"trailing dash", "if .Values.x -", "if .Values.x"},
		{"both dashes", "- if .Values.x -", "if .Values.x"},
		{"dashes with spaces", " - if .Values.x - ", "if .Values.x"},
		{"tabs", "\tif .Values.x\t", "if .Values.x"},
		{"only whitespace", "   ", ""},
		{"only dashes", " - - ", ""},
		{"empty string", "", ""},
		{"dash with tab padding", "\t- range .Items -\t", "range .Items"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimDirectiveExpr(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIndexOf(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected int
	}{
		{"found at start", "{{hello}}", "{{", 0},
		{"found in middle", "abc{{def", "{{", 3},
		{"found at end", "abc}}", "}}", 3},
		{"not found", "hello", "{{", -1},
		{"empty haystack", "", "{{", -1},
		{"empty needle in non-empty", "abc", "", 0},
		{"same length match", "{{", "{{", 0},
		{"same length no match", "ab", "{{", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexOf(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractDirectivesFallbackEdgeCases(t *testing.T) {
	t.Run("deeply nested directives", func(t *testing.T) {
		raw := `{{- if .Values.outer }}
  {{- range .Values.items }}
    {{- if .enabled }}
      {{- with .config }}
        {{- range .ports }}
        - containerPort: {{ . }}
        {{- end }}
      {{- end }}
    {{- end }}
  {{- end }}
{{- end }}`
		directives := extractDirectivesFromLines(raw)

		typeCount := map[string]int{}
		for _, d := range directives {
			typeCount[d.typ]++
		}

		assert.Equal(t, 2, typeCount["if"], "should find 2 if directives")
		assert.Equal(t, 2, typeCount["range"], "should find 2 range directives")
		assert.Equal(t, 1, typeCount["with"], "should find 1 with directive")
	})

	t.Run("template with only comments", func(t *testing.T) {
		raw := `# This is a comment
# Another comment
# No directives here`
		directives := extractDirectivesFromLines(raw)
		assert.Empty(t, directives)
	})

	t.Run("unclosed double braces", func(t *testing.T) {
		raw := `{{ if .Values.x
no closing braces here`
		directives := extractDirectivesFromLines(raw)
		// The {{ without }} on same line should be skipped
		assert.Empty(t, directives)
	})

	t.Run("directives with define and block", func(t *testing.T) {
		raw := `{{- define "mychart.labels" -}}
app: {{ .Chart.Name }}
{{- end -}}
{{- block "extra" . -}}
default content
{{- end -}}`
		directives := extractDirectivesFromLines(raw)

		types := map[string]bool{}
		for _, d := range directives {
			types[d.typ] = true
		}
		assert.True(t, types["define"], "should find define directive")
		assert.True(t, types["block"], "should find block directive")
	})

	t.Run("line numbers are correct", func(t *testing.T) {
		raw := `line1
{{ if .Values.a }}
line3
{{ range .Values.b }}
line5`
		directives := extractDirectivesFromLines(raw)
		require.Len(t, directives, 2)
		assert.Equal(t, 2, directives[0].line)
		assert.Equal(t, 4, directives[1].line)
	})

	t.Run("action expressions are not classified", func(t *testing.T) {
		raw := `{{ .Values.replicaCount }}
{{ printf "%s-%s" .Release.Name .Chart.Name }}
{{ .Release.Namespace }}`
		directives := extractDirectivesFromLines(raw)
		assert.Empty(t, directives, "plain action expressions should not be classified as directives")
	})
}

func TestClassifyDirectiveEdgeCases(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		// Verify that partial matches don't trigger false positives
		{"iffy", ""},
		{"ranger .Values", ""},
		{"within .Values", ""},
		{"defined \"x\"", ""},
		{"blocked \"x\"", ""},
		{"including \"x\"", ""},
		// Extra spaces should not match (prefix-based)
		{" if .Values.x", ""},
		{" range .Values.x", ""},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			assert.Equal(t, tt.expected, classifyDirective(tt.expr))
		})
	}
}

func TestFindClosingDelim(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{
			name: "trivial",
			in:   " .Values.name }} rest",
			want: 14, // offset of `}}`
		},
		{
			name: "no closing delim",
			in:   " .Values.name",
			want: -1,
		},
		{
			name: "embedded }} inside double-quoted string is skipped",
			// PVE-style embedded `}}` inside a quoted argument used to
			// terminate the directive prematurely; findClosingDelim
			// looks past it.
			in:   ` dict "key" "}}" }} rest`,
			want: 17,
		},
		{
			name: "embedded }} inside backtick string is skipped",
			in:   " printf `oops}}` }} rest",
			want: 17,
		},
		{
			name: "escaped quote inside double-quoted string doesn't end the literal",
			in:   ` printf "with \"quote\" and }}" }} rest`,
			want: 32,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findClosingDelim(tt.in)
			require.Equal(t, tt.want, got, "input: %q", tt.in)
		})
	}
}

// Exercise the full fallback path against an input shaped like the
// edge case that motivated findClosingDelim. The naive `strings.Index`
// version desynced the loop on the first directive; the fixed version
// produces a single, well-formed directive.
func TestExtractDirectivesFallback_HandlesEmbeddedDelimiter(t *testing.T) {
	runtime := newTestRuntime()
	content := `kind: ConfigMap
data:
  hello: {{ if and (eq .Values.kind "}}") (.Values.enabled) }}greeting{{ end }}
`
	dirs, err := extractDirectivesFallback(runtime, "configmap.yaml", content)
	require.NoError(t, err)
	// The opening `{{ if ... }}` should classify as `if` and the
	// closing `{{ end }}` should not. Two callable lines, but only one
	// directive recognized (end is filtered to "" by classifyDirective).
	assert.Len(t, dirs, 1, "expected exactly one classified directive, got %d", len(dirs))
}
