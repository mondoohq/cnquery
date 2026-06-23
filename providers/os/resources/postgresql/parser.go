// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package postgresql contains parsers for PostgreSQL on-disk configuration
// files: postgresql.conf, pg_hba.conf, and pg_ident.conf. The parsers operate
// on already-read file content so they don't depend on a particular
// filesystem implementation — that lets them be unit-tested against
// inlined fixtures and re-used over different transports (local, SSH,
// container snapshot, ...).
package postgresql

import (
	"path/filepath"
	"strconv"
	"strings"
)

// Conf is the flattened result of parsing postgresql.conf (and its include
// fragments). Last-write-wins semantics match PostgreSQL's own behaviour.
type Conf struct {
	// Params is the effective key->value map across the main file and all
	// included fragments. Keys are normalised to lowercase to match
	// PostgreSQL's case-insensitive parameter names.
	Params map[string]string
	// Files lists every file that contributed (main + includes, in load
	// order, deduplicated).
	Files []string
}

// FileReader returns the textual content of `path`. Returning an error other
// than not-found will short-circuit the parser; not-found errors are silently
// ignored for `include_if_exists` directives. The parser does not interpret
// the error — it bubbles errors up via the returned slice on the parser.
type FileReader func(path string) (string, error)

// GlobExpander expands a single shell-style glob pattern into a list of file
// paths. PostgreSQL itself does not glob include directives — but providing
// this hook lets the caller resolve `include_dir 'conf.d'` by reading the
// directory listing. The default behaviour (when this is nil) treats
// include_dir's argument as a literal path with no globbing.
type GlobExpander func(pattern string) ([]string, error)

// ParseConf parses postgresql.conf at `path`, following any include /
// include_if_exists / include_dir directives encountered. `fileReader` is
// used for both the root file and includes. `dirLister` (optional) expands
// `include_dir` arguments into a sorted list of files; pass nil when the
// caller doesn't have a way to enumerate a directory.
func ParseConf(path string, fileReader FileReader, dirLister func(dir string) ([]string, error)) (*Conf, error) {
	c := &Conf{Params: map[string]string{}}
	visited := map[string]bool{}
	err := parseConfRec(c, path, fileReader, dirLister, visited)
	return c, err
}

func parseConfRec(c *Conf, path string, fileReader FileReader, dirLister func(dir string) ([]string, error), visited map[string]bool) error {
	// Canonicalise the path before checking the cycle guard so equivalent
	// spellings (`./foo.conf` vs `conf.d/../foo.conf` vs `foo.conf`) collapse
	// to the same key. Without this the recursive include detection would
	// miss self-referential loops that walk through `..` segments.
	key := filepath.Clean(path)
	if visited[key] {
		return nil
	}
	visited[key] = true

	content, err := fileReader(path)
	if err != nil {
		return err
	}
	c.Files = append(c.Files, path)

	baseDir := filepath.Dir(path)

	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || line[0] == '#' {
			continue
		}
		// Trim inline comments. PostgreSQL does NOT support inline comments
		// inside quoted strings, so we walk respecting quoting.
		line = trimInlineComment(line)
		if line == "" {
			continue
		}

		key, value, ok := splitConfKV(line)
		if !ok {
			continue
		}
		key = strings.ToLower(key)

		switch key {
		case "include":
			next := resolveInclude(baseDir, value)
			if err := parseConfRec(c, next, fileReader, dirLister, visited); err != nil {
				return err
			}
		case "include_if_exists":
			next := resolveInclude(baseDir, value)
			if err := parseConfRec(c, next, fileReader, dirLister, visited); err != nil {
				// Best-effort: only swallow not-found-style errors. We treat
				// any error as "file does not exist" since the parser has no
				// portable way to discriminate not-found from other I/O
				// errors across filesystem implementations.
				_ = err
			}
		case "include_dir":
			dir := resolveInclude(baseDir, value)
			if dirLister == nil {
				continue
			}
			entries, err := dirLister(dir)
			if err != nil {
				continue
			}
			// PostgreSQL loads files in C-locale sort order, *.conf only.
			for _, entry := range entries {
				if !strings.HasSuffix(entry, ".conf") {
					continue
				}
				if err := parseConfRec(c, entry, fileReader, dirLister, visited); err != nil {
					return err
				}
			}
		default:
			c.Params[key] = value
		}
	}
	return nil
}

// splitConfKV parses a single non-comment, non-blank postgresql.conf line
// into a key and a value. PostgreSQL accepts either `key = value` or
// `key value` (the `=` is optional). The value may be a bare token, a
// number with a unit suffix, or a single-quoted string that uses two
// consecutive single quotes to represent an embedded single quote.
func splitConfKV(line string) (string, string, bool) {
	// Find the end of the key — first whitespace or '='
	i := 0
	for i < len(line) && line[i] != ' ' && line[i] != '\t' && line[i] != '=' {
		i++
	}
	if i == 0 {
		return "", "", false
	}
	key := line[:i]
	rest := strings.TrimSpace(line[i:])
	if strings.HasPrefix(rest, "=") {
		rest = strings.TrimSpace(rest[1:])
	}
	value := unquoteConfValue(rest)
	return key, value, true
}

