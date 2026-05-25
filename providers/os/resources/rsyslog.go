// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"

	"go.mondoo.com/mql/v13/checksums"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

// rsyslogConfPaths maps platform names to their rsyslog.conf location.
// BSD variants install rsyslog via package managers to non-default prefixes.
var rsyslogConfPaths = map[string]string{
	"freebsd":      "/usr/local/etc/rsyslog.conf",
	"dragonflybsd": "/usr/local/etc/rsyslog.conf",
	"openbsd":      "/usr/local/etc/rsyslog.conf",
	"netbsd":       "/usr/pkg/etc/rsyslog.conf",
}

func rsyslogConfPath(conn shared.Connection) string {
	asset := conn.Asset()
	if asset != nil && asset.Platform != nil {
		if p, ok := rsyslogConfPaths[asset.Platform.Name]; ok {
			return p
		}
	}
	return "/etc/rsyslog.conf"
}

func (s *mqlRsyslogConf) id() (string, error) {
	files := s.GetFiles()
	if files.Error != nil {
		return "", files.Error
	}

	checksum := checksums.New
	for i := range files.Data {
		path := files.Data[i].(*mqlFile).Path.Data
		checksum = checksum.Add(path)
	}

	return checksum.String(), nil
}

func (s *mqlRsyslogConf) path() (string, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)
	return rsyslogConfPath(conn), nil
}

// rsyslogIncludeDirective matches the legacy `$IncludeConfig path` form,
// case-insensitively. The path captured is everything after the directive
// up to end-of-line.
var rsyslogIncludeDirective = regexp.MustCompile(`(?i)^\s*\$IncludeConfig\s+(\S.*?)\s*$`)

// rsyslogModernInclude matches the modern RainerScript `include(...)` block.
// Inner args are parsed separately to handle key=value pairs in any order.
// Multi-line blocks are pre-coalesced into a single line before matching,
// so this anchored form sees the joined text.
var rsyslogModernInclude = regexp.MustCompile(`^\s*include\s*\((.*)\)\s*$`)

// rsyslogModernIncludeOpen recognizes a line that begins an `include(...)`
// directive, used to detect the start of a possibly multi-line block.
var rsyslogModernIncludeOpen = regexp.MustCompile(`^\s*include\s*\(`)

// rsyslogIncludeFileKV pulls the file="..." (or unquoted) value out of a
// modern include() argument list.
var rsyslogIncludeFileKV = regexp.MustCompile(`\bfile\s*=\s*(?:"([^"]*)"|'([^']*)'|(\S+))`)

// parseRsyslogIncludes extracts file-glob patterns referenced by
// $IncludeConfig (legacy) and include(file="...") (modern RainerScript)
// directives from a single rsyslog config blob. Returned patterns are in
// source order with duplicates removed. The include(text="...") form is
// skipped because it inlines content rather than referencing a file.
//
// rsyslog itself ignores `#` comments anywhere on a line outside of quoted
// strings. We strip everything from the first unquoted `#` to EOL before
// matching, so a comment hanging off an include directive does not pollute
// the pattern. Multi-line include() blocks (common in Ansible-templated
// configs) are coalesced into a single logical line before matching.
func parseRsyslogIncludes(content string) []string {
	var out []string
	seen := map[string]bool{}

	for _, line := range coalesceIncludeBlocks(content) {
		var pat string
		if m := rsyslogIncludeDirective.FindStringSubmatch(line); m != nil {
			pat = strings.Trim(strings.TrimSpace(m[1]), `"'`)
		} else if m := rsyslogModernInclude.FindStringSubmatch(line); m != nil {
			if kv := rsyslogIncludeFileKV.FindStringSubmatch(m[1]); kv != nil {
				// Exactly one of the three capture groups (quoted-double,
				// quoted-single, unquoted) is non-empty.
				for _, v := range kv[1:] {
					if v != "" {
						pat = v
						break
					}
				}
			}
		}

		if pat == "" || seen[pat] {
			continue
		}
		seen[pat] = true
		out = append(out, pat)
	}

	return out
}

