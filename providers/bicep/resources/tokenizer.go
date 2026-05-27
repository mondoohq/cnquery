// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "strings"

// bicepStatement is a single top-level construct in a .bicep file —
// a `param`, `var`, `resource`, `module`, `output`, `targetScope`, or an
// unrecognized line — together with any leading `@decorator(...)` lines
// that attach to it. The tokenizer yields these so the parser can dispatch
// on a complete, brace/bracket/string-aware statement body rather than
// re-walking raw source lines.
type bicepStatement struct {
	decorators []string // leading @decorators attached to this statement
	keyword    string   // "targetScope" | "param" | "var" | "resource" | "module" | "output" | "type" | "func" | "import" | "metadata" | "" (unknown)
	text       string   // the full statement source, multi-line, from keyword to the end of its body
	startLine  int      // 1-based, for diagnostics/future use
}

// knownKeywords maps a leading token to the canonical statement keyword.
// Anything not in this set produces a statement with an empty keyword so
// the parser never silently drops a line and future PRs can extend the
// classification (user-defined types, functions, imports, metadata, …).
var knownKeywords = map[string]string{
	"targetScope": "targetScope",
	"param":       "param",
	"var":         "var",
	"resource":    "resource",
	"module":      "module",
	"output":      "output",
	"type":        "type",
	"func":        "func",
	"import":      "import",
	"metadata":    "metadata",
}

// tokenizeBicep walks the source once and returns its top-level statements.
//
// A statement begins at the first non-blank, non-comment line after any
// leading decorators. It ends when, after consuming its first line, the
// running bracket/brace/paren depth (tracked string-aware via scanState)
// has returned to zero — so a single-line `param x string` ends at end of
// line, while a `resource ... = { ... }`, a `var x = [ ... ]`, or a
// multi-line object default keep consuming lines until their delimiters
// balance.
//
// Leading `@decorator(...)` lines — themselves possibly multi-line and
// depth-aware (e.g. `@allowed([\n ... \n])`) — attach to the next
// statement. Blank lines and `// ...` comment-only lines between
// statements are skipped.
func tokenizeBicep(content string) []bicepStatement {
	lines := strings.Split(content, "\n")
	var statements []bicepStatement

	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])

		// Skip blank lines and comment-only lines that sit between
		// statements (decorators are handled below, so we only land here
		// for separators).
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			i++
			continue
		}

		// Collect leading decorators. Each decorator may span multiple
		// lines; the string-aware scanner reassembles it so that delimiters
		// inside string literals (e.g. `@description('contains ] bracket')`)
		// don't keep it open past its real end.
		var decorators []string
		for strings.HasPrefix(trimmed, "@") {
			decLine := trimmed
			st := scanState{}
			st.feed(trimmed)
			i++
			for st.totalDepth() > 0 && i < len(lines) {
				t := strings.TrimSpace(lines[i])
				decLine += "\n" + t
				st.feed(t)
				i++
			}
			decorators = append(decorators, decLine)

			// Advance past any blank/comment lines between a decorator and
			// the statement (or the next decorator) it attaches to.
			for i < len(lines) {
				t := strings.TrimSpace(lines[i])
				if t == "" || strings.HasPrefix(t, "//") {
					i++
					continue
				}
				break
			}
			if i >= len(lines) {
				break
			}
			trimmed = strings.TrimSpace(lines[i])
		}

		// If decorators ran to EOF without a following statement, still emit
		// them as an empty-keyword statement so nothing is dropped.
		if i >= len(lines) {
			if len(decorators) > 0 {
				statements = append(statements, bicepStatement{
					decorators: decorators,
					keyword:    "",
					text:       "",
					startLine:  len(lines),
				})
			}
			break
		}

		// Reassemble the statement body: consume its first line, then keep
		// pulling continuation lines until the running depth returns to zero.
		startLine := i + 1 // 1-based
		st := scanState{}
		var bodyLines []string

		bodyLines = append(bodyLines, lines[i])
		st.feed(lines[i])
		i++
		for st.totalDepth() > 0 && i < len(lines) {
			bodyLines = append(bodyLines, lines[i])
			st.feed(lines[i])
			i++
		}

		text := strings.Join(bodyLines, "\n")
		keyword := classifyKeyword(strings.TrimSpace(bodyLines[0]))

		statements = append(statements, bicepStatement{
			decorators: decorators,
			keyword:    keyword,
			text:       text,
			startLine:  startLine,
		})
	}

	return statements
}

// classifyKeyword returns the canonical statement keyword for the leading
// token of a (trimmed) statement line, or "" when the leading token is not
// a recognized Bicep top-level keyword.
func classifyKeyword(line string) string {
	// The leading token ends at the first whitespace or `=` (covers
	// `targetScope = '...'` with no space before `=`).
	end := len(line)
	for idx := 0; idx < len(line); idx++ {
		c := line[idx]
		if c == ' ' || c == '\t' || c == '=' {
			end = idx
			break
		}
	}
	return knownKeywords[line[:end]]
}
