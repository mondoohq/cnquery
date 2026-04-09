// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"regexp"
	"strings"
)

// parsedBicepFile holds the result of parsing a .bicep file.
type parsedBicepFile struct {
	targetScope string
	parameters  []parsedParameter
	variables   []parsedVariable
	resources   []parsedResource
	modules     []parsedModule
	outputs     []parsedOutput
}

type parsedParameter struct {
	name         string
	typ          string
	defaultValue string
	description  string
	secure       bool
	allowed      []string
	decorators   []string
}

type parsedVariable struct {
	name        string
	expression  string
	description string
}

type parsedResource struct {
	symbolicName string
	typ          string
	apiVersion   string
	name         string
	location     string
	existing     bool
	condition    string
	parent       string
	body         string
	dependsOn    []string
	decorators   []string
}

type parsedModule struct {
	name           string
	source         string
	scope          string
	condition      string
	body           string
	description    string
	isRegistry     bool
	isTemplateSpec bool
	decorators     []string
}

type parsedOutput struct {
	name        string
	typ         string
	expression  string
	description string
}

var (
	targetScopeRe = regexp.MustCompile(`(?m)^targetScope\s*=\s*'([^']+)'`)
	paramRe       = regexp.MustCompile(`(?m)^param\s+(\w+)\s+(\w+)(.*)$`)
	varRe         = regexp.MustCompile(`(?m)^var\s+(\w+)\s*=\s*(.+)$`)
	resourceRe    = regexp.MustCompile(`(?m)^(resource)\s+(\w+)\s+'([^']+)'(\s+existing)?\s*=`)
	moduleRe      = regexp.MustCompile(`(?m)^module\s+(\w+)\s+'([^']+)'\s*=`)
	outputRe      = regexp.MustCompile(`(?m)^output\s+(\w+)\s+(\w+)\s*=\s*(.+)$`)
	decoratorRe   = regexp.MustCompile(`(?m)^@(\w+)\(([^)]*)\)`)
	descDecRe     = regexp.MustCompile(`@description\('([^']*)'\)`)
	secureDecRe   = regexp.MustCompile(`@secure\(\)`)
	allowedDecRe  = regexp.MustCompile(`@allowed\(\[([^\]]*)\]\)`)
)

func parseBicep(content string) *parsedBicepFile {
	result := &parsedBicepFile{}

	// Target scope
	if m := targetScopeRe.FindStringSubmatch(content); len(m) > 1 {
		result.targetScope = m[1]
	}

	lines := strings.Split(content, "\n")
	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Collect decorators. Multiline decorators (e.g. @allowed([\n...\n]))
		// are reassembled by tracking paren/bracket depth.
		var decorators []string
		for strings.HasPrefix(trimmed, "@") {
			decLine := trimmed
			depth := parenBracketDepth(decLine)
			i++
			for depth > 0 && i < len(lines) {
				line = lines[i]
				trimmed = strings.TrimSpace(line)
				decLine += "\n" + trimmed
				depth += parenBracketDepth(trimmed)
				i++
			}
			decorators = append(decorators, decLine)
			if i >= len(lines) {
				break
			}
			line = lines[i]
			trimmed = strings.TrimSpace(line)
		}

		if strings.HasPrefix(trimmed, "param ") {
			result.parameters = append(result.parameters, parseParameter(trimmed, decorators))
			i++
			continue
		}

		if strings.HasPrefix(trimmed, "var ") {
			result.variables = append(result.variables, parseVariable(trimmed, decorators))
			i++
			continue
		}

		if strings.HasPrefix(trimmed, "resource ") {
			res, consumed := parseResourceDecl(lines, i, decorators)
			if res != nil {
				result.resources = append(result.resources, *res)
			}
			i = consumed
			continue
		}

		if strings.HasPrefix(trimmed, "module ") {
			mod, consumed := parseModuleDecl(lines, i, decorators)
			if mod != nil {
				result.modules = append(result.modules, *mod)
			}
			i = consumed
			continue
		}

		if strings.HasPrefix(trimmed, "output ") {
			result.outputs = append(result.outputs, parseOutput(trimmed, decorators))
			i++
			continue
		}

		i++
	}

	return result
}

