// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package redisconf contains the parser for the Redis configuration file,
// which Valkey uses unchanged. It is named for the file format rather than
// for either product, the way mycnf serves both MySQL and MariaDB.
//
// The format is neither ini nor YAML: every line is a directive name followed
// by whitespace-separated arguments, quoted the way redis-server's
// sdssplitargs quotes them. Only a whole line can be a comment, so a `#` in
// the middle of a line is part of an argument rather than the start of a
// comment.
//
// Two behaviors make a naive last-write-wins map wrong. `include` splices
// another file in at the point it appears, so ordering across files decides
// which value survives, and a handful of directives accumulate rather than
// replace: every `save` line adds a snapshot rule, and `rename-command`,
// `user`, and `client-output-buffer-limit` each add an entry.
package redisconf

import (
	"strconv"
	"strings"
)

// Directive is one configuration line, with the file and line it came from so
// a reader can attribute a value to its source.
type Directive struct {
	Name string
	Args []string
	File string
	Line int
}

// Conf is a fully resolved configuration: every include spliced in, and every
// directive in the order the server would apply it.
type Conf struct {
	// Files lists the main file and every file pulled in by an include, in
	// load order.
	Files []string
	// Directives holds every directive in effective order. Later entries win
	// for the settings that replace, and add for the settings that
	// accumulate.
	Directives []Directive
}

// Loader supplies file content and glob expansion to Load, so the parser
// stays independent of any particular filesystem.
type Loader interface {
	Read(path string) (string, error)
	Glob(pattern string) ([]string, error)
}

// maxIncludeDepth bounds include recursion. Redis itself does not detect a
// cycle here, so a self-including file would otherwise loop forever.
const maxIncludeDepth = 16

// Load reads a configuration file and splices in every include it reaches.
//
// A missing or unreadable include is skipped rather than failed: redis-server
// refuses to start on one, so a file that references a missing fragment is
// already not the running configuration, and reporting the directives that do
// resolve is more useful than reporting nothing.
func Load(path string, loader Loader) (*Conf, error) {
	conf := &Conf{}
	seen := map[string]bool{}
	if err := load(conf, path, loader, seen, 0); err != nil {
		return nil, err
	}
	return conf, nil
}

