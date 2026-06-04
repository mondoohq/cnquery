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
	types       []parsedType
	functions   []parsedFunction
	imports     []parsedImport
	metadata    map[string]string
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
	loop        loopInfo
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
	scope        string
	body         string
	tags         map[string]string
	dependsOn    []string
	decorators   []string
	loop         loopInfo
	nested       []parsedResource
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
	loop           loopInfo
}

type parsedOutput struct {
	name        string
	typ         string
	expression  string
	description string
	loop        loopInfo
}

// loopInfo captures a Bicep `for`-loop on a declaration. Bicep iterates
// resources, modules, outputs, and variables with
// `[for <iterator> in <expression>: <body>]` (or the indexed
// `[for (<iterator>, <indexVar>) in <expression>: <body>]` form). When a
// declaration's value is such a loop, isLoop is true and the iterator,
// optional indexVar, and collection expression are extracted; body holds
// the text after the header colon (the per-iteration object or expression).
type loopInfo struct {
	isLoop     bool
	iterator   string
	indexVar   string
	expression string
	body       string
}

type parsedType struct {
	name        string
	definition  string
	description string
	exported    bool
	decorators  []string

	// kind classifies the definition: "object", "union", "array", "tuple"
	// (folded into "array"), "primitive", or "alias".
	kind string
	// unionMembers holds the literal members of a union type, each kept
	// exactly as written (including quotes). Empty for non-unions.
	unionMembers []string
	// properties holds the name/type pairs of an object type. Empty otherwise.
	properties []parsedTypeProperty
	// discriminator is the key captured from an `@discriminator('<key>')`
	// decorator; empty when no such decorator is present.
	discriminator string
}

// parsedTypeProperty is one `name: type` member of an object-typed declaration.
type parsedTypeProperty struct {
	name string
	typ  string
}

type parsedFunction struct {
	name        string
	parameters  map[string]string
	returnType  string
	expression  string
	description string
	decorators  []string
}

type parsedImport struct {
	source    string
	symbols   []string
	namespace string
	wildcard  bool
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
	exportDecRe    = regexp.MustCompile(`@export\(\)`)

	// discriminatorDecRe captures the key argument of an
	// `@discriminator('<key>')` decorator on a tagged-union type.
	discriminatorDecRe = regexp.MustCompile(`@discriminator\(\s*'([^']*)'\s*\)`)

	// typeRe matches the header of a `type Name = <definition>` statement.
	// The definition (everything after the first `=`) is captured raw;
	// unions, object types, and references all flow through as text.
	typeRe = regexp.MustCompile(`(?s)^type\s+(\w+)\s*=\s*(.+)$`)

	// funcHeadRe matches just the `func name(` prefix of a function
	// declaration. The parameter list, return type, and body are split out
	// with depth-aware scanning rather than a single regex, so object-typed
	// parameters (`opts { name: string }`, which contain `)`/`,`/`{}`) and
	// array/object return types (`string[]`, `{ a: int }`) parse correctly.
	funcHeadRe = regexp.MustCompile(`(?s)^func\s+(\w+)\s*\(`)

	// metadataRe matches a `metadata <name> = '<literal>'` entry. Only
	// literal single-quoted values are captured; expression-/object-valued
	// metadata is skipped (mirrors the tag-extraction behavior).
	metadataRe = regexp.MustCompile(`(?m)^metadata\s+(\w+)\s*=\s*'([^']*)'\s*$`)

	// usingRe matches the `using '<target>'` statement at the head of a
	// `.bicepparam` file. The target is the referenced template
	// ('./main.bicep', 'none', or a registry/template-spec ref like 'br:...').
	usingRe = regexp.MustCompile(`(?m)^\s*using\s+'([^']*)'`)

	// paramAssignRe matches a `.bicepparam` parameter assignment
	// `param <name> = <value>`. Unlike a `.bicep` `param` declaration there
	// is no type between the name and `=`, so this deliberately does not reuse
	// paramRe. The right-hand side is captured raw (everything after `=`).
	paramAssignRe = regexp.MustCompile(`^param\s+(\w+)\s*=\s*(.+)$`)
)

// parseBicepParam parses a `.bicepparam` parameter file into the `using`
// target and a name->value map. The right-hand-side value text is stored
// EXACTLY as written, NOT quote-stripped: a literal stays quoted
// (`'Standard_LRS'`) and an expression stays bare
// (`resourceGroup().location`, `readEnvironmentVariable('ADMIN_PW')`). This
// preserves the literal-vs-expression distinction audits care about, and
// intentionally differs from `bicep.parameter.defaultValue`, which strips
// quotes.
//
// The tokenizer keys off the leading keyword of each statement, so `using`
// and `param` assignments are dispatched the same brace/string-aware way as
// `.bicep` constructs; multi-line object/array values are reassembled into a
// single collapsed-whitespace value.
func parseBicepParam(content string) (string, map[string]string) {
	var using string
	if m := usingRe.FindStringSubmatch(content); len(m) > 1 {
		using = m[1]
	}

	params := map[string]string{}
	for _, stmt := range tokenizeBicep(content) {
		if stmt.keyword != "param" {
			continue
		}
		stmtLines := strings.Split(stmt.text, "\n")
		// Reassemble multi-line values (object/array RHS) into one line with
		// runs of whitespace collapsed, mirroring parseVariableDecl.
		combined := strings.Join(strings.Fields(strings.Join(stmtLines, " ")), " ")
		m := paramAssignRe.FindStringSubmatch(strings.TrimSpace(combined))
		if len(m) < 3 {
			continue
		}
		params[m[1]] = strings.TrimSpace(m[2])
	}

	return using, params
}