// coalesceIncludeBlocks strips comments and trims each line, then joins
// continuation lines inside an unterminated `include(...)` block into a
// single logical line. Tools like Ansible templates routinely emit:
//
//	include(
//	    file="/etc/rsyslog.d/*.conf"
//	)
//
// which rsyslog accepts. Lines outside an open `include(` are returned
// one per source line so the line-anchored `$IncludeConfig` regex still
// matches correctly. Blank lines outside a block are dropped.
func coalesceIncludeBlocks(content string) []string {
	rawLines := strings.Split(content, "\n")
	var out []string
	var pending strings.Builder
	openParens := 0

	for _, raw := range rawLines {
		line := stripRsyslogComment(raw)
		line = strings.TrimSpace(line)
		if line == "" && openParens == 0 {
			continue
		}

		if openParens == 0 && rsyslogModernIncludeOpen.MatchString(line) {
			openParens = countUnquotedParens(line)
			if openParens == 0 {
				// Single-line include(...) — emit as-is.
				out = append(out, line)
				continue
			}
			pending.WriteString(line)
			continue
		}
		if openParens > 0 {
			if pending.Len() > 0 {
				pending.WriteByte(' ')
			}
			pending.WriteString(line)
			openParens += countUnquotedParens(line)
			if openParens <= 0 {
				out = append(out, pending.String())
				pending.Reset()
				openParens = 0
			}
			continue
		}
		out = append(out, line)
	}

	// Unterminated block — emit what we have so the regex can still try
	// to extract a pattern. rsyslog itself would reject this config at
	// load time, so we surface whatever was given rather than silently
	// dropping the directive.
	if pending.Len() > 0 {
		out = append(out, pending.String())
	}
	return out
}

// countUnquotedParens returns (# of `(`) - (# of `)`) on a line,
// ignoring parens inside double- or single-quoted strings.
func countUnquotedParens(line string) int {
	count := 0
	inDouble, inSingle := false, false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '(':
			if !inDouble && !inSingle {
				count++
			}
		case ')':
			if !inDouble && !inSingle {
				count--
			}
		}
	}
	return count
}

// stripRsyslogComment removes everything from the first unquoted `#` to
// end-of-line, mirroring rsyslog's own lexer behaviour. Quotes can be
// double or single.
func stripRsyslogComment(line string) string {
	inDouble, inSingle := false, false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '#':
			if !inDouble && !inSingle {
				return line[:i]
			}
		}
	}
	return line
}

// resolveRsyslogInclude turns a (possibly relative, possibly globbed) include
// pattern into a directory + basename-regex pair suitable for passing to
// files.find. The returned regex is anchored on both ends and matches the
// basename only. A pattern with no glob meta-characters resolves to a regex
// that matches that single name exactly.
func resolveRsyslogInclude(parentDir, pattern string) (dir, nameRegex string) {
	if !filepath.IsAbs(pattern) {
		pattern = filepath.Join(parentDir, pattern)
	}
	dir = filepath.Dir(pattern)
	base := filepath.Base(pattern)
	return dir, "^" + globToRegex(base) + "$"
}

