// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"regexp"
	"strconv"
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
	minLength    *int64
	maxLength    *int64
	minValue     *int64
	maxValue     *int64
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
	tags         map[string]string
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
	targetScopeRe  = regexp.MustCompile(`(?m)^targetScope\s*=\s*'([^']+)'`)
	paramRe        = regexp.MustCompile(`(?m)^param\s+(\w+)\s+(\w+)(.*)$`)
	varRe          = regexp.MustCompile(`(?m)^var\s+(\w+)\s*=\s*(.+)$`)
	resourceRe     = regexp.MustCompile(`(?m)^(resource)\s+(\w+)\s+'([^']+)'(\s+existing)?\s*=`)
	moduleRe       = regexp.MustCompile(`(?m)^module\s+(\w+)\s+'([^']+)'\s*=`)
	outputRe       = regexp.MustCompile(`(?m)^output\s+(\w+)\s+(\w+)\s*=\s*(.+)$`)
	decoratorRe    = regexp.MustCompile(`(?m)^@(\w+)\(([^)]*)\)`)
	descDecRe      = regexp.MustCompile(`@description\('([^']*)'\)`)
	secureDecRe    = regexp.MustCompile(`@secure\(\)`)
	allowedDecRe   = regexp.MustCompile(`@allowed\(\[([^\]]*)\]\)`)
	minLengthDecRe = regexp.MustCompile(`@minLength\(\s*(-?\d+)\s*\)`)
	maxLengthDecRe = regexp.MustCompile(`@maxLength\(\s*(-?\d+)\s*\)`)
	minValueDecRe  = regexp.MustCompile(`@minValue\(\s*(-?\d+)\s*\)`)
	maxValueDecRe  = regexp.MustCompile(`@maxValue\(\s*(-?\d+)\s*\)`)
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
			v, consumed := parseVariableDecl(lines, i, decorators)
			result.variables = append(result.variables, v)
			i = consumed
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
	p.minLength = parseIntDecorator(decText, minLengthDecRe)
	p.maxLength = parseIntDecorator(decText, maxLengthDecRe)
	p.minValue = parseIntDecorator(decText, minValueDecRe)
	p.maxValue = parseIntDecorator(decText, maxValueDecRe)

	return p
}