func parseParameter(line string, decorators []string) parsedParameter {
	p := parsedParameter{decorators: decorators}

	m := paramRe.FindStringSubmatch(line)
	if len(m) >= 3 {
		p.name = m[1]
		p.typ = m[2]
		if len(m) > 3 {
			rest := strings.TrimSpace(m[3])
			if strings.HasPrefix(rest, "=") {
				val := strings.TrimSpace(rest[1:])
				// Strip Bicep single-quote string delimiters
				if len(val) >= 2 && val[0] == '\'' && val[len(val)-1] == '\'' {
					val = val[1 : len(val)-1]
				}
				p.defaultValue = val
			}
		}
	}

	decText := strings.Join(decorators, "\n")
	if m := descDecRe.FindStringSubmatch(decText); len(m) > 1 {
		p.description = m[1]
	}
	p.secure = secureDecRe.MatchString(decText)
	if m := allowedDecRe.FindStringSubmatch(decText); len(m) > 1 {
		p.allowed = parseAllowedValues(m[1])
	}

	return p
}

// allowedValueRe matches individual quoted values like 'foo' or "foo".
var allowedValueRe = regexp.MustCompile(`'([^']*)'`)

func parseAllowedValues(raw string) []string {
	// Extract all single-quoted values from the raw content.
	// This handles both newline-separated and comma-separated formats:
	//   'Standard_LRS'\n'Standard_GRS'
	//   'win10', 'ws2019'
	matches := allowedValueRe.FindAllStringSubmatch(raw, -1)
	var vals []string
	for _, m := range matches {
		vals = append(vals, m[1])
	}
	return vals
}

func parseVariable(line string, decorators []string) parsedVariable {
	v := parsedVariable{}
	m := varRe.FindStringSubmatch(line)
	if len(m) >= 3 {
		v.name = m[1]
		v.expression = strings.TrimSpace(m[2])
	}
	decText := strings.Join(decorators, "\n")
	if m := descDecRe.FindStringSubmatch(decText); len(m) > 1 {
		v.description = m[1]
	}
	return v
}

func parseResourceDecl(lines []string, startIdx int, decorators []string) (*parsedResource, int) {
	line := strings.TrimSpace(lines[startIdx])
	m := resourceRe.FindStringSubmatch(line)
	if len(m) < 4 {
		return nil, startIdx + 1
	}

	r := &parsedResource{
		symbolicName: m[2],
		decorators:   decorators,
	}

	// Parse type@apiVersion from the quoted string
	typeAndVersion := m[3]
	if parts := strings.SplitN(typeAndVersion, "@", 2); len(parts) == 2 {
		r.typ = parts[0]
		r.apiVersion = parts[1]
	} else {
		r.typ = typeAndVersion
	}

	r.existing = len(m) > 4 && strings.TrimSpace(m[4]) == "existing"

	// Find the body between { and }
	body, endIdx := extractBlock(lines, startIdx)
	r.body = body

	// Extract common fields from body
	r.name = extractFieldValue(body, "name")
	r.location = extractFieldValue(body, "location")
	r.condition = extractCondition(line)
	r.parent = extractFieldValue(body, "parent")
	r.dependsOn = extractDependsOn(body)

	return r, endIdx
}

func parseModuleDecl(lines []string, startIdx int, decorators []string) (*parsedModule, int) {
	line := strings.TrimSpace(lines[startIdx])
	m := moduleRe.FindStringSubmatch(line)
	if len(m) < 3 {
		return nil, startIdx + 1
	}

	mod := &parsedModule{
		name:           m[1],
		source:         m[2],
		isRegistry:     strings.HasPrefix(m[2], "br:") || strings.HasPrefix(m[2], "br/"),
		isTemplateSpec: strings.HasPrefix(m[2], "ts:") || strings.HasPrefix(m[2], "ts/"),
		decorators:     decorators,
	}

	mod.condition = extractCondition(line)
	body, endIdx := extractBlock(lines, startIdx)
	mod.body = body
	mod.scope = extractFieldValue(body, "scope")

	decText := strings.Join(decorators, "\n")
	if dm := descDecRe.FindStringSubmatch(decText); len(dm) > 1 {
		mod.description = dm[1]
	}

	return mod, endIdx
}