// globToRegex converts a shell glob (the only metas rsyslog needs in
// practice are `*`, `?`, and character classes) into a regular expression
// suitable for files.find's `name` parameter, which is compiled as regexp.
// Glob `*` becomes `[^/]*`, `?` becomes `[^/]`, `[abc]` is passed through.
// Everything else is regex-escaped.
func globToRegex(glob string) string {
	var b strings.Builder
	b.Grow(len(glob) + 4)
	inClass := false
	for i := 0; i < len(glob); i++ {
		c := glob[i]
		switch {
		case inClass:
			b.WriteByte(c)
			if c == ']' {
				inClass = false
			}
		case c == '[':
			b.WriteByte(c)
			inClass = true
		case c == '*':
			b.WriteString(`[^/]*`)
		case c == '?':
			b.WriteString(`[^/]`)
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	return b.String()
}

// maxRsyslogIncludeDepth bounds how many levels of nested includes we
// follow. rsyslog itself has no documented limit, but real configurations
// rarely nest beyond two or three levels — a deeper chain is almost
// certainly a misconfiguration or a self-reference we missed.
const maxRsyslogIncludeDepth = 32

func (s *mqlRsyslogConf) files(path string) ([]any, error) {
	if !strings.HasSuffix(path, ".conf") {
		return nil, errors.New("failed to initialize, path must end in `.conf` so we can find files in `.d` directory")
	}

	visited := map[string]bool{}
	var out []any

	type queued struct {
		path  string
		depth int
	}
	queue := []queued{{path: path, depth: 0}}

	for len(queue) > 0 {
		head := queue[0]
		queue = queue[1:]

		clean := filepath.Clean(head.path)
		if visited[clean] {
			continue
		}
		visited[clean] = true

		f, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
			"path": llx.StringData(clean),
		})
		if err != nil {
			return nil, err
		}
		mf := f.(*mqlFile)
		out = append(out, mf)

		if head.depth >= maxRsyslogIncludeDepth {
			continue
		}

		content := mf.GetContent()
		if content.Error != nil {
			if errors.Is(content.Error, resources.NotFoundError{}) {
				continue
			}
			// Other read errors (permission denied, IO) are non-fatal here:
			// the file is still listed via the resource, and the caller can
			// inspect it for the error. Don't abort the whole walk.
			continue
		}

		patterns := parseRsyslogIncludes(content.Data)
		parentDir := filepath.Dir(clean)
		for _, pat := range patterns {
			matches, err := s.expandIncludePattern(parentDir, pat)
			if err != nil {
				continue
			}
			for _, m := range matches {
				if !visited[filepath.Clean(m)] {
					queue = append(queue, queued{path: m, depth: head.depth + 1})
				}
			}
		}
	}

	// Legacy `.d` auto-discovery: configurations that rely on the
	// distribution's default to drop fragments into `<conf>.d/` without
	// an explicit `$IncludeConfig` should still surface those files.
	// Skip entries already visited via include traversal so the list
	// doesn't double-count when both paths reach the same fragment.
	//
	// No depth bound here: distro packages drop fragments directly into
	// `<conf>.d/` (no subdirs in practice), and this matches the original
	// behaviour of the resource — narrowing it now would silently change
	// the file list for callers relying on it.
	confD := path[0:len(path)-5] + ".d"
	o, err := CreateResource(s.MqlRuntime, "files.find", map[string]*llx.RawData{
		"from": llx.StringData(confD),
		"type": llx.StringData("file"),
	})
	if err == nil {
		list := o.(*mqlFilesFind).GetList()
		if list.Error == nil {
			for _, item := range list.Data {
				mf, ok := item.(*mqlFile)
				if !ok {
					continue
				}
				if visited[filepath.Clean(mf.Path.Data)] {
					continue
				}
				visited[filepath.Clean(mf.Path.Data)] = true
				out = append(out, mf)
			}
		}
	}

	return out, nil
}

// expandIncludePattern resolves a single include pattern (relative to
// parentDir if not absolute) into the list of files it matches, using
// files.find against the asset's filesystem.
//
// depth=1 restricts the search to the immediate directory, matching
// rsyslog's own glob(3) semantics: `$IncludeConfig /etc/rsyslog.d/*.conf`
// does not pick up `/etc/rsyslog.d/nested/extra.conf`.
func (s *mqlRsyslogConf) expandIncludePattern(parentDir, pattern string) ([]string, error) {
	dir, nameRegex := resolveRsyslogInclude(parentDir, pattern)

	o, err := CreateResource(s.MqlRuntime, "files.find", map[string]*llx.RawData{
		"from":  llx.StringData(dir),
		"type":  llx.StringData("file"),
		"name":  llx.StringData(nameRegex),
		"depth": llx.IntData(1),
	})
	if err != nil {
		return nil, err
	}

	list := o.(*mqlFilesFind).GetList()
	if list.Error != nil {
		return nil, list.Error
	}

	paths := make([]string, 0, len(list.Data))
	for _, item := range list.Data {
		if mf, ok := item.(*mqlFile); ok {
			paths = append(paths, mf.Path.Data)
		}
	}
	return paths, nil
}

func (s *mqlRsyslogConf) content(files []any) (string, error) {
	var res strings.Builder

	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes
	for i := range files {
		file := files[i].(*mqlFile)
		content := file.GetContent()
		if content.Error != nil {
			if errors.Is(content.Error, resources.NotFoundError{}) {
				continue
			}
		}

		res.WriteString(content.Data)
		res.WriteString("\n")
	}

	return res.String(), nil
}

func (s *mqlRsyslogConf) settings(content string) ([]any, error) {
	lines := strings.Split(content, "\n")

	settings := []any{}
	var line string
	for i := range lines {
		line = lines[i]
		line = stripRsyslogComment(line)
		line = strings.Trim(line, " \t\r")

		if line != "" {
			settings = append(settings, line)
		}
	}

	return settings, nil
}