// parseIntDecorator matches a decorator like `@minLength(8)` and returns the
// numeric argument. Returns nil when the decorator is not present or its
// argument is non-numeric, so callers can distinguish an explicit constraint
// of 0 (e.g., `@minValue(0)`) from "no constraint at all".
func parseIntDecorator(decText string, re *regexp.Regexp) *int64 {
	m := re.FindStringSubmatch(decText)
	if len(m) < 2 {
		return nil
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return nil
	}
	return &n
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

// parseVariableDecl handles `var foo = ...` declarations, collecting
// continuation lines when the value opens an object (`{`) or array (`[`)
// that closes on a later line. Without this, a `var pet = { name: 'x' }`
// spread across multiple lines used to truncate at the first newline.
func parseVariableDecl(lines []string, startIdx int, decorators []string) (parsedVariable, int) {
	firstLine := lines[startIdx]
	first := strings.TrimSpace(firstLine)
	depth := parenBracketDepth(first) + braceDepth(first)

	if depth <= 0 {
		return parseVariable(first, decorators), startIdx + 1
	}

	// Value opens a block; reassemble until depth returns to zero.
	joined := []string{first}
	i := startIdx + 1
	for depth > 0 && i < len(lines) {
		t := strings.TrimSpace(lines[i])
		joined = append(joined, t)
		depth += parenBracketDepth(t) + braceDepth(t)
		i++
	}

	combined := strings.Join(joined, " ")
	// Collapse runs of whitespace inside the reassembled expression so
	// the captured value is a single readable line.
	combined = strings.Join(strings.Fields(combined), " ")
	return parseVariable(combined, decorators), i
}

// braceDepth mirrors parenBracketDepth but counts curly braces.
func braceDepth(s string) int {
	depth := 0
	for _, ch := range s {
		switch ch {
		case '{':
			depth++
		case '}':
			depth--
		}
	}
	return depth
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
	r.condition = extractCondition(joinDeclHeader(lines, startIdx))
	r.parent = extractFieldValue(body, "parent")
	r.dependsOn = extractDependsOn(body)
	r.tags = extractTags(body)

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

	mod.condition = extractCondition(joinDeclHeader(lines, startIdx))
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

// joinDeclHeader reassembles the declaration header — everything from
// startIdx up to but not including the body's opening `{` — into a
// single line. This lets extractCondition see the whole `if (...)`
// clause even when it spans several source lines, e.g.:
//
//	resource foo 'Type@ver' = if (
//	  expr1 &&
//	  expr2
//	) { ... }
//
// The `{` is only counted as the body opener once paren/bracket depth
// returns to zero.
func joinDeclHeader(lines []string, startIdx int) string {
	var parts []string
	parenDepth := 0
	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		end := -1
		for j, ch := range line {
			switch ch {
			case '(', '[':
				parenDepth++
			case ')', ']':
				parenDepth--
			case '{':
				if parenDepth == 0 {
					end = j
				}
			}
			if end >= 0 {
				break
			}
		}
		if end >= 0 {
			parts = append(parts, line[:end])
			break
		}
		parts = append(parts, line)
	}
	return strings.Join(parts, " ")
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
// the braces, or empty string if the field is not found. The opening `{`
// may be on the same line as the field name or on a subsequent line (both
// are valid Bicep):
//
//	params: { foo: 'x' }
//	params:
//	{
//	  foo: 'x'
//	}
func extractFieldBlock(body string, fieldName string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, fieldName+":") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(trimmed, fieldName+":"))
		startIdx := i
		// When the value is empty on this line, the `{` must be on a
		// later (non-blank) line; advance to it.
		if rest == "" {
			j := i + 1
			for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
				j++
			}
			if j >= len(lines) || !strings.HasPrefix(strings.TrimSpace(lines[j]), "{") {
				continue
			}
			startIdx = j
		} else if !strings.HasPrefix(rest, "{") {
			continue
		}
		block, _ := extractBlock(lines, startIdx)
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

// parseBicepObject takes the body of a Bicep object (text between the
// outer braces) and returns it as a key/value map. Nested objects
// become nested maps, arrays become slices, single-/double-quoted
// scalars are unquoted, and anything else (booleans, numbers, function
// calls, expressions) is kept in its raw text form so policy code can
// pattern-match on it.
//
// This is a deliberately small parser, not a full Bicep lexer: it
// honors string literals, `// ...` line comments, and brace/bracket/
// paren nesting when splitting top-level entries, which covers the
// shapes that show up in real-world `properties:` and `params:`
// blocks. Anything it can't parse cleanly falls back to a string —
// audits can still match on the text.
func parseBicepObject(body string) map[string]any {
	entries := splitTopLevelEntries(body)
	out := make(map[string]any, len(entries))
	for _, entry := range entries {
		key, value, ok := splitFirstColon(entry)
		if !ok {
			continue
		}
		out[strings.TrimSpace(key)] = parseBicepValue(strings.TrimSpace(value))
	}
	return out
}

func parseBicepValue(v string) any {
	if v == "" {
		return ""
	}
	switch v[0] {
	case '{':
		return parseBicepObject(stripOuter(v, '{', '}'))
	case '[':
		return parseBicepArray(stripOuter(v, '[', ']'))
	case '\'', '"':
		if len(v) >= 2 && v[len(v)-1] == v[0] {
			return v[1 : len(v)-1]
		}
	}
	return v
}

func parseBicepArray(body string) []any {
	entries := splitTopLevelEntries(body)
	out := make([]any, 0, len(entries))
	for _, e := range entries {
		out = append(out, parseBicepValue(strings.TrimSpace(e)))
	}
	return out
}

