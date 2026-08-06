// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"regexp"
	"sort"
	"strings"
)

// aide selection rule kinds, named after what the line does to coverage rather
// than after the punctuation that introduces it.
const (
	aideSelectionRecursive = "recursive"
	aideSelectionEquals    = "equals"
	aideSelectionNegative  = "negative"
)

// aideMaxGroupDepth bounds group resolution so a configuration defining a group
// in terms of itself cannot spin.
const aideMaxGroupDepth = 16

// aideMaxIncludeDepth bounds include recursion so a configuration including
// itself cannot spin.
const aideMaxIncludeDepth = 8

// aideSelectionRule is one selection line from an AIDE configuration.
type aideSelectionRule struct {
	Path       string
	Selection  string
	Expression string
	Attributes []string
	LineNumber int
	File       string
}

// aideConfig accumulates the state of an AIDE configuration as it is parsed.
// Includes are parsed into the same value, because a macro or group defined in
// one file is visible to every file parsed after it.
type aideConfig struct {
	Macros map[string]string
	Groups map[string]string
	Params map[string]string
	Rules  []aideSelectionRule
}

// aideConfigOptions are the settings AIDE recognizes as configuration rather
// than as attribute group definitions. A `key = value` line whose key is absent
// here is treated as a group definition, which is how AIDE itself distinguishes
// the two.
var aideConfigOptions = map[string]struct{}{
	"acl_no_symlink_follow":       {},
	"config_version":              {},
	"database":                    {},
	"database_add_metadata":       {},
	"database_attrs":              {},
	"database_gzip":               {},
	"database_in":                 {},
	"database_new":                {},
	"database_out":                {},
	"grouped":                     {},
	"gzip_dbout":                  {},
	"log_level":                   {},
	"num_workers":                 {},
	"report_append":               {},
	"report_base16":               {},
	"report_detailed_init":        {},
	"report_force_attrs":          {},
	"report_grouped":              {},
	"report_ignore_added_attrs":   {},
	"report_ignore_changed_attrs": {},
	"report_ignore_e2fsattrs":     {},
	"report_ignore_removed_attrs": {},
	"report_level":                {},
	"report_quiet":                {},
	"report_summarize_changes":    {},
	"report_url":                  {},
	"root_prefix":                 {},
	"summarize_changes":           {},
	"syslog_format":               {},
	"verbose":                     {},
	"warn_dead_symlinks":          {},
	"warn_unrestricted_rules":     {},
}

var (
	aideMacroRefRegex = regexp.MustCompile(`@@\{([A-Za-z_][A-Za-z0-9_]*)\}`)
	aideDirectiveArgs = regexp.MustCompile(`\s+`)
)

func newAideConfig() *aideConfig {
	return &aideConfig{
		Macros: map[string]string{},
		Groups: map[string]string{},
		Params: map[string]string{},
		Rules:  []aideSelectionRule{},
	}
}

// aideIncludeFile is one file an include target expanded to.
type aideIncludeFile struct {
	Path    string
	Content string
}

// aideIncludeResolver reads an include target, returning the files it expands to
// in the order AIDE would read them. A target naming a directory expands to its
// entries; one naming a file expands to just that file.
type aideIncludeResolver func(target string) []aideIncludeFile