func parseBicep(content string) *parsedBicepFile {
	result := &parsedBicepFile{}

	// Target scope
	if m := targetScopeRe.FindStringSubmatch(content); len(m) > 1 {
		result.targetScope = m[1]
	}

	// Walk the file once into brace/bracket/string-aware statements, then
	// dispatch each one to the construct-specific parser. Each statement's
	// `text` is a guaranteed-complete construct (its delimiters balance), so
	// the per-construct helpers receive the whole body rather than re-walking
	// the raw line slice.
	for _, stmt := range tokenizeBicep(content) {
		// The Decl helpers still operate on a line slice indexed from the
		// start of the statement. Because `stmt.text` is already a complete
		// construct, splitting it and starting at index 0 yields exactly the
		// lines the helper would have seen in the old line-walking loop.
		stmtLines := strings.Split(stmt.text, "\n")
		firstLine := strings.TrimSpace(stmtLines[0])

		switch stmt.keyword {
		case "param":
			result.parameters = append(result.parameters, parseParameter(firstLine, stmt.decorators))
		case "var":
			v, _ := parseVariableDecl(stmtLines, 0, stmt.decorators)
			result.variables = append(result.variables, v)
		case "resource":
			if res, _ := parseResourceDecl(stmtLines, 0, stmt.decorators); res != nil {
				result.resources = append(result.resources, *res)
			}
		case "module":
			if mod, _ := parseModuleDecl(stmtLines, 0, stmt.decorators); mod != nil {
				result.modules = append(result.modules, *mod)
			}
		case "output":
			// Reassemble multi-line output values (a looped output's
			// `[for ... : ...]` spans lines) into a single collapsed-whitespace
			// line so the value regex and loop detector see the whole RHS.
			outLine := firstLine
			if len(stmtLines) > 1 {
				outLine = strings.Join(strings.Fields(strings.Join(stmtLines, " ")), " ")
			}
			result.outputs = append(result.outputs, parseOutput(outLine, stmt.decorators))
		case "type":
			if t, ok := parseTypeDecl(stmt.text, stmt.decorators); ok {
				result.types = append(result.types, t)
			}
		case "func":
			if fn, ok := parseFunctionDecl(stmt.text, stmt.decorators); ok {
				result.functions = append(result.functions, fn)
			}
		case "import":
			if imp, ok := parseImportDecl(stmt.text); ok {
				result.imports = append(result.imports, imp)
			}
		case "metadata":
			if name, value, ok := parseMetadataDecl(stmt.text); ok {
				if result.metadata == nil {
					result.metadata = map[string]string{}
				}
				result.metadata[name] = value
			}
		default:
			// targetScope (already captured above) and unknown leading
			// tokens carry no per-construct parsing here; they are retained
			// as statements purely so the tokenizer never drops a line.
		}
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
		expr := strings.TrimSpace(m[2])
		// A looped variable (`var names = [for i in range(0, 3): 'item-${i}']`)
		// builds an array. Intentionally store the per-iteration value
		// expression in `expression` (here `'item-${i}'`), not the raw
		// `[for ...]` text, and record the loop header separately. This keeps
		// `expression` describing the value each iteration produces.
		if loop := detectLoop(expr); loop.isLoop {
			v.loop = loop
			v.expression = loop.body
		} else {
			v.expression = expr
		}
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
// Depth tracking is string-aware so braces or brackets that live inside
// a Bicep string literal (e.g. `var x = { msg: 'closing brace }' }`)
// don't confuse the counter.
func parseVariableDecl(lines []string, startIdx int, decorators []string) (parsedVariable, int) {
	first := strings.TrimSpace(lines[startIdx])
	st := scanState{}
	st.feed(first)

	if st.totalDepth() <= 0 {
		return parseVariable(first, decorators), startIdx + 1
	}

	// Value opens a block; reassemble until depth returns to zero.
	joined := []string{first}
	i := startIdx + 1
	for st.totalDepth() > 0 && i < len(lines) {
		t := strings.TrimSpace(lines[i])
		joined = append(joined, t)
		st.feed(t)
		i++
	}

	combined := strings.Join(joined, " ")
	// Collapse runs of whitespace inside the reassembled expression so
	// the captured value is a single readable line.
	combined = strings.Join(strings.Fields(combined), " ")
	return parseVariable(combined, decorators), i
}

// declValueAfterEquals returns the text following the top-level `=` of a
// declaration that spans lines[startIdx:] — i.e. everything to the right of
// the `=` in `resource foo 'Type@ver' = <value>` or `module m 'src' = <value>`.
// The whole statement is reassembled and the first `=` that sits outside any
// string, paren, bracket, or brace is taken as the assignment operator (so an
// `=` inside the resource type string or a condition expression is ignored).
// Returns "" when no top-level `=` is found.
func declValueAfterEquals(lines []string, startIdx int) string {
	full := strings.Join(lines[startIdx:], "\n")
	st := scanState{}
	for i := 0; i < len(full); {
		if st.inStr == 0 && !st.inMulti && full[i] == '=' &&
			st.paren == 0 && st.bracket == 0 && st.brace == 0 {
			// Guard against `==`/`=>` which can appear in conditions; a real
			// assignment `=` is followed by whitespace or a value char, not
			// another `=` or `>`.
			if i+1 < len(full) && (full[i+1] == '=' || full[i+1] == '>') {
				i = st.stepAt(full, i)
				continue
			}
			return strings.TrimSpace(full[i+1:])
		}
		i = st.stepAt(full, i)
	}
	return ""
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
	r.condition = extractCondition(joinDeclHeader(lines, startIdx))

	// endIdx is where the next statement begins; with the tokenizer feeding a
	// single complete statement it is the end of the slice.
	_, endIdx := extractBlock(lines, startIdx)

	// When deployed via a `for`-loop the body object lives inside
	// `[for <hdr>: { ... }]`. Detect the loop and parse the per-iteration
	// object body the same way a plain resource body is parsed.
	r.loop = detectLoop(declValueAfterEquals(lines, startIdx))
	bodyLines := lines
	bodyStart := startIdx
	if r.loop.isLoop {
		bodyLines = strings.Split(r.loop.body, "\n")
		bodyStart = 0
	}

	body, _ := extractBlock(bodyLines, bodyStart)
	r.body = body

	// Extract common fields from body
	r.name = extractFieldValue(body, "name")
	r.location = extractFieldValue(body, "location")
	r.parent = extractFieldValue(body, "parent")
	r.scope = extractFieldValue(body, "scope")
	r.dependsOn = extractDependsOn(body)
	r.tags = extractTags(body)

	// A resource may declare child resources inside its body. Each nested
	// declaration is parsed with the same logic as a top-level resource and
	// has its `parent` set to this resource's symbolic name.
	r.nested = extractNestedResources(body, r.symbolicName)

	return r, endIdx
}

// extractNestedResources finds `resource <sym> '<type>'( existing)? =`
// declarations that sit at the top level of the parent's body (one brace
// deep, relative to the parent's outer braces) and parses each one
// recursively with the same logic as a top-level resource. The body text
// passed in is the parent's full `{ ... }` block including its outer braces,
// so child declarations live at brace depth 1. The parent's symbolic name is
// recorded on each child's `parent` field.
func extractNestedResources(body, parentSymbolic string) []parsedResource {
	var nested []parsedResource
	st := scanState{}
	for i := 0; i < len(body); {
		// We only consider a `resource` keyword when it begins a statement at
		// brace depth 1 (directly inside the parent body) and outside any
		// string/paren/bracket. depthBefore is captured before stepping.
		if st.inStr == 0 && !st.inMulti && st.brace == 1 && st.paren == 0 && st.bracket == 0 &&
			isStatementStart(body, i) && hasKeywordAt(body, i, "resource") {
			// Reassemble the nested declaration from here until its delimiters
			// balance again, then hand the statement text to the recursive
			// resource parser.
			stmtEnd := scanStatementEnd(body[i:]) + i
			stmt := body[i:stmtEnd]
			stmtLines := strings.Split(stmt, "\n")
			if child, _ := parseResourceDecl(stmtLines, 0, nil); child != nil {
				child.parent = parentSymbolic
				nested = append(nested, *child)
			}
			// Resume scanning after the consumed statement, keeping the depth
			// state consistent by feeding the skipped text.
			for j := i; j < stmtEnd; {
				j = st.stepAt(body, j)
			}
			i = stmtEnd
			continue
		}
		i = st.stepAt(body, i)
	}
	return nested
}

// isStatementStart reports whether position i in s begins a token — i.e. the
// preceding non-ignored byte is a newline, brace, or the start of input. This
// keeps `extractNestedResources` from matching a `resource` substring that is
// part of a larger identifier or appears mid-line (e.g. an expression).
func isStatementStart(s string, i int) bool {
	for j := i - 1; j >= 0; j-- {
		c := s[j]
		if c == ' ' || c == '\t' || c == '\r' {
			continue
		}
		return c == '\n' || c == '{' || c == '}'
	}
	return true
}

// hasKeywordAt reports whether the identifier starting at i in s is exactly
// `kw` (followed by whitespace, not more identifier characters).
func hasKeywordAt(s string, i int, kw string) bool {
	if i+len(kw) > len(s) {
		return false
	}
	if s[i:i+len(kw)] != kw {
		return false
	}
	next := i + len(kw)
	if next >= len(s) {
		return false
	}
	c := s[next]
	return c == ' ' || c == '\t'
}

// scanStatementEnd returns the index just past the end of the statement at the
// start of s. It mirrors the tokenizer: consume the first line, then keep
// pulling continuation lines until the running depth (string-aware) returns to
// zero. This way a `= if (cond) { ... }` header whose parens close before the
// body brace opens doesn't terminate the statement early, and a multi-line
// `= { ... }` body is consumed whole.
func scanStatementEnd(s string) int {
	st := scanState{}
	// Consume the first line.
	nl := strings.IndexByte(s, '\n')
	if nl < 0 {
		return len(s)
	}
	st.feed(s[:nl])
	pos := nl + 1
	if st.totalDepth() <= 0 {
		return nl
	}
	for pos < len(s) {
		next := strings.IndexByte(s[pos:], '\n')
		var line string
		if next < 0 {
			line = s[pos:]
		} else {
			line = s[pos : pos+next]
		}
		st.feed(line)
		if next < 0 {
			return len(s)
		}
		pos += next + 1
		if st.totalDepth() <= 0 {
			return pos - 1
		}
	}
	return len(s)
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
	_, endIdx := extractBlock(lines, startIdx)

	// As with resources, a looped module wraps its body object in
	// `[for <hdr>: { ... }]`; feed the loop body to the body parser so
	// `name`/`scope`/`params` extraction still works.
	mod.loop = detectLoop(declValueAfterEquals(lines, startIdx))
	bodyLines := lines
	bodyStart := startIdx
	if mod.loop.isLoop {
		bodyLines = strings.Split(mod.loop.body, "\n")
		bodyStart = 0
	}

	body, _ := extractBlock(bodyLines, bodyStart)
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
		expr := strings.TrimSpace(m[3])
		// A looped output (`output ids array = [for sa in sas: sa.id]`)
		// produces an array. Intentionally store the per-iteration value
		// expression in `expression` (here `sa.id`), not the raw `[for ...]`
		// text, and record the loop header separately. This keeps `expression`
		// describing the value each iteration produces.
		if loop := detectLoop(expr); loop.isLoop {
			o.loop = loop
			o.expression = loop.body
		} else {
			o.expression = expr
		}
	}
	decText := strings.Join(decorators, "\n")
	if m := descDecRe.FindStringSubmatch(decText); len(m) > 1 {
		o.description = m[1]
	}
	return o
}

// parseTypeDecl handles a `type Name = <definition>` statement. The whole
// statement body is passed in (it may span multiple lines for object types),
// so runs of whitespace in the captured definition are collapsed into single
// spaces for a readable single-line value. The `@export()` decorator marks
// the type as shared/exported, and `@description(...)` is captured.
func parseTypeDecl(text string, decorators []string) (parsedType, bool) {
	trimmed := strings.TrimSpace(text)
	m := typeRe.FindStringSubmatch(trimmed)
	if len(m) < 3 {
		return parsedType{}, false
	}
	t := parsedType{
		name:       m[1],
		definition: strings.Join(strings.Fields(m[2]), " "),
		decorators: decorators,
	}
	decText := strings.Join(decorators, "\n")
	if dm := descDecRe.FindStringSubmatch(decText); len(dm) > 1 {
		t.description = dm[1]
	}
	t.exported = exportDecRe.MatchString(decText)
	if dm := discriminatorDecRe.FindStringSubmatch(decText); len(dm) > 1 {
		t.discriminator = dm[1]
	}

	// Decompose the definition into a kind plus (for objects/unions) its
	// structured members. Classification runs on the RAW (pre-collapse)
	// right-hand side because object properties may be newline-separated with
	// no commas — collapsing whitespace would merge them — and the splitter is
	// already string/bracket-aware so newlines and nested braces are handled.
	t.kind, t.unionMembers, t.properties = classifyTypeDefinition(m[2])
	return t, true
}

// builtinTypeNames is the set of Bicep built-in primitive types a bare type
// identifier may name. A definition that is exactly one of these classifies as
// "primitive"; any other bare identifier classifies as "alias" (it names
// another user-defined type).
var builtinTypeNames = map[string]bool{
	"string":       true,
	"int":          true,
	"bool":         true,
	"object":       true,
	"array":        true,
	"secureString": true,
	"securestring": true,
	"secureObject": true,
	"secureobject": true,
}

// classifyTypeDefinition inspects a type's raw right-hand-side definition and
// returns its kind plus, for object/union types, the structured members:
//
//   - a trimmed `{ ... }` body  -> "object", with each `key: type` pair parsed
//   - a top-level `|`           -> "union", with members split on that `|`
//   - a leading `[`             -> "array" (a `[...]` tuple is folded in)
//   - a `[]` suffix             -> "array"
//   - a bare built-in name      -> "primitive"
//   - any other bare identifier -> "alias"
//
// Splitting (object pairs, union members) is depth-aware via scanState so
// commas/pipes inside nested `{}`/`[]`/`<>`/strings don't split early.
func classifyTypeDefinition(def string) (string, []string, []parsedTypeProperty) {
	trimmed := strings.TrimSpace(def)
	if trimmed == "" {
		return "", nil, nil
	}

	// Union type: members separated by a top-level `|`. Checked before the
	// object case because a discriminated tagged union's first member is itself
	// an object (`{ kind: 'circle', ... } | { kind: 'square', ... }`), so a
	// leading `{` does not imply a plain object type. Members keep their raw
	// form (including the surrounding whitespace collapsed to a single line).
	if members := splitTopLevelPipes(trimmed); len(members) > 1 {
		out := make([]string, 0, len(members))
		for _, m := range members {
			m = strings.Join(strings.Fields(m), " ")
			if m != "" {
				out = append(out, m)
			}
		}
		return "union", out, nil
	}

	// Object type: `{ name: string, tier: sku }`.
	if trimmed[0] == '{' {
		return "object", nil, parseTypeObjectProperties(stripOuter(trimmed, '{', '}'))
	}

	// Array/tuple: a `[...]` tuple or a `<type>[]` suffix.
	if trimmed[0] == '[' || strings.HasSuffix(trimmed, "[]") {
		return "array", nil, nil
	}

	// A bare identifier: a built-in primitive, otherwise an alias for another
	// type. Anything else (a function-shaped or otherwise complex definition)
	// also falls through to "alias".
	if builtinTypeNames[trimmed] {
		return "primitive", nil, nil
	}
	return "alias", nil, nil
}

// parseTypeObjectProperties splits an object type body (the text between the
// outer braces) into its `name: type` members. Entries are split on top-level
// commas and newlines (string/bracket/brace-aware) so a nested object property
// like `nested: { a: int, b: int }` stays intact; a trailing `?` optional
// marker on a key is stripped.
func parseTypeObjectProperties(body string) []parsedTypeProperty {
	var props []parsedTypeProperty
	for _, entry := range splitTopLevelEntries(body) {
		key, value, ok := splitFirstColon(entry)
		if !ok {
			continue
		}
		name := strings.TrimSpace(key)
		name = strings.TrimSuffix(name, "?")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		props = append(props, parsedTypeProperty{
			name: name,
			typ:  strings.TrimSpace(value),
		})
	}
	return props
}

// splitTopLevelPipes splits s on `|` characters that sit outside any string,
// bracket, brace, or paren and outside a triple-quoted string — so a `|` inside
// a nested object/array or a string literal doesn't split the union.
func splitTopLevelPipes(s string) []string {
	var parts []string
	st := scanState{}
	start := 0
	for i := 0; i < len(s); {
		if s[i] == '|' && st.inStr == 0 && !st.inMulti &&
			st.paren == 0 && st.bracket == 0 && st.brace == 0 {
			parts = append(parts, s[start:i])
			i++
			start = i
			continue
		}
		i = st.stepAt(s, i)
	}
	return append(parts, s[start:])
}

// parseFunctionDecl handles a `func name(p1 t1, p2 t2) returnType => expr`
// declaration. The parameter list is split into a name->type map, and the
// body expression after `=>` is captured raw (whitespace collapsed).
func parseFunctionDecl(text string, decorators []string) (parsedFunction, bool) {
	trimmed := strings.TrimSpace(text)
	head := funcHeadRe.FindStringSubmatchIndex(trimmed)
	if head == nil {
		return parsedFunction{}, false
	}
	name := trimmed[head[2]:head[3]]
	// The full match ends on the opening `(` of the parameter list.
	openParen := head[1] - 1
	closeParen := matchingParenIndex(trimmed, openParen)
	if closeParen < 0 {
		return parsedFunction{}, false
	}
	paramList := trimmed[openParen+1 : closeParen]

	// After the parameter list comes `<returnType> => <expression>`. The
	// first `=>` is the function arrow; any later `=>` belongs to a lambda
	// inside the body, so splitting on the first occurrence is correct.
	rest := strings.TrimSpace(trimmed[closeParen+1:])
	arrow := strings.Index(rest, "=>")
	if arrow < 0 {
		return parsedFunction{}, false
	}

	fn := parsedFunction{
		name:       name,
		parameters: parseFunctionParams(paramList),
		returnType: strings.TrimSpace(rest[:arrow]),
		expression: strings.Join(strings.Fields(rest[arrow+2:]), " "),
		decorators: decorators,
	}
	decText := strings.Join(decorators, "\n")
	if dm := descDecRe.FindStringSubmatch(decText); len(dm) > 1 {
		fn.description = dm[1]
	}
	return fn, true
}

// parseFunctionParams splits a `p1 t1, p2 t2` parameter list into a
// name->type map. Commas are split at top-level depth only, so an
// object-typed parameter such as `opts { name: string, tier: int }` is kept
// intact. Each entry is `name <type...>`; the first token is the name and the
// remainder (whitespace-collapsed) is the type. Returns nil for an empty list.
func parseFunctionParams(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	params := map[string]string{}
	for _, part := range splitTopLevelCommas(raw) {
		fields := strings.Fields(part)
		if len(fields) < 2 {
			continue
		}
		params[fields[0]] = strings.Join(fields[1:], " ")
	}
	if len(params) == 0 {
		return nil
	}
	return params
}

// matchingParenIndex returns the index of the `)` that closes the `(` at
// position open, using the shared string-aware lexer so parens inside string
// literals or nested brackets/braces don't throw off the count. Returns -1 if
// the paren is unbalanced.
func matchingParenIndex(s string, open int) int {
	st := scanState{}
	i := st.stepAt(s, open) // consume the opening '(' -> paren depth 1
	for i < len(s) {
		if st.paren == 0 {
			return i - 1
		}
		i = st.stepAt(s, i)
	}
	if st.paren == 0 {
		return len(s) - 1
	}
	return -1
}

// splitTopLevelCommas splits s on commas that sit outside any string,
// bracket, brace, or paren — so commas inside an object-typed parameter or a
// nested expression don't split the list.
func splitTopLevelCommas(s string) []string {
	var parts []string
	st := scanState{}
	start := 0
	for i := 0; i < len(s); {
		if s[i] == ',' && st.inStr == 0 && !st.inMulti &&
			st.paren == 0 && st.bracket == 0 && st.brace == 0 {
			parts = append(parts, s[start:i])
			i++
			start = i
			continue
		}
		i = st.stepAt(s, i)
	}
	return append(parts, s[start:])
}

var (
	// importNamedRe matches `import { a, b } from './x.bicep'`.
	importNamedRe = regexp.MustCompile(`(?s)^import\s*\{([^}]*)\}\s*from\s*'([^']+)'`)
	// importWildcardRe matches `import * as ns from './x.bicep'`.
	importWildcardRe = regexp.MustCompile(`^import\s*\*\s*as\s+(\w+)\s+from\s*'([^']+)'`)
	// importProviderRe matches a bare provider import like `import 'az@2.0.0'`.
	importProviderRe = regexp.MustCompile(`^import\s+'([^']+)'`)
)

// loopHeaderRe matches the `[for` opener of a loop value and the iteration
// variable form that follows. Group 1 captures the indexed
// `(item, index)` form's item, group 2 the index var, group 3 the single
// `item` form. Only the header up to the first identifier(s) and the
// trailing `in ` is matched; the `<expression>: <body>` tail is split out
// depth-aware so collection expressions and bodies that contain `:`/`]`
// inside strings, parens, or nested brackets parse correctly.
var loopHeaderRe = regexp.MustCompile(`^\[\s*for\s+(?:\(\s*(\w+)\s*,\s*(\w+)\s*\)|(\w+))\s+in\s+`)

// detectLoop inspects a declaration's value text (everything after the `=`)
// and, when it is a `for`-loop, returns the parsed loop header plus the
// per-iteration body. For a non-loop value it returns loopInfo{} with
// isLoop=false, so callers can fall back to their normal parse.
//
// The collection expression runs from `in` up to the top-level `:` that
// separates the loop header from its body; the body runs from that `:` to
// the matching closing `]`. Both boundaries are found with the shared
// string/bracket-aware scanState so a `:` or `]` inside a string literal,
// a `range(0, 3)` call, or a nested object/array doesn't split early.
func detectLoop(value string) loopInfo {
	trimmed := strings.TrimSpace(value)
	m := loopHeaderRe.FindStringSubmatchIndex(trimmed)
	if m == nil {
		return loopInfo{}
	}

	info := loopInfo{isLoop: true}
	// Submatch groups: [2:3]=indexed item, [4:5]=index var, [6:7]=single item.
	if m[2] >= 0 {
		info.iterator = trimmed[m[2]:m[3]]
		info.indexVar = trimmed[m[4]:m[5]]
	} else if m[6] >= 0 {
		info.iterator = trimmed[m[6]:m[7]]
	}

	// The collection expression starts right after the matched `... in ` prefix.
	exprStart := m[1]

	// Find the top-level `:` separating the loop header from the body, and
	// the matching closing `]`. Seed the scanner with the leading `[`
	// already consumed (bracket depth 1).
	st := scanState{bracket: 1}
	colon := -1
	closeBracket := -1
	for i := exprStart; i < len(trimmed); {
		if st.inStr == 0 && !st.inMulti {
			// The header/body separator is the first `:` at the loop's own
			// bracket depth (1) with no open paren/brace.
			if trimmed[i] == ':' && colon < 0 && st.bracket == 1 && st.paren == 0 && st.brace == 0 {
				colon = i
			}
		}
		prev := i
		i = st.stepAt(trimmed, i)
		if st.bracket == 0 {
			closeBracket = prev
			break
		}
	}

	if colon < 0 {
		// Malformed loop (no header colon found); treat as non-loop so the
		// caller's normal parse still runs.
		return loopInfo{}
	}
	if closeBracket < 0 {
		closeBracket = len(trimmed)
	}

	info.expression = strings.TrimSpace(trimmed[exprStart:colon])
	info.body = strings.TrimSpace(trimmed[colon+1 : closeBracket])
	return info
}

// parseImportDecl handles the three Bicep import forms:
//
//	import { typeA, funcB } from './shared.bicep'
//	import * as shared from './shared.bicep'
//	import 'az@2.0.0'
//
// The `from` target (or the bare provider string) becomes `source`; named
// imports populate `symbols`; a `* as ns` import sets `namespace` and
// `wildcard`.
func parseImportDecl(text string) (parsedImport, bool) {
	trimmed := strings.Join(strings.Fields(text), " ")

	if m := importWildcardRe.FindStringSubmatch(trimmed); len(m) > 2 {
		return parsedImport{
			source:    m[2],
			namespace: m[1],
			wildcard:  true,
		}, true
	}

	if m := importNamedRe.FindStringSubmatch(trimmed); len(m) > 2 {
		var symbols []string
		for _, s := range strings.Split(m[1], ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				symbols = append(symbols, s)
			}
		}
		return parsedImport{
			source:  m[2],
			symbols: symbols,
		}, true
	}

	if m := importProviderRe.FindStringSubmatch(trimmed); len(m) > 1 {
		return parsedImport{source: m[1]}, true
	}

	return parsedImport{}, false
}