// unquoteConfValue strips surrounding single quotes from a postgresql.conf
// value and resolves the doubled single-quote escape (two consecutive
// single quotes) inside the string.
// Returned unchanged when the value isn't quoted.
func unquoteConfValue(s string) string {
	if len(s) < 2 || s[0] != '\'' {
		return s
	}
	// Find the matching closing quote, treating "''" as an escaped single quote.
	var b strings.Builder
	b.Grow(len(s))
	for i := 1; i < len(s); i++ {
		c := s[i]
		if c == '\'' {
			if i+1 < len(s) && s[i+1] == '\'' {
				b.WriteByte('\'')
				i++
				continue
			}
			return b.String()
		}
		b.WriteByte(c)
	}
	// Unterminated quote — return what we have.
	return b.String()
}

// trimInlineComment removes a trailing `# comment` while respecting single-
// quoted strings (where a `#` is literal). Returns the trimmed line.
func trimInlineComment(line string) string {
	inQuote := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch c {
		case '\'':
			if i+1 < len(line) && line[i+1] == '\'' {
				i++
				continue
			}
			inQuote = !inQuote
		case '#':
			if !inQuote {
				return strings.TrimSpace(line[:i])
			}
		}
	}
	return strings.TrimSpace(line)
}

func resolveInclude(baseDir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}

// SplitListParam splits a postgresql.conf parameter value that holds a
// comma- or whitespace-separated list (e.g. `listen_addresses`,
// `shared_preload_libraries`). Individual elements are trimmed of
// whitespace and surrounding double quotes.
func SplitListParam(value string) []string {
	if value == "" {
		return nil
	}
	// Replace commas with spaces and split on whitespace — this matches
	// PostgreSQL's `SplitGUCList()` for the most common cases without
	// pulling in its full string-list grammar.
	value = strings.ReplaceAll(value, ",", " ")
	fields := strings.Fields(value)
	for i, f := range fields {
		if len(f) >= 2 && f[0] == '"' && f[len(f)-1] == '"' {
			fields[i] = f[1 : len(f)-1]
		}
	}
	return fields
}

// IsTruthy returns whether a postgresql.conf value is one of the truthy
// tokens PostgreSQL recognises (on, true, yes, 1) — case-insensitive.
func IsTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "true", "yes", "1":
		return true
	}
	return false
}

// HbaRule is one parsed entry from pg_hba.conf. Fields preserve the file's
// token order so the consumer can faithfully audit the original line.
type HbaRule struct {
	LineNumber int
	Type       string
	Database   string
	User       string
	Address    string
	AuthMethod string
	Options    map[string]string
}

// record is a single logical line from pg_hba.conf or pg_ident.conf after
// inline comments have been stripped and backslash continuations joined. num
// is the 1-based line number where the record starts in the source file.
type record struct {
	num  int
	text string
}

// preprocessRecords turns raw pg_hba.conf / pg_ident.conf content into logical
// records. It strips inline comments (unquoted `#` to end of line) and joins
// physical lines that a trailing unquoted backslash marks as continued, both
// features PostgreSQL's own tokenizer supports. Blank and comment-only lines
// collapse to an empty record so callers can skip them. Double-quoted spans
// are respected: a `#` or `\` inside quotes is treated literally.
func preprocessRecords(content string) []record {
	var out []record
	var buf strings.Builder
	start := 0
	continuing := false

	for i, raw := range strings.Split(content, "\n") {
		body, cont := stripCommentAndContinuation(strings.TrimRight(raw, "\r"))
		if !continuing {
			start = i + 1
			buf.Reset()
		}
		buf.WriteString(body)
		if cont {
			// Insert a separator so tokens on either side of the join don't
			// merge into one when the backslash directly abutted a token.
			buf.WriteByte(' ')
			continuing = true
			continue
		}
		out = append(out, record{num: start, text: strings.TrimSpace(buf.String())})
		continuing = false
	}
	// A trailing backslash on the final line has nothing to continue onto;
	// flush whatever was accumulated rather than dropping it.
	if continuing {
		out = append(out, record{num: start, text: strings.TrimSpace(buf.String())})
	}
	return out
}