// parseAideConfig folds one configuration file into cfg, following @@include
// directives through resolve at the point they appear. Reading files stays with
// the caller through resolve, which keeps the parser pure and lets a test drive
// the include graph without a filesystem.
//
// The recursion is positional rather than batched at the end of the file,
// because a macro or group is only visible to the lines parsed after it, so
// where an include sits changes what it sees.
//
// @@ifdef, @@ifndef, @@else, and @@endif are evaluated against the macros
// defined so far. @@ifhost cannot be evaluated without knowing the host, so its
// body is parsed rather than skipped, on the basis that reporting a rule that
// may not apply is safer than hiding one that does.
func parseAideConfig(cfg *aideConfig, filePath string, content string, depth int, resolve aideIncludeResolver) {
	// each entry records whether the enclosing conditional branch is being kept
	branches := []bool{}

	for i, rawLine := range strings.Split(content, "\n") {
		lineNumber := i + 1

		line := strings.TrimSpace(stripAideComment(rawLine))
		if line == "" {
			continue
		}

		// "@@{NAME}" opens a macro reference, not a directive, and a selection
		// line commonly starts with one
		if strings.HasPrefix(line, "@@") && !strings.HasPrefix(line, "@@{") {
			target := parseAideDirective(cfg, line, &branches)
			if target == "" || !aideBranchActive(branches) {
				continue
			}
			if resolve == nil || depth >= aideMaxIncludeDepth {
				continue
			}
			for _, included := range resolve(target) {
				parseAideConfig(cfg, included.Path, included.Content, depth+1, resolve)
			}
			continue
		}

		if !aideBranchActive(branches) {
			continue
		}

		line = expandAideMacros(cfg, line)

		if rule, ok := parseAideSelectionLine(cfg, line, filePath, lineNumber); ok {
			cfg.Rules = append(cfg.Rules, rule)
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}

		if _, isOption := aideConfigOptions[strings.ToLower(key)]; isOption {
			cfg.Params[strings.ToLower(key)] = value
			continue
		}
		cfg.Groups[key] = value
	}
}

// parseAideDirective handles an @@ line, returning an include path when the
// directive names one.
func parseAideDirective(cfg *aideConfig, line string, branches *[]bool) (include string) {
	fields := aideDirectiveArgs.Split(line, 3)
	directive := strings.ToLower(fields[0])

	arg := ""
	if len(fields) > 1 {
		arg = strings.TrimSpace(fields[1])
	}
	rest := ""
	if len(fields) > 2 {
		rest = strings.TrimSpace(fields[2])
	}

	switch directive {
	case "@@define":
		if arg != "" {
			cfg.Macros[arg] = expandAideMacros(cfg, rest)
		}

	case "@@undef":
		delete(cfg.Macros, arg)

	case "@@include", "@@x_include":
		// only the first argument names the path; @@x_include carries a regex
		// after it, and the path itself may be built from macros
		target := strings.TrimSpace(expandAideMacros(cfg, arg))
		if target != "" {
			return target
		}

	case "@@ifdef":
		_, defined := cfg.Macros[arg]
		*branches = append(*branches, defined)
		return ""

	case "@@ifndef":
		_, defined := cfg.Macros[arg]
		*branches = append(*branches, !defined)
		return ""

	case "@@ifhost", "@@ifnhost":
		// the host is not knowable here, so keep the body
		*branches = append(*branches, true)
		return ""

	case "@@else":
		if len(*branches) > 0 {
			(*branches)[len(*branches)-1] = !(*branches)[len(*branches)-1]
		}
		return ""

	case "@@endif":
		if len(*branches) > 0 {
			*branches = (*branches)[:len(*branches)-1]
		}
		return ""
	}

	return ""
}

// aideBranchActive reports whether every enclosing conditional is being kept.
func aideBranchActive(branches []bool) bool {
	for _, keep := range branches {
		if !keep {
			return false
		}
	}
	return true
}

// stripAideComment drops a trailing comment. A '#' inside a macro reference or a
// path is not a comment introducer in practice, but AIDE treats any unescaped
// '#' as one, so this follows AIDE.
func stripAideComment(line string) string {
	for i := 0; i < len(line); i++ {
		if line[i] != '#' {
			continue
		}
		if i > 0 && line[i-1] == '\\' {
			continue
		}
		return line[:i]
	}
	return line
}

// expandAideMacros substitutes @@{NAME} references with the macro's value. An
// undefined reference is left in place so it stays visible rather than silently
// collapsing a path to something shorter than AIDE would use.
func expandAideMacros(cfg *aideConfig, line string) string {
	if !strings.Contains(line, "@@{") {
		return line
	}

	return aideMacroRefRegex.ReplaceAllStringFunc(line, func(match string) string {
		name := aideMacroRefRegex.FindStringSubmatch(match)[1]
		if value, ok := cfg.Macros[name]; ok {
			return value
		}
		return match
	})
}

