// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shell

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// MQL syntax highlighting colors
var (
	hlKeyword  = lipgloss.NewStyle().Foreground(lipgloss.Color("204")) // Pink for keywords
	hlResource = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))  // Blue for resources
	hlField    = lipgloss.NewStyle().Foreground(lipgloss.Color("156")) // Light green for fields
	hlString   = lipgloss.NewStyle().Foreground(lipgloss.Color("221")) // Yellow for strings
	hlNumber   = lipgloss.NewStyle().Foreground(lipgloss.Color("141")) // Purple for numbers
	hlOperator = lipgloss.NewStyle().Foreground(lipgloss.Color("203")) // Red for operators
	hlBracket  = lipgloss.NewStyle().Foreground(lipgloss.Color("245")) // Gray for brackets
	hlComment  = lipgloss.NewStyle().Foreground(lipgloss.Color("242")) // Dark gray for comments
)

// MQL keywords
var mqlKeywords = map[string]bool{
	"if": true, "else": true, "return": true, "where": true,
	"contains": true, "in": true, "not": true, "and": true, "or": true,
	"true": true, "false": true, "null": true, "props": true,
}

// Common MQL resources (top-level)
var mqlResources = map[string]bool{
	"asset": true, "mondoo": true, "users": true, "groups": true,
	"packages": true, "services": true, "processes": true, "ports": true,
	"files": true, "file": true, "command": true, "parse": true,
	"platform": true, "kernel": true, "sshd": true, "os": true,
	"aws": true, "gcp": true, "azure": true, "k8s": true, "terraform": true,
	"arista": true, "github": true, "gitlab": true, "okta": true, "ms365": true,
	"vsphere": true, "docker": true, "container": true, "image": true,
}

// highlightMQL applies syntax highlighting to MQL code
func highlightMQL(code string) string {
	// Handle empty input
	if code == "" {
		return code
	}

	var result strings.Builder
	i := 0

	for i < len(code) {
		// Check for comments
		if i+1 < len(code) && code[i:i+2] == "//" {
			end := strings.Index(code[i:], "\n")
			if end == -1 {
				result.WriteString(hlComment.Render(code[i:]))
				break
			}
			result.WriteString(hlComment.Render(code[i : i+end]))
			i += end
			continue
		}

		// Check for strings (double quotes)
		if code[i] == '"' {
			end := i + 1
			for end < len(code) && code[end] != '"' {
				if code[end] == '\\' && end+1 < len(code) {
					end += 2
				} else {
					end++
				}
			}
			if end < len(code) {
				end++ // include closing quote
			}
			result.WriteString(hlString.Render(code[i:end]))
			i = end
			continue
		}

		// Check for strings (single quotes)
		if code[i] == '\'' {
			end := i + 1
			for end < len(code) && code[end] != '\'' {
				if code[end] == '\\' && end+1 < len(code) {
					end += 2
				} else {
					end++
				}
			}
			if end < len(code) {
				end++ // include closing quote
			}
			result.WriteString(hlString.Render(code[i:end]))
			i = end
			continue
		}

		// Check for numbers
		if isDigit(code[i]) {
			end := i
			for end < len(code) && (isDigit(code[end]) || code[end] == '.') {
				end++
			}
			result.WriteString(hlNumber.Render(code[i:end]))
			i = end
			continue
		}

		// Check for operators
		if isOperator(code[i]) {
			// Handle multi-character operators
			op := string(code[i])
			if i+1 < len(code) {
				twoChar := code[i : i+2]
				if twoChar == "==" || twoChar == "!=" || twoChar == ">=" ||
					twoChar == "<=" || twoChar == "&&" || twoChar == "||" ||
					twoChar == "=~" || twoChar == "!~" {
					op = twoChar
				}
			}
			result.WriteString(hlOperator.Render(op))
			i += len(op)
			continue
		}

		// Check for brackets
		if isBracket(code[i]) {
			result.WriteString(hlBracket.Render(string(code[i])))
			i++
			continue
		}

		// Check for identifiers (words)
		if isAlpha(code[i]) || code[i] == '_' {
			end := i
			for end < len(code) && (isAlphaNum(code[end]) || code[end] == '_') {
				end++
			}
			word := code[i:end]

			// Check if it's followed by a dot (field access)
			isFieldAccess := end < len(code) && code[end] == '.'

			// Check what comes before (to detect if it's a field after a dot)
			isAfterDot := i > 0 && code[i-1] == '.'

			if mqlKeywords[word] {
				result.WriteString(hlKeyword.Render(word))
			} else if mqlResources[word] && !isAfterDot {
				result.WriteString(hlResource.Render(word))
			} else if isAfterDot || isFieldAccess {
				result.WriteString(hlField.Render(word))
			} else {
				result.WriteString(word)
			}
			i = end
			continue
		}

		// Default: pass through
		result.WriteByte(code[i])
		i++
	}

	return result.String()
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isAlphaNum(c byte) bool {
	return isAlpha(c) || isDigit(c)
}

func isOperator(c byte) bool {
	return c == '=' || c == '!' || c == '<' || c == '>' ||
		c == '+' || c == '-' || c == '*' || c == '/' ||
		c == '&' || c == '|' || c == '~'
}

func isBracket(c byte) bool {
	return c == '{' || c == '}' || c == '[' || c == ']' || c == '(' || c == ')'
}

// Regex for more complex patterns (unused but available)
var (
	_ = regexp.MustCompile(`"[^"]*"`)
	_ = regexp.MustCompile(`'[^']*'`)
)
