// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package apache2

import (
	"strings"

	"github.com/rs/zerolog/log"
)

// Module represents a LoadModule directive.
type Module struct {
	Name string // e.g., "ssl_module"
	Path string // e.g., "modules/mod_ssl.so"
}

// VirtualHost represents a <VirtualHost> block.
type VirtualHost struct {
	Address      string         // e.g., "*:443"
	ServerName   string         // ServerName directive
	DocumentRoot string         // DocumentRoot directive
	SSL          bool           // SSLEngine on
	Params       map[string]any // all directives in this block
}

// Directory represents a <Directory> block.
type Directory struct {
	Path          string         // e.g., "/var/www/html"
	Options       string         // Options directive
	AllowOverride string         // AllowOverride directive
	Params        map[string]any // all directives in this block
}

// Config is the parsed result of Apache configuration files.
type Config struct {
	Params   map[string]any // top-level directives (key → value)
	Modules  []Module       // LoadModule directives
	VHosts   []VirtualHost  // <VirtualHost> blocks
	Dirs     []Directory    // <Directory> blocks
	Includes []string       // Include/IncludeOptional paths (unexpanded)
}

type (
	fileContentFunc func(string) (string, error)
	globExpandFunc  func(string) ([]string, error)
)

// Parse parses a single Apache config file content.
func Parse(content string) *Config {
	cfg := &Config{
		Params: map[string]any{},
	}

	lines := splitAndClean(content)
	parseLines(cfg, lines, 0)
	return cfg
}

// ParseWithGlob parses Apache config files, recursively expanding Include and
// IncludeOptional directives using the provided glob and file-content functions.
func ParseWithGlob(rootPath string, fileContent fileContentFunc, globExpand globExpandFunc) (*Config, error) {
	content, err := fileContent(rootPath)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Params: map[string]any{},
	}

	parseWithGlobRecursive(cfg, rootPath, content, fileContent, globExpand)
	return cfg, nil
}

func parseWithGlobRecursive(cfg *Config, filePath, content string, fileContent fileContentFunc, globExpand globExpandFunc) {
	lines := splitAndClean(content)
	i := 0
	for i < len(lines) {
		line := lines[i]

		// Block directives: <VirtualHost>, <Directory>, etc.
		if strings.HasPrefix(line, "<") {
			blockTag, blockArg := parseBlockOpen(line)
			blockLines, end := collectBlock(lines, i+1, blockTag)
			i = end + 1

			switch strings.ToLower(blockTag) {
			case "virtualhost":
				vh := parseVirtualHost(blockArg, blockLines)
				cfg.VHosts = append(cfg.VHosts, vh)
			case "directory", "directorymatch":
				d := parseDirectory(blockArg, blockLines)
				cfg.Dirs = append(cfg.Dirs, d)
			}
			// Other block types (Location, Files, etc.) are silently skipped for now
			continue
		}

		key, value := parseDirective(line)
		if key == "" {
			i++
			continue
		}

		keyLower := strings.ToLower(key)

		switch keyLower {
		case "include", "includeoptional":
			cfg.Includes = append(cfg.Includes, value)
			if globExpand != nil && fileContent != nil {
				expandInclude(cfg, filePath, value, fileContent, globExpand, keyLower == "includeoptional")
			}
		case "loadmodule":
			parts := strings.Fields(value)
			if len(parts) >= 2 {
				cfg.Modules = append(cfg.Modules, Module{Name: parts[0], Path: parts[1]})
			}
		default:
			setParam(cfg.Params, key, value)
		}

		i++
	}
}

func expandInclude(cfg *Config, parentPath, pattern string, fileContent fileContentFunc, globExpand globExpandFunc, optional bool) {
	paths, err := globExpand(pattern)
	if err != nil {
		if !optional {
			log.Warn().Err(err).Str("pattern", pattern).Msg("unable to expand Include directive")
		}
		return
	}

	for _, p := range paths {
		content, err := fileContent(p)
		if err != nil {
			if !optional {
				log.Warn().Err(err).Str("path", p).Msg("unable to read included file")
			}
			continue
		}
		parseWithGlobRecursive(cfg, p, content, fileContent, globExpand)
	}
}

// parseLines parses lines at the top level (no glob expansion).
func parseLines(cfg *Config, lines []string, start int) {
	i := start
	for i < len(lines) {
		line := lines[i]

		if strings.HasPrefix(line, "<") {
			blockTag, blockArg := parseBlockOpen(line)
			blockLines, end := collectBlock(lines, i+1, blockTag)
			i = end + 1

			switch strings.ToLower(blockTag) {
			case "virtualhost":
				vh := parseVirtualHost(blockArg, blockLines)
				cfg.VHosts = append(cfg.VHosts, vh)
			case "directory", "directorymatch":
				d := parseDirectory(blockArg, blockLines)
				cfg.Dirs = append(cfg.Dirs, d)
			}
			continue
		}

		key, value := parseDirective(line)
		if key == "" {
			i++
			continue
		}

		keyLower := strings.ToLower(key)
		switch keyLower {
		case "include", "includeoptional":
			cfg.Includes = append(cfg.Includes, value)
		case "loadmodule":
			parts := strings.Fields(value)
			if len(parts) >= 2 {
				cfg.Modules = append(cfg.Modules, Module{Name: parts[0], Path: parts[1]})
			}
		default:
			setParam(cfg.Params, key, value)
		}

		i++
	}
}