// parseAideSelectionLine parses a selection line, reporting false when the line
// is not one. A line whose path is still an unexpanded macro reference names no
// absolute path and so is not a selection line, which matches AIDE rejecting the
// configuration outright in that state.
func parseAideSelectionLine(cfg *aideConfig, line string, filePath string, lineNumber int) (aideSelectionRule, bool) {
	selection := aideSelectionRecursive

	switch {
	case strings.HasPrefix(line, "!"):
		selection = aideSelectionNegative
		line = strings.TrimSpace(line[1:])
	case strings.HasPrefix(line, "="):
		selection = aideSelectionEquals
		line = strings.TrimSpace(line[1:])
	}

	// a selection line always names an absolute path
	if !strings.HasPrefix(line, "/") {
		return aideSelectionRule{}, false
	}

	path := line
	expression := ""
	if idx := strings.IndexAny(line, " \t"); idx >= 0 {
		path = strings.TrimSpace(line[:idx])
		expression = strings.TrimSpace(line[idx+1:])
	}

	return aideSelectionRule{
		Path:       path,
		Selection:  selection,
		Expression: expression,
		Attributes: resolveAideAttributes(cfg, expression),
		LineNumber: lineNumber,
		File:       filePath,
	}, true
}

// resolveAideAttributes expands an attribute expression into the attributes it
// stands for. Group names defined in the configuration are substituted
// recursively; a token with no definition is kept as written, because AIDE
// defines a handful internally and their meaning moves between releases. A '-'
// term removes what it names from the result.
func resolveAideAttributes(cfg *aideConfig, expression string) []string {
	if strings.TrimSpace(expression) == "" {
		return []string{}
	}

	included := map[string]struct{}{}
	excluded := map[string]struct{}{}

	collectAideAttributes(cfg, expression, false, 0, included, excluded)

	for token := range excluded {
		delete(included, token)
	}

	res := make([]string, 0, len(included))
	for token := range included {
		res = append(res, token)
	}
	sort.Strings(res)
	return res
}

func collectAideAttributes(cfg *aideConfig, expression string, negated bool, depth int, included, excluded map[string]struct{}) {
	if depth > aideMaxGroupDepth {
		return
	}

	for _, token := range splitAideExpression(expression) {
		remove := negated != token.remove

		if definition, ok := cfg.Groups[token.name]; ok {
			collectAideAttributes(cfg, definition, remove, depth+1, included, excluded)
			continue
		}

		if remove {
			excluded[token.name] = struct{}{}
			continue
		}
		included[token.name] = struct{}{}
	}
}

type aideExpressionToken struct {
	name   string
	remove bool
}

// splitAideExpression breaks an attribute expression into its terms, carrying
// whether each was introduced by '-' rather than '+'.
func splitAideExpression(expression string) []aideExpressionToken {
	res := []aideExpressionToken{}

	current := strings.Builder{}
	remove := false

	flush := func() {
		name := strings.TrimSpace(current.String())
		current.Reset()
		if name == "" {
			return
		}
		res = append(res, aideExpressionToken{name: name, remove: remove})
	}

	for _, char := range expression {
		switch char {
		case '+':
			flush()
			remove = false
		case '-':
			flush()
			remove = true
		case ' ', '\t':
			// a restriction such as "f" may be separated by whitespace; treat it
			// as its own term rather than joining it to the next one
			flush()
		default:
			current.WriteRune(char)
		}
	}
	flush()

	return res
}

// aideDatabasePath turns a database setting into a filesystem path. AIDE accepts
// a URL, and the only form naming a local file is "file:".
func aideDatabasePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	lower := strings.ToLower(value)
	switch {
	case strings.HasPrefix(lower, "file://"):
		return value[len("file://"):]
	case strings.HasPrefix(lower, "file:"):
		return value[len("file:"):]
	case strings.HasPrefix(value, "/"):
		return value
	}

	// stdout, stderr, fd:, url: and the like name no local file
	return ""
}

// parseAideVersion pulls the release out of "aide --version" output, whose first
// line reads like "Aide 0.17.4".
func parseAideVersion(out string) string {
	for _, rawLine := range strings.Split(out, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		for _, field := range fields {
			if field == "" {
				continue
			}
			if field[0] >= '0' && field[0] <= '9' {
				return field
			}
		}
		// only the first non-empty line carries the version
		return ""
	}
	return ""
}