// parseMetadataDecl handles a `metadata <name> = '<literal>'` entry. Only
// literal single-quoted values are captured; expression-/object-valued
// metadata returns ok=false and is skipped, mirroring tag extraction.
func parseMetadataDecl(text string) (string, string, bool) {
	trimmed := strings.TrimSpace(text)
	m := metadataRe.FindStringSubmatch(trimmed)
	if len(m) < 3 {
		return "", "", false
	}
	return m[1], m[2], true
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
// The `{` is only counted as the body opener when paren/bracket depth
// is zero and the cursor isn't inside a string literal.
func joinDeclHeader(lines []string, startIdx int) string {
	var parts []string
	st := scanState{}
	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		end := st.scanForBodyBrace(line)
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

// scanState is a tiny lexer state for tracking paren/bracket/brace
// depth across one or more lines of Bicep source while respecting
// string literals (single- and double-quoted, with `\<char>` escapes),
// triple-quoted multi-line strings (`”'…”'`), and `// ...` line
// comments. All of the bracket-balancing helpers in this package
// share it so brackets that live inside a string literal — including
// a `”'…}…”'` block that straddles multiple lines — can't fool
// the depth counter.
type scanState struct {
	paren   int
	bracket int
	brace   int
	inStr   byte // 0, '\'', or '"' — single-line string
	inMulti bool // true while inside a `'''…'''` multi-line string
}

func (s *scanState) totalDepth() int { return s.paren + s.bracket + s.brace }

// stepAt processes one position in `body` and returns the next index
// to resume from. It handles the full set of Bicep token states
// (single-line strings with `\<char>` escapes, triple-quoted
// multi-line strings, `// …` line comments, and bracket nesting) so
// every walker in this file can share the same lexer rules without
// duplicating them.
func (s *scanState) stepAt(body string, i int) int {
	if s.inMulti {
		if i+2 < len(body) && body[i] == '\'' && body[i+1] == '\'' && body[i+2] == '\'' {
			s.inMulti = false
			return i + 3
		}
		return i + 1
	}
	ch := body[i]
	if s.inStr != 0 {
		// Bicep string escapes: `\\`, `\'`, `\n`, `\$`, etc. The next
		// byte is literal regardless of what it is, so just skip it.
		if ch == '\\' && i+1 < len(body) {
			return i + 2
		}
		if ch == s.inStr {
			s.inStr = 0
		}
		return i + 1
	}
	// `// …` line comment — skip to end of line.
	if ch == '/' && i+1 < len(body) && body[i+1] == '/' {
		j := i
		for j < len(body) && body[j] != '\n' {
			j++
		}
		return j
	}
	// Triple-quoted multi-line string opener: enter `inMulti` and
	// hand back the index just past the `'''`.
	if ch == '\'' && i+2 < len(body) && body[i+1] == '\'' && body[i+2] == '\'' {
		s.inMulti = true
		return i + 3
	}
	switch ch {
	case '\'', '"':
		s.inStr = ch
	case '(':
		s.paren++
	case ')':
		s.paren--
	case '[':
		s.bracket++
	case ']':
		s.bracket--
	case '{':
		s.brace++
	case '}':
		s.brace--
	}
	return i + 1
}

// feed advances the state through one line of source, updating the
// running depth counters. Characters inside a string literal or after
// a top-level `//` comment marker do not affect depth.
func (s *scanState) feed(line string) {
	for i := 0; i < len(line); {
		i = s.stepAt(line, i)
	}
}

// scanForBodyBrace advances the state through one line and returns the
// byte index of the first `{` that lands outside any string and at
// paren/bracket depth 0 — i.e., the opener of a resource/module body.
// Returns -1 if no such brace is on the line. The state still tracks
// every other character so subsequent calls keep their depth context.
func (s *scanState) scanForBodyBrace(line string) int {
	for i := 0; i < len(line); {
		if s.inStr == 0 && !s.inMulti && line[i] == '{' && s.paren == 0 && s.bracket == 0 {
			return i
		}
		i = s.stepAt(line, i)
	}
	return -1
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
	// Bare `true`/`false` and numeric literals become typed values so policies
	// can compare them naturally (`== true`, `>= 30`) instead of against the
	// stringified form. Anything else (an expression like `resourceGroup().id`,
	// a bare symbol) is returned as-is.
	switch v {
	case "true":
		return true
	case "false":
		return false
	}
	if n, ok := parseBicepNumber(v); ok {
		return n
	}
	return v
}

// parseBicepNumber parses a bare integer or float literal into a float64 (the
// shape MQL uses for numbers inside a dict). Returns false for anything that
// isn't a plain decimal literal so expressions — and the special float forms
// `Inf`/`NaN` that `strconv.ParseFloat` would otherwise accept — stay raw
// strings.
func parseBicepNumber(v string) (float64, bool) {
	if v == "" {
		return 0, false
	}
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case c >= '0' && c <= '9', c == '.':
		case (c == '-' || c == '+') && i == 0:
		default:
			return 0, false
		}
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f, true
	}
	return 0, false
}