func parseOutput(line string, decorators []string) parsedOutput {
	o := parsedOutput{}
	m := outputRe.FindStringSubmatch(line)
	if len(m) >= 4 {
		o.name = m[1]
		o.typ = m[2]
		o.expression = strings.TrimSpace(m[3])
	}
	decText := strings.Join(decorators, "\n")
	if m := descDecRe.FindStringSubmatch(decText); len(m) > 1 {
		o.description = m[1]
	}
	return o
}

// extractBlock extracts a brace-delimited block starting from startIdx by
// counting '{' and '}' characters to track nesting depth.
//
// Known limitation: the depth counter scans the entire line without skipping
// string literals (e.g., Bicep string interpolation '...${expr}...') or
// comments that may contain braces. A full lexer would be needed to handle
// these cases correctly, but the regex/line-scanning approach used by the
// Bicep parser is intentionally lightweight. In practice, mismatched depth
// from braces inside strings or comments is rare in typical Bicep files and
// the impact is a slightly shifted block boundary rather than a hard failure.
func extractBlock(lines []string, startIdx int) (string, int) {
	depth := 0
	started := false
	var blockLines []string

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		for _, ch := range line {
			if ch == '{' {
				depth++
				started = true
			}
			if ch == '}' {
				depth--
			}
		}
		if started {
			blockLines = append(blockLines, line)
		}
		if started && depth == 0 {
			return strings.Join(blockLines, "\n"), i + 1
		}
	}

	return strings.Join(blockLines, "\n"), len(lines)
}

// fieldValueRegexCache stores pre-compiled regexes for extractFieldValue lookups.
var fieldValueRegexCache = func() map[string]*regexp.Regexp {
	fields := []string{"name", "location", "parent", "scope"}
	m := make(map[string]*regexp.Regexp, len(fields))
	for _, f := range fields {
		m[f] = regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(f) + `\s*:\s*(.+)$`)
	}
	return m
}()

func extractFieldValue(body string, fieldName string) string {
	re, ok := fieldValueRegexCache[fieldName]
	if !ok {
		re = regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(fieldName) + `\s*:\s*(.+)$`)
	}
	m := re.FindStringSubmatch(body)
	if len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func extractCondition(line string) string {
	// condition is expressed as: resource foo 'Type@ver' = if (expr) { ... }
	if idx := strings.Index(line, "= if"); idx >= 0 {
		rest := strings.TrimSpace(line[idx+4:])
		// Extract the condition expression in parens
		if strings.HasPrefix(rest, "(") {
			depth := 0
			for i, ch := range rest {
				if ch == '(' {
					depth++
				}
				if ch == ')' {
					depth--
					if depth == 0 {
						return rest[1:i]
					}
				}
			}
		}
	}
	return ""
}

// extractFieldBlock extracts the brace-delimited block for a top-level field
// like "params: { ... }" from a body string. Returns the raw content between
// the braces, or empty string if the field is not found.
func extractFieldBlock(body string, fieldName string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, fieldName+":") {
			// Find the opening brace on this or subsequent lines
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, fieldName+":"))
			if strings.HasPrefix(rest, "{") {
				block, _ := extractBlock(lines, i)
				// Strip the outer braces
				if idx := strings.Index(block, "{"); idx >= 0 {
					inner := block[idx+1:]
					if last := strings.LastIndex(inner, "}"); last >= 0 {
						inner = inner[:last]
					}
					return strings.TrimSpace(inner)
				}
				return block
			}
		}
	}
	return ""
}

// parenBracketDepth returns the net depth change from parens and brackets on a line.
// Positive means more openers than closers.
func parenBracketDepth(s string) int {
	depth := 0
	for _, ch := range s {
		switch ch {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		}
	}
	return depth
}

var dependsOnRe = regexp.MustCompile(`(?m)dependsOn\s*:\s*\[([^\]]*)\]`)

func extractDependsOn(body string) []string {
	re := dependsOnRe
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		return nil
	}

	var deps []string
	for _, part := range strings.Split(m[1], "\n") {
		part = strings.TrimSpace(part)
		part = strings.TrimSuffix(part, ",")
		part = strings.TrimSpace(part)
		if part != "" {
			deps = append(deps, part)
		}
	}
	return deps
}