// stripCommentAndContinuation removes an unquoted trailing `# comment` from a
// physical line and reports whether the line ends with an unquoted backslash
// continuation marker. When it does, the returned body has that backslash
// removed. A `#` inside a double-quoted span is literal (not a comment), and a
// backslash inside an unterminated quote is not treated as a continuation.
func stripCommentAndContinuation(line string) (string, bool) {
	inQuote := false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"':
			inQuote = !inQuote
		case '#':
			if !inQuote {
				// The comment consumes the rest of the line, including any
				// trailing backslash, so this record cannot continue.
				return strings.TrimRight(line[:i], " \t"), false
			}
		}
	}
	trimmed := strings.TrimRight(line, " \t")
	if !inQuote && strings.HasSuffix(trimmed, `\`) {
		return trimmed[:len(trimmed)-1], true
	}
	return line, false
}

// ParseHba parses the textual content of a pg_hba.conf file into a list of
// rules in file order. Inline comments (an unquoted `#` to the end of the
// line) are stripped, physical lines joined by a trailing unquoted backslash
// are merged into a single record, and double-quoted tokens are honored so a
// `#`, backslash, or whitespace inside quotes stays literal. Lines that don't
// form a well-formed rule (too few fields, or a leading token that isn't a
// known connection type, such as an `include` directive) are silently skipped.
// Each rule's LineNumber is the 1-based line where its record begins.
func ParseHba(content string) []HbaRule {
	var rules []HbaRule
	for _, rec := range preprocessRecords(content) {
		if rec.text == "" {
			continue
		}
		tokens := tokenizeHba(rec.text)
		if len(tokens) < 4 {
			continue
		}
		rule := HbaRule{LineNumber: rec.num, Type: tokens[0]}
		switch tokens[0] {
		case "local":
			// type database user auth-method [options...]
			if len(tokens) < 4 {
				continue
			}
			rule.Database = tokens[1]
			rule.User = tokens[2]
			rule.AuthMethod = tokens[3]
			rule.Options = parseHbaOptions(tokens[4:])
		case "host", "hostssl", "hostnossl", "hostgssenc", "hostnogssenc":
			// type database user address auth-method [options...]
			// Address may be `IP/CIDR`, `hostname`, `samehost`, `samenet`,
			// `all`, or an `IP NETMASK` pair (two tokens).
			if len(tokens) < 5 {
				continue
			}
			rule.Database = tokens[1]
			rule.User = tokens[2]
			rule.Address = tokens[3]
			authIdx := 4
			// Detect the netmask form: `IP NETMASK` (two tokens). The next
			// token is a netmask when it looks like an IP/CIDR-ish value and
			// the token after it would be the auth method.
			if len(tokens) >= 6 && looksLikeNetmask(tokens[4]) {
				rule.Address = tokens[3] + " " + tokens[4]
				authIdx = 5
			}
			rule.AuthMethod = tokens[authIdx]
			rule.Options = parseHbaOptions(tokens[authIdx+1:])
		default:
			continue
		}
		rules = append(rules, rule)
	}
	return rules
}

// tokenizeHba splits a pg_hba.conf line into whitespace-separated tokens,
// honoring double-quoted tokens (`"my user"`).
func tokenizeHba(line string) []string {
	var tokens []string
	var cur strings.Builder
	quoted := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case c == '"':
			quoted = !quoted
		case (c == ' ' || c == '\t') && !quoted:
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

func parseHbaOptions(tokens []string) map[string]string {
	if len(tokens) == 0 {
		return nil
	}
	out := make(map[string]string, len(tokens))
	for _, t := range tokens {
		if eq := strings.IndexByte(t, '='); eq >= 0 {
			out[t[:eq]] = strings.Trim(t[eq+1:], `"`)
		} else {
			out[t] = ""
		}
	}
	return out
}

// looksLikeNetmask returns true when the token looks like a netmask in
// dotted-quad form (255.255.255.0) or an IPv6 mask (ffff:ffff::). It is
// deliberately permissive — the parser only needs to distinguish a netmask
// from an auth method like `md5`.
func looksLikeNetmask(token string) bool {
	if token == "" {
		return false
	}
	// IPv4 dotted-quad
	if strings.Count(token, ".") == 3 {
		for _, part := range strings.Split(token, ".") {
			if _, err := strconv.Atoi(part); err != nil {
				return false
			}
		}
		return true
	}
	// IPv6 mask (contains ":" and hex digits/colons only)
	if strings.ContainsRune(token, ':') {
		for _, r := range token {
			if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') || r == ':' {
				continue
			}
			return false
		}
		return true
	}
	return false
}

// IdentMapping is one parsed entry from pg_ident.conf.
type IdentMapping struct {
	LineNumber     int
	MapName        string
	SystemUsername string
	PgUsername     string
}

// ParseIdent parses the textual content of a pg_ident.conf file into a list
// of mappings in file order. It shares pg_hba.conf's preprocessing: inline
// comments are stripped, backslash-continued physical lines are joined, and
// double-quoted tokens (used for regular-expression system-username patterns)
// are honored. Lines that don't have at least three tokens are silently
// skipped. Each mapping's LineNumber is the 1-based line where its record
// begins.
func ParseIdent(content string) []IdentMapping {
	var out []IdentMapping
	for _, rec := range preprocessRecords(content) {
		if rec.text == "" {
			continue
		}
		tokens := tokenizeHba(rec.text) // same quoting rules
		if len(tokens) < 3 {
			continue
		}
		out = append(out, IdentMapping{
			LineNumber:     rec.num,
			MapName:        tokens[0],
			SystemUsername: tokens[1],
			PgUsername:     tokens[2],
		})
	}
	return out
}