// stripLiteralQuotes removes the surrounding single quotes from a Bicep string
// literal (`'eastus'` -> `eastus`), leaving unquoted expressions
// (`resourceGroup().location`) untouched. Mirrors how parameter default values
// are unquoted, so equality checks like `name == "eastus"` work directly.
func stripLiteralQuotes(s string) string {
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
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
	st := scanState{}
	for i := 0; i < len(s); {
		if st.inStr == 0 && !st.inMulti && s[i] == ':' && st.totalDepth() == 0 {
			return s[:i], s[i+1:], true
		}
		i = st.stepAt(s, i)
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
	st := scanState{}
	flush := func() {
		s := strings.TrimSpace(current.String())
		if s != "" {
			entries = append(entries, s)
		}
		current.Reset()
	}
	i := 0
	for i < len(body) {
		// Special cases only fire when we're not inside any string,
		// and only at top-level depth so nested object/array entries
		// preserve their commas and newlines.
		if st.inStr == 0 && !st.inMulti && st.totalDepth() == 0 {
			// Top-level newline/comma terminates the current entry.
			if body[i] == '\n' || body[i] == ',' {
				flush()
				i++
				continue
			}
			// Drop `// …` line comments without copying them into the
			// running entry — without this they'd leak into the next
			// key or value.
			if body[i] == '/' && i+1 < len(body) && body[i+1] == '/' {
				for i+1 < len(body) && body[i+1] != '\n' {
					i++
				}
				i++
				continue
			}
		}
		next := st.stepAt(body, i)
		// Copy the bytes we just walked into the running entry —
		// escape sequences (`\'`, `\\`, …) are preserved verbatim
		// because stepAt walks the escape pair as a single step.
		current.WriteString(body[i:next])
		i = next
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
// raw entries. It walks the body via the shared `scanState` lexer so
// brackets that live inside string literals (`'[indexed]'`) or
// indexed expressions (`storageAccounts['blobServices']`) don't drop
// the closing `]` early.
func extractDependsOn(body string) []string {
	loc := dependsOnHeaderRe.FindStringIndex(body)
	if loc == nil {
		return nil
	}
	// loc[1] points just past the opening `[`. Seed the scanner with
	// that opener already consumed.
	start := loc[1]
	st := scanState{bracket: 1}
	end := -1
	i := start
	for i < len(body) {
		prev := i
		i = st.stepAt(body, i)
		if st.bracket == 0 {
			end = prev
			break
		}
	}
	if end < 0 {
		return nil
	}
	// splitTopLevelEntries handles both newline- and comma-separated
	// entries and shares the string-aware lexer, so a `]` inside a
	// string literal or an indexed expression never splits the list.
	deps := splitTopLevelEntries(body[start:end])
	if len(deps) == 0 {
		return nil
	}
	return deps
}