func load(conf *Conf, path string, loader Loader, seen map[string]bool, depth int) error {
	if depth > maxIncludeDepth || seen[path] {
		return nil
	}
	seen[path] = true

	content, err := loader.Read(path)
	if err != nil {
		// Only the top-level file is required; see Load.
		if depth == 0 {
			return err
		}
		return nil
	}
	conf.Files = append(conf.Files, path)

	for _, d := range ParseDirectives(content) {
		d.File = path
		if !strings.EqualFold(d.Name, "include") {
			conf.Directives = append(conf.Directives, d)
			continue
		}
		for _, arg := range d.Args {
			for _, target := range expand(arg, loader) {
				if err := load(conf, target, loader, seen, depth+1); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// expand resolves an include argument, which may be a glob.
func expand(arg string, loader Loader) []string {
	if !strings.ContainsAny(arg, "*?[") {
		return []string{arg}
	}
	matches, err := loader.Glob(arg)
	if err != nil {
		return nil
	}
	return matches
}

// ParseDirectives splits file content into directives, keeping include lines
// in place so the caller can splice them.
func ParseDirectives(content string) []Directive {
	out := []Directive{}
	for i, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		// A `#` only opens a comment at the start of a line. Mid-line it is
		// an ordinary character, which matters for values like a password or
		// a key pattern that contains one.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		args := SplitArgs(line)
		if len(args) == 0 {
			continue
		}
		out = append(out, Directive{Name: args[0], Args: args[1:], Line: i + 1})
	}
	return out
}

// SplitArgs splits a configuration line the way redis-server's sdssplitargs
// does: whitespace separates arguments, double quotes support escapes, and
// single quotes are literal apart from an escaped quote.
//
// An unbalanced quote makes redis-server reject the whole line, so this
// reports no arguments for one rather than guessing at the intent.
func SplitArgs(line string) []string {
	var args []string
	i := 0
	for {
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
		if i >= len(line) {
			return args
		}

		var sb strings.Builder
		closed := false
		switch line[i] {
		case '"':
			i++
			for i < len(line) {
				if line[i] == '\\' && i+3 < len(line) && line[i+1] == 'x' &&
					isHex(line[i+2]) && isHex(line[i+3]) {
					v, _ := strconv.ParseUint(line[i+2:i+4], 16, 8)
					sb.WriteByte(byte(v))
					i += 4
					continue
				}
				if line[i] == '\\' && i+1 < len(line) {
					sb.WriteByte(unescape(line[i+1]))
					i += 2
					continue
				}
				if line[i] == '"' {
					i++
					closed = true
					break
				}
				sb.WriteByte(line[i])
				i++
			}
		case '\'':
			i++
			for i < len(line) {
				if line[i] == '\\' && i+1 < len(line) && line[i+1] == '\'' {
					sb.WriteByte('\'')
					i += 2
					continue
				}
				if line[i] == '\'' {
					i++
					closed = true
					break
				}
				sb.WriteByte(line[i])
				i++
			}
		default:
			for i < len(line) && line[i] != ' ' && line[i] != '\t' {
				sb.WriteByte(line[i])
				i++
			}
			closed = true
		}

		if !closed {
			// Unbalanced quote: redis-server rejects the line outright.
			return nil
		}
		// A closing quote has to be followed by a separator, same as the
		// server requires.
		if i < len(line) && line[i] != ' ' && line[i] != '\t' {
			return nil
		}
		args = append(args, sb.String())
	}
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func unescape(c byte) byte {
	switch c {
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	case 'b':
		return '\b'
	case 'a':
		return '\a'
	default:
		return c
	}
}

// ---------------------------------------------------------------------------
// directive lookups
// ---------------------------------------------------------------------------

// All reports every occurrence of a directive, in effective order. Use it for
// the settings that accumulate.
func (c *Conf) All(name string) [][]string {
	var out [][]string
	for _, d := range c.Directives {
		if strings.EqualFold(d.Name, name) {
			out = append(out, d.Args)
		}
	}
	return out
}

// Last reports the arguments of the final occurrence of a directive, and nil
// when it never appears. Use it for the settings that replace.
func (c *Conf) Last(name string) []string {
	var out []string
	found := false
	for _, d := range c.Directives {
		if strings.EqualFold(d.Name, name) {
			out = d.Args
			found = true
		}
	}
	if !found {
		return nil
	}
	if out == nil {
		// The directive appeared with no arguments, which is distinct from
		// never appearing.
		return []string{}
	}
	return out
}

// Has reports whether a directive appears at all, which is what separates an
// unset value from one explicitly set to the server default.
func (c *Conf) Has(name string) bool {
	return c.Last(name) != nil
}

// String reports the first argument of the last occurrence, or def.
func (c *Conf) String(name, def string) string {
	args := c.Last(name)
	if len(args) == 0 {
		return def
	}
	return args[0]
}

// Bool reports a yes/no directive, or def.
func (c *Conf) Bool(name string, def bool) bool {
	switch strings.ToLower(c.String(name, "")) {
	case "yes":
		return true
	case "no":
		return false
	default:
		return def
	}
}

// Int reports a plain integer directive, or def.
func (c *Conf) Int(name string, def int64) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(c.String(name, "")), 10, 64)
	if err != nil {
		return def
	}
	return v
}

// Bytes reports a memory-size directive, or def.
//
// Redis distinguishes the two spellings deliberately: 1k is 1000 bytes and
// 1kb is 1024, so they cannot share a multiplier.
func (c *Conf) Bytes(name string, def int64) int64 {
	raw := strings.ToLower(strings.TrimSpace(c.String(name, "")))
	if raw == "" {
		return def
	}

	for _, unit := range []struct {
		suffix string
		mult   int64
	}{
		{"kb", 1024},
		{"mb", 1024 * 1024},
		{"gb", 1024 * 1024 * 1024},
		{"k", 1000},
		{"m", 1000 * 1000},
		{"g", 1000 * 1000 * 1000},
		{"b", 1},
	} {
		if strings.HasSuffix(raw, unit.suffix) {
			n, err := strconv.ParseInt(strings.TrimSpace(strings.TrimSuffix(raw, unit.suffix)), 10, 64)
			if err != nil {
				return def
			}
			return n * unit.mult
		}
	}

	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return def
	}
	return n
}
