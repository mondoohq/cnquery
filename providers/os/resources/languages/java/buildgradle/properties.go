// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package buildgradle

import (
	"bufio"
	"io"
	"strings"
)

// Where a Gradle project's versions actually live.
//
// A build script rarely states its versions inline. They are declared once, for
// the whole build, in a `gradle.properties` or a root `ext` block — often in a
// script of their own (`constants.gradle`, `versions.gradle`) that the modules
// apply. The module then writes `implementation "androidx.annotation:annotation:$androidxAnnotationVersion"`,
// and reading that file alone yields an artifact with no version, which no
// advisory can match.
//
// These two collectors read the two formats those declarations come in. Which
// files to feed them is the caller's question, because it depends on the
// project layout, not on the file format.

// maxPropertyLines bounds a property-source read. These are small files by
// nature, and the scan is driven by repository content.
const maxPropertyLines = 20000

// CollectScriptProperties reads the version properties a Gradle script
// declares: `def x = '1.2'`, `val x = "1.2"`, `ext.x = '1.2'`, and the bare
// `x = '1.2'` form used inside an `ext { }` or `project.ext { }` block.
func CollectScriptProperties(r io.Reader) map[string]string {
	var lines []string
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() && len(lines) < maxPropertyLines {
		lines = append(lines, scanner.Text())
	}
	return collectVars(lines)
}

// CollectPropertiesFile reads a gradle.properties: `key=value` lines, where the
// value is unquoted and runs to the end of the line.
//
// Only property names that look like a version reference are kept. A
// gradle.properties also carries build settings — `org.gradle.jvmargs`,
// `android.useAndroidX`, memory flags — and those share a namespace with the
// version properties a script interpolates. Keeping them would let a build
// setting be substituted into a coordinate.
func CollectPropertiesFile(r io.Reader) map[string]string {
	out := map[string]string{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for n := 0; scanner.Scan() && n < maxPropertyLines; n++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" || !isVersionProperty(key) {
			continue
		}
		out[key] = value
	}
	return out
}

// isVersionProperty reports whether a property name reads as a version.
//
// A dotted name is always a build setting: Gradle's own settings are namespaced
// (`org.gradle.*`, `android.*`), and a script interpolating `$x` cannot name a
// dotted property that way regardless.
func isVersionProperty(key string) bool {
	if strings.Contains(key, ".") {
		return false
	}
	return strings.Contains(strings.ToLower(key), "version")
}
