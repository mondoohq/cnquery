// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package snmpd parses Net-SNMP daemon (snmpd.conf) configuration files.
package snmpd

import "strings"

// Directive is a single configuration directive: a keyword followed by its
// whitespace-separated arguments.
type Directive struct {
	// Keyword is the directive name as written (case preserved). Callers that
	// match on it should fold case, since snmpd keywords are case-insensitive.
	Keyword string
	// Args holds the tokenized arguments after the keyword.
	Args []string
	// Line is the 1-based line number of this directive within its file.
	Line int
}

// Parse reads snmpd.conf content and returns its directives in order. Comment
// lines and blank lines produce no directive. A `#` outside of quotes begins a
// comment that runs to the end of the line. Double- and single-quoted tokens
// keep their interior whitespace and are returned without the quotes.
func Parse(content string) []Directive {
	res := []Directive{}
	for i, raw := range strings.Split(content, "\n") {
		tokens := tokenize(raw)
		if len(tokens) == 0 {
			continue
		}
		res = append(res, Directive{
			Keyword: tokens[0],
			Args:    tokens[1:],
			Line:    i + 1,
		})
	}
	return res
}

// tokenize splits a single line into whitespace-separated tokens, honoring
// quoting and stopping at an unquoted comment.
func tokenize(line string) []string {
	var tokens []string
	runes := []rune(line)
	n := len(runes)
	i := 0

	for i < n {
		// skip leading whitespace
		for i < n && isSpace(runes[i]) {
			i++
		}
		if i >= n {
			break
		}
		// a `#` at a token boundary starts a comment
		if runes[i] == '#' {
			break
		}

		var sb strings.Builder
		if runes[i] == '"' || runes[i] == '\'' {
			quote := runes[i]
			i++
			for i < n && runes[i] != quote {
				if runes[i] == '\\' && i+1 < n {
					i++
				}
				sb.WriteRune(runes[i])
				i++
			}
			if i < n {
				i++ // consume the closing quote
			}
		} else {
			for i < n && !isSpace(runes[i]) {
				sb.WriteRune(runes[i])
				i++
			}
		}
		tokens = append(tokens, sb.String())
	}

	return tokens
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\r'
}