func splitFirstColon(s string) (string, string, bool) {
	depth := 0
	inStr := byte(0)
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inStr != 0 {
			if ch == inStr {
				inStr = 0
			}
			continue
		}
		switch ch {
		case '\'', '"':
			inStr = ch
		case '{', '[', '(':
			depth++
		case '}', ']', ')':
			depth--
		case ':':
			if depth == 0 {
				return s[:i], s[i+1:], true
			}
		}
	}
	return "", "", false
}

func stripOuter(s string, open, close byte) string {
	if len(s) < 2 || s[0] != open {
		return s
	}
	if s[len(s)-1] != close {
		return s[1:]
	}
	return s[1 : len(s)-1]
}

func splitTopLevelEntries(body string) []string {
	var entries []string
	var current strings.Builder
	depth := 0
	inStr := byte(0)
	flush := func() {
		s := strings.TrimSpace(current.String())
		if s != "" {
			entries = append(entries, s)
		}
		current.Reset()
	}
	i := 0
	for i < len(body) {
		ch := body[i]
		if inStr != 0 {
			current.WriteByte(ch)
			if ch == inStr {
				inStr = 0
			}
			i++
			continue
		}
		// Skip `// ...` line comments at top level so they don't leak
		// into the next entry's key or value.
		if depth == 0 && ch == '/' && i+1 < len(body) && body[i+1] == '/' {
			for i < len(body) && body[i] != '\n' {
				i++
			}
			continue
		}
		switch ch {
		case '\'', '"':
			inStr = ch
			current.WriteByte(ch)
		case '{', '[', '(':
			depth++
			current.WriteByte(ch)
		case '}', ']', ')':
			depth--
			current.WriteByte(ch)
		case '\n', ',':
			if depth == 0 {
				flush()
			} else {
				current.WriteByte(ch)
			}
		default:
			current.WriteByte(ch)
		}
		i++
	}
	flush()
	return entries
}

// tagsEntryRe matches one `key: 'value'` line inside a `tags: { ... }` block.
// Keys can be bare identifiers (`env`) or single-quoted strings (`'env-1'`);
// only literal single-quoted values are captured — expression-valued tags
// like `env: parameters('env')` are skipped because the dict shape on the
// resource gives audits a way to reach them in raw form.
var tagsEntryRe = regexp.MustCompile(`(?m)^\s*(?:'([^']+)'|(\w[\w-]*))\s*:\s*'([^']*)'\s*,?\s*$`)

// extractTags pulls a `tags: { ... }` block out of a resource body and
// returns the literal key/value pairs as a map. Expression-valued tags are
// dropped; the resource's `properties` dict still surfaces the raw text.
func extractTags(body string) map[string]string {
	raw := extractFieldBlock(body, "tags")
	if raw == "" {
		return nil
	}
	tags := map[string]string{}
	for _, m := range tagsEntryRe.FindAllStringSubmatch(raw, -1) {
		key := m[1]
		if key == "" {
			key = m[2]
		}
		tags[key] = m[3]
	}
	if len(tags) == 0 {
		return nil
	}
	return tags
}

var dependsOnHeaderRe = regexp.MustCompile(`(?m)dependsOn\s*:\s*\[`)

// extractDependsOn finds a `dependsOn: [ ... ]` block and returns the
// raw entries. Bracket depth is tracked manually so the matcher survives
// nested-bracket expressions like `dependsOn: [ foo[0], bar ]` that the
// prior regex form (`\[([^\]]*)\]`) terminated on the first inner `]`.
func extractDependsOn(body string) []string {
	loc := dependsOnHeaderRe.FindStringIndex(body)
	if loc == nil {
		return nil
	}
	// loc[1] points just past the opening `[`.
	start := loc[1]
	depth := 1
	end := -1
	for i := start; i < len(body); i++ {
		switch body[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				end = i
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		return nil
	}
	inner := body[start:end]

	var deps []string
	for _, part := range strings.Split(inner, "\n") {
		part = strings.TrimSpace(part)
		part = strings.TrimSuffix(part, ",")
		part = strings.TrimSpace(part)
		if part != "" {
			deps = append(deps, part)
		}
	}
	return deps
}