// parseVirtualHost parses the lines inside a <VirtualHost> block.
func parseVirtualHost(address string, lines []string) VirtualHost {
	vh := VirtualHost{
		Address: address,
		Params:  map[string]any{},
	}

	for _, line := range lines {
		// Skip nested blocks inside VirtualHost (e.g., <Directory>, <Location>)
		if strings.HasPrefix(line, "<") {
			continue
		}

		key, value := parseDirective(line)
		if key == "" {
			continue
		}

		setParam(vh.Params, key, value)

		switch strings.ToLower(key) {
		case "servername":
			vh.ServerName = value
		case "documentroot":
			vh.DocumentRoot = value
		case "sslengine":
			vh.SSL = strings.EqualFold(value, "on")
		}
	}

	return vh
}

// parseDirectory parses the lines inside a <Directory> block.
func parseDirectory(path string, lines []string) Directory {
	d := Directory{
		Path:   path,
		Params: map[string]any{},
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "<") {
			continue
		}

		key, value := parseDirective(line)
		if key == "" {
			continue
		}

		setParam(d.Params, key, value)

		switch strings.ToLower(key) {
		case "options":
			d.Options = value
		case "allowoverride":
			d.AllowOverride = value
		}
	}

	return d
}

// splitAndClean splits content into lines, strips comments and blank lines,
// and handles continuation lines (trailing backslash).
func splitAndClean(content string) []string {
	raw := strings.Split(content, "\n")
	var lines []string
	var continued string

	for _, line := range raw {
		line = strings.TrimRight(line, "\r")
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || line[0] == '#' {
			continue
		}

		// Strip inline comments (but not inside quotes)
		line = stripInlineComment(line)
		if line == "" {
			continue
		}

		// Handle continuation lines
		if strings.HasSuffix(line, "\\") {
			continued += strings.TrimSuffix(line, "\\") + " "
			continue
		}
		if continued != "" {
			line = continued + line
			continued = ""
		}

		lines = append(lines, line)
	}

	// Flush any trailing continuation
	if continued != "" {
		lines = append(lines, strings.TrimSpace(continued))
	}

	return lines
}

// stripInlineComment removes # comments that aren't inside quotes.
func stripInlineComment(line string) string {
	inQuote := false
	for i, c := range line {
		switch c {
		case '"':
			inQuote = !inQuote
		case '#':
			if !inQuote {
				return strings.TrimSpace(line[:i])
			}
		}
	}
	return line
}

// parseDirective splits "Key value" or "Key" into key and value.
func parseDirective(line string) (string, string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", ""
	}

	// Find the key (first whitespace-delimited token)
	idx := strings.IndexAny(line, " \t")
	if idx < 0 {
		return line, ""
	}

	key := line[:idx]
	value := strings.TrimSpace(line[idx+1:])

	// Remove surrounding quotes from value
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}

	return key, value
}

// parseBlockOpen parses "<Tag arg>" returning tag and arg.
func parseBlockOpen(line string) (string, string) {
	line = strings.TrimSpace(line)
	// Remove < and >
	if len(line) < 2 {
		return "", ""
	}
	line = line[1:] // remove <
	if line[len(line)-1] == '>' {
		line = line[:len(line)-1]
	}

	idx := strings.IndexAny(line, " \t")
	if idx < 0 {
		return line, ""
	}

	arg := strings.TrimSpace(line[idx+1:])
	// Remove surrounding quotes from argument
	if len(arg) >= 2 && arg[0] == '"' && arg[len(arg)-1] == '"' {
		arg = arg[1 : len(arg)-1]
	}

	return line[:idx], arg
}

// collectBlock collects lines until the matching </tag> closing tag.
// Returns the inner lines and the index of the closing tag line.
func collectBlock(lines []string, start int, tag string) ([]string, int) {
	closeTag := "</" + strings.ToLower(tag)
	depth := 1
	var inner []string

	for i := start; i < len(lines); i++ {
		lower := strings.ToLower(strings.TrimSpace(lines[i]))
		if strings.HasPrefix(lower, "<"+strings.ToLower(tag)) {
			depth++
		} else if strings.HasPrefix(lower, closeTag) {
			depth--
			if depth == 0 {
				return inner, i
			}
		}
		inner = append(inner, lines[i])
	}

	// Unclosed block — return what we have
	return inner, len(lines) - 1
}

// setParam sets a directive value. For directives that can appear multiple
// times (Listen, Header, etc.), values are comma-concatenated.
func setParam(m map[string]any, key string, value string) {
	if isMultiParam[strings.ToLower(key)] {
		if v, ok := m[key]; ok {
			m[key] = v.(string) + "," + value
			return
		}
	}
	m[key] = value
}

// isMultiParam lists directives that can appear multiple times and should
// be concatenated rather than overwritten.
var isMultiParam = map[string]bool{
	"listen":          true,
	"header":          true,
	"loadmodule":      true,
	"alias":           true,
	"redirect":        true,
	"rewriterule":     true,
	"rewritecond":     true,
	"setenvif":        true,
	"customlog":       true,
	"logformat":       true,
	"serveralias":     true,
	"allowmethods":    true,
	"require":         true,
	"addtype":         true,
	"addhandler":      true,
	"addoutputfilter": true,
}
