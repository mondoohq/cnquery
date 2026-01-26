// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v12/checksums"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/types"
)

const (
	defaultSudoersFile = "/etc/sudoers"
	defaultSudoersDir  = "/etc/sudoers.d"
)

var (
	// Regular expressions for parsing sudoers entries
	sudoersAliasRegex = regexp.MustCompile(`^(User_Alias|Runas_Alias|Host_Alias|Cmnd_Alias)\s+(\w+)\s*=\s*(.+)$`)
	defaultsRegex     = regexp.MustCompile(`^Defaults\b`)
	// Include directives: @include, @includedir, #include, #includedir (sudo 1.9.1+)
	includeRegex    = regexp.MustCompile(`^[@#]include\s+(.+)$`)
	includedirRegex = regexp.MustCompile(`^[@#]includedir\s+(.+)$`)
)

func initSudoers(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' it must be a string")
		}

		// If a custom path is provided, just use that file
		files, err := getSortedPathFiles(runtime, path)
		if err != nil {
			return nil, nil, err
		}
		args["files"] = llx.ArrayData(files, types.Resource("file"))
		delete(args, "path")
	}

	return args, nil, nil
}

func (s *mqlSudoers) id() (string, error) {
	checksum := checksums.New
	for i := range s.Files.Data {
		path := s.Files.Data[i].(*mqlFile).Path.Data
		checksum = checksum.Add(path)
	}
	return checksum.String(), nil
}

func (se *mqlSudoersUserSpec) id() (string, error) {
	file := se.File.Data
	lineNum := strconv.FormatInt(se.LineNumber.Data, 10)

	// Create unique ID from file path and line number
	id := file + ":" + lineNum + ":user_spec"

	return id, nil
}

func (sd *mqlSudoersDefault) id() (string, error) {
	file := sd.File.Data
	lineNum := strconv.FormatInt(sd.LineNumber.Data, 10)

	// Create unique ID from file path and line number
	id := file + ":" + lineNum + ":default"

	return id, nil
}

func (sa *mqlSudoersAlias) id() (string, error) {
	file := sa.File.Data
	lineNum := strconv.FormatInt(sa.LineNumber.Data, 10)
	aliasType := sa.Type.Data

	// Create unique ID from file path, line number, and alias type
	id := file + ":" + lineNum + ":" + aliasType + "_alias"

	return id, nil
}

// files returns the list of sudoers configuration files
func (s *mqlSudoers) files() ([]any, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)
	visited := make(map[string]bool)
	var allFiles []any

	// Start with the main sudoers file
	err := s.collectSudoersFiles(conn, defaultSudoersFile, visited, &allFiles)
	if err != nil {
		return nil, err
	}

	return allFiles, nil
}

// collectSudoersFiles recursively collects sudoers files, following include directives
func (s *mqlSudoers) collectSudoersFiles(conn shared.Connection, path string, visited map[string]bool, allFiles *[]any) error {
	// Avoid infinite loops
	if visited[path] {
		return nil
	}
	visited[path] = true

	// Check if file exists
	fileRes, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(path),
	})
	if err != nil {
		return err
	}
	f := fileRes.(*mqlFile)
	exists := f.GetExists()
	if exists.Error != nil {
		return exists.Error
	}

	if !exists.Data {
		return nil
	}

	// Add this file to the list
	*allFiles = append(*allFiles, f)

	// Read file content to find include directives
	file, err := conn.FileSystem().Open(path)
	if err != nil {
		return err
	}
	raw, err := io.ReadAll(file)
	file.Close()
	if err != nil {
		return err
	}

	// Parse include directives
	lines := strings.Split(string(raw), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for @include or #include
		if matches := includeRegex.FindStringSubmatch(line); matches != nil {
			includePath := strings.TrimSpace(matches[1])
			if err := s.collectSudoersFiles(conn, includePath, visited, allFiles); err != nil {
				// Continue on error - included file might not exist
				continue
			}
		}

		// Check for @includedir or #includedir
		if matches := includedirRegex.FindStringSubmatch(line); matches != nil {
			includeDir := strings.TrimSpace(matches[1])
			if err := s.collectSudoersDir(conn, includeDir, visited, allFiles); err != nil {
				// Continue on error - directory might not exist
				continue
			}
		}
	}

	return nil
}

// collectSudoersDir collects all sudoers files from a directory
func (s *mqlSudoers) collectSudoersDir(conn shared.Connection, dirPath string, visited map[string]bool, allFiles *[]any) error {
	// Check if directory exists
	dirRes, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(dirPath),
	})
	if err != nil {
		return err
	}
	dir := dirRes.(*mqlFile)
	dirExists := dir.GetExists()
	if dirExists.Error != nil {
		return dirExists.Error
	}

	if !dirExists.Data {
		return nil
	}

	// Get all files from the directory
	files, err := CreateResource(s.MqlRuntime, "files.find", map[string]*llx.RawData{
		"from": llx.StringData(dirPath),
		"type": llx.StringData("file"),
	})
	if err != nil {
		return err
	}

	ff := files.(*mqlFilesFind)
	list := ff.GetList()
	if list.Error != nil {
		return list.Error
	}

	// Process each file in the directory
	for i := range list.Data {
		file := list.Data[i].(*mqlFile)
		basename := file.GetBasename()
		if basename.Error != nil {
			continue
		}

		// Skip README files as per sudoers convention
		if strings.Contains(strings.ToUpper(basename.Data), "README") {
			continue
		}

		// Skip files with . or ~ in the name (sudoers convention)
		if strings.Contains(basename.Data, ".") || strings.Contains(basename.Data, "~") {
			continue
		}

		// Recursively process this file (it may have its own includes)
		filePath := file.Path.Data
		if err := s.collectSudoersFiles(conn, filePath, visited, allFiles); err != nil {
			continue
		}
	}

	return nil
}

// content aggregates the content from all sudoers files
func (s *mqlSudoers) content(files []any) (string, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	var res strings.Builder

	for i := range files {
		file := files[i].(*mqlFile)

		f, err := conn.FileSystem().Open(file.Path.Data)
		if err != nil {
			return "", err
		}

		raw, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return "", err
		}

		res.WriteString(string(raw))
		res.WriteString("\n")
	}

	return res.String(), nil
}

// userSpecs parses all sudoers files and returns user specification entries
func (s *mqlSudoers) userSpecs(files []any) ([]any, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	var allEntries []any

	for i := range files {
		file := files[i].(*mqlFile)

		f, err := conn.FileSystem().Open(file.Path.Data)
		if err != nil {
			return nil, err
		}

		raw, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, err
		}

		entries, err := parseUserSpecs(s.MqlRuntime, file.Path.Data, string(raw))
		if err != nil {
			return nil, err
		}

		allEntries = append(allEntries, entries...)
	}

	return allEntries, nil
}

// defaults parses all sudoers files and returns default entries
func (s *mqlSudoers) defaults(files []any) ([]any, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	var allDefaults []any

	for i := range files {
		file := files[i].(*mqlFile)

		f, err := conn.FileSystem().Open(file.Path.Data)
		if err != nil {
			return nil, err
		}

		raw, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, err
		}

		defaults, err := parseDefaults(s.MqlRuntime, file.Path.Data, string(raw))
		if err != nil {
			return nil, err
		}

		allDefaults = append(allDefaults, defaults...)
	}

	return allDefaults, nil
}

// aliases parses all sudoers files and returns alias definitions
func (s *mqlSudoers) aliases(files []any) ([]any, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	var allAliases []any

	for i := range files {
		file := files[i].(*mqlFile)

		f, err := conn.FileSystem().Open(file.Path.Data)
		if err != nil {
			return nil, err
		}

		raw, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, err
		}

		aliases, err := parseSudoersAliases(s.MqlRuntime, file.Path.Data, string(raw))
		if err != nil {
			return nil, err
		}

		allAliases = append(allAliases, aliases...)
	}

	return allAliases, nil
}

// sudoersLine represents a parsed line from a sudoers file
type sudoersLine struct {
	entryType   string
	users       []string
	hosts       []string
	runasUsers  []string
	runasGroups []string
	tags        []string
	commands    []string
}

// parseUserSpecs parses user specification entries from sudoers content
func parseUserSpecs(runtime *plugin.Runtime, filePath string, content string) ([]any, error) {
	var entries []any
	lines := strings.Split(content, "\n")

	// Track line continuations
	var continuedLine string
	var continuedLineNum int

	for lineNum, line := range lines {
		actualLineNum := lineNum + 1

		// Handle line continuations
		if strings.HasSuffix(strings.TrimSpace(line), "\\") {
			if continuedLine == "" {
				continuedLineNum = actualLineNum
			}
			continuedLine += strings.TrimSuffix(strings.TrimSpace(line), "\\") + " "
			continue
		}

		// If we were continuing a line, append this final part
		if continuedLine != "" {
			line = continuedLine + line
			actualLineNum = continuedLineNum
			continuedLine = ""
		}

		// Skip empty lines and comments
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse the line
		parsed := parseSudoersLine(line)
		if parsed == nil || parsed.entryType != "user_spec" {
			continue
		}

		// Convert to string slices for llx
		users := make([]any, len(parsed.users))
		for i, u := range parsed.users {
			users[i] = u
		}

		hosts := make([]any, len(parsed.hosts))
		for i, h := range parsed.hosts {
			hosts[i] = h
		}

		runasUsers := make([]any, len(parsed.runasUsers))
		for i, u := range parsed.runasUsers {
			runasUsers[i] = u
		}

		runasGroups := make([]any, len(parsed.runasGroups))
		for i, g := range parsed.runasGroups {
			runasGroups[i] = g
		}

		tags := make([]any, len(parsed.tags))
		for i, t := range parsed.tags {
			tags[i] = t
		}

		commands := make([]any, len(parsed.commands))
		for i, c := range parsed.commands {
			commands[i] = c
		}

		entry, err := CreateResource(runtime, "sudoers.userSpec", map[string]*llx.RawData{
			"file":        llx.StringData(filePath),
			"lineNumber":  llx.IntData(int64(actualLineNum)),
			"users":       llx.ArrayData(users, types.String),
			"hosts":       llx.ArrayData(hosts, types.String),
			"runasUsers":  llx.ArrayData(runasUsers, types.String),
			"runasGroups": llx.ArrayData(runasGroups, types.String),
			"tags":        llx.ArrayData(tags, types.String),
			"commands":    llx.ArrayData(commands, types.String),
		})
		if err != nil {
			return nil, err
		}

		entries = append(entries, entry.(*mqlSudoersUserSpec))
	}

	return entries, nil
}

// parseDefaults parses default entries from sudoers content
func parseDefaults(runtime *plugin.Runtime, filePath string, content string) ([]any, error) {
	var defaults []any
	lines := strings.Split(content, "\n")

	// Track line continuations
	var continuedLine string
	var continuedLineNum int

	for lineNum, line := range lines {
		actualLineNum := lineNum + 1

		// Handle line continuations
		if strings.HasSuffix(strings.TrimSpace(line), "\\") {
			if continuedLine == "" {
				continuedLineNum = actualLineNum
			}
			continuedLine += strings.TrimSuffix(strings.TrimSpace(line), "\\") + " "
			continue
		}

		// If we were continuing a line, append this final part
		if continuedLine != "" {
			line = continuedLine + line
			actualLineNum = continuedLineNum
			continuedLine = ""
		}

		// Skip empty lines and comments
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Only process Defaults lines
		if !defaultsRegex.MatchString(line) {
			continue
		}

		// Parse the Defaults line
		scope, target, parameter, value, operation, negated := parseDefaultsLine(line)

		entry, err := CreateResource(runtime, "sudoers.default", map[string]*llx.RawData{
			"file":       llx.StringData(filePath),
			"lineNumber": llx.IntData(int64(actualLineNum)),
			"raw":        llx.StringData(line),
			"scope":      llx.StringData(scope),
			"target":     llx.StringData(target),
			"parameter":  llx.StringData(parameter),
			"value":      llx.StringData(value),
			"operation":  llx.StringData(operation),
			"negated":    llx.BoolData(negated),
		})
		if err != nil {
			return nil, err
		}

		defaults = append(defaults, entry.(*mqlSudoersDefault))
	}

	return defaults, nil
}

// parseSudoersAliases parses alias definitions from sudoers content
func parseSudoersAliases(runtime *plugin.Runtime, filePath string, content string) ([]any, error) {
	var aliases []any
	lines := strings.Split(content, "\n")

	// Track line continuations
	var continuedLine string
	var continuedLineNum int

	for lineNum, line := range lines {
		actualLineNum := lineNum + 1

		// Handle line continuations
		if strings.HasSuffix(strings.TrimSpace(line), "\\") {
			if continuedLine == "" {
				continuedLineNum = actualLineNum
			}
			continuedLine += strings.TrimSuffix(strings.TrimSpace(line), "\\") + " "
			continue
		}

		// If we were continuing a line, append this final part
		if continuedLine != "" {
			line = continuedLine + line
			actualLineNum = continuedLineNum
			continuedLine = ""
		}

		// Skip empty lines and comments
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for alias definitions
		matches := sudoersAliasRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		aliasType := matches[1] // User_Alias, Host_Alias, etc.
		aliasName := matches[2]
		aliasValue := matches[3]

		// Convert alias type to lowercase without "_Alias" suffix
		typeStr := strings.ToLower(strings.TrimSuffix(aliasType, "_Alias"))

		// Parse members
		memberList := splitAndTrim(aliasValue, ",")
		members := make([]any, len(memberList))
		for i, m := range memberList {
			members[i] = m
		}

		entry, err := CreateResource(runtime, "sudoers.alias", map[string]*llx.RawData{
			"file":       llx.StringData(filePath),
			"lineNumber": llx.IntData(int64(actualLineNum)),
			"type":       llx.StringData(typeStr),
			"name":       llx.StringData(aliasName),
			"members":    llx.ArrayData(members, types.String),
		})
		if err != nil {
			return nil, err
		}

		aliases = append(aliases, entry.(*mqlSudoersAlias))
	}

	return aliases, nil
}

// parseDefaultsLine parses a Defaults line and extracts its components
// Returns: scope, target, parameter, value, operation, negated
func parseDefaultsLine(line string) (string, string, string, string, string, bool) {
	// Strip "Defaults" prefix
	line = strings.TrimSpace(strings.TrimPrefix(line, "Defaults"))

	scope := "global"
	target := ""

	// Check for scope specifiers
	if len(line) > 0 {
		switch line[0] {
		case ':': // User-specific
			scope = "user"
			// Extract target until first whitespace
			parts := strings.SplitN(line[1:], " ", 2)
			target = parts[0]
			if len(parts) > 1 {
				line = parts[1]
			} else {
				line = ""
			}
		case '@': // Host-specific
			scope = "host"
			parts := strings.SplitN(line[1:], " ", 2)
			target = parts[0]
			if len(parts) > 1 {
				line = parts[1]
			} else {
				line = ""
			}
		case '>': // Runas-specific
			scope = "runas"
			parts := strings.SplitN(line[1:], " ", 2)
			target = parts[0]
			if len(parts) > 1 {
				line = parts[1]
			} else {
				line = ""
			}
		case '!': // Command-specific
			scope = "command"
			parts := strings.SplitN(line[1:], " ", 2)
			target = parts[0]
			if len(parts) > 1 {
				line = parts[1]
			} else {
				line = ""
			}
		}
	}

	line = strings.TrimSpace(line)

	// Check for negation
	negated := false
	if strings.HasPrefix(line, "!") {
		negated = true
		line = strings.TrimPrefix(line, "!")
	}

	// Parse parameter[operator]value
	parameter := ""
	value := ""
	operation := ""

	// Check for operators: =, +=, -=
	if idx := strings.Index(line, "+="); idx != -1 {
		parameter = strings.TrimSpace(line[:idx])
		value = strings.TrimSpace(line[idx+2:])
		operation = "+="
	} else if idx := strings.Index(line, "-="); idx != -1 {
		parameter = strings.TrimSpace(line[:idx])
		value = strings.TrimSpace(line[idx+2:])
		operation = "-="
	} else if idx := strings.Index(line, "="); idx != -1 {
		parameter = strings.TrimSpace(line[:idx])
		value = strings.TrimSpace(line[idx+1:])
		operation = "="
	} else {
		// No operator, just a parameter (boolean flag)
		parameter = strings.TrimSpace(line)
		operation = ""
	}

	// Remove quotes from value if present
	value = strings.Trim(value, "\"")

	return scope, target, parameter, value, operation, negated
}

// parseSudoersLine parses a single line from a sudoers file
func parseSudoersLine(line string) *sudoersLine {
	// Filter out comments
	if strings.HasPrefix(line, "#") {
		return nil
	}

	// Check for Defaults entries
	if defaultsRegex.MatchString(line) {
		return &sudoersLine{
			entryType: "defaults",
			commands:  []string{strings.TrimSpace(strings.TrimPrefix(line, "Defaults"))},
		}
	}

	// Check for alias definitions
	if matches := sudoersAliasRegex.FindStringSubmatch(line); matches != nil {
		aliasType := matches[1]
		aliasName := matches[2]
		aliasValue := matches[3]

		return &sudoersLine{
			entryType: strings.ToLower(aliasType),
			users:     []string{aliasName},
			commands:  splitAndTrim(aliasValue, ","),
		}
	}

	// Parse user specification
	// Format: user host=(runasuser:runasgroup) tag: command
	result := &sudoersLine{
		entryType: "user_spec",
	}

	// First, extract runas specification from the line before splitting
	// This is important because the host can be followed directly by =(...)
	runasStart := strings.Index(line, "=(")
	var remaining string
	var beforeRunas string

	if runasStart != -1 {
		runasEnd := strings.Index(line[runasStart:], ")")
		if runasEnd != -1 {
			runasEnd += runasStart
			runasSpec := line[runasStart+2 : runasEnd]

			// Parse runas users and groups
			if strings.Contains(runasSpec, ":") {
				runasParts := strings.SplitN(runasSpec, ":", 2)
				result.runasUsers = splitAndTrim(runasParts[0], ",")
				result.runasGroups = splitAndTrim(runasParts[1], ",")
			} else {
				result.runasUsers = splitAndTrim(runasSpec, ",")
			}

			// Split line into parts: before runas, and after runas
			beforeRunas = strings.TrimSpace(line[:runasStart])
			remaining = strings.TrimSpace(line[runasEnd+1:])
		}
	} else {
		// No runas specification, need to find where host ends
		// Look for the first whitespace after the second token
		parts := smartSplit(line)
		if len(parts) < 3 {
			return nil
		}

		// First part is user, second is host, rest is commands
		result.users = splitAndTrim(parts[0], ",")
		result.hosts = splitAndTrim(parts[1], ",")
		remaining = strings.Join(parts[2:], " ")

		// Extract tags and commands from remaining
		extractTagsAndCommands(result, remaining)
		return result
	}

	// For lines with runas, split beforeRunas to get user and host
	// The format is: "user host" or "user1, user2 host1, host2"
	// We need to find where user ends and host begins
	tokens := smartSplit(beforeRunas)
	if len(tokens) < 2 {
		return nil
	}

	// Find the boundary between user list and host list
	// Tokens ending with comma (after trim) are part of a multi-item list
	// We need to find the last group of comma-separated tokens - that's the host list

	// First, identify which tokens are part of comma-separated groups
	// by checking if they end with a comma or if the next token follows a comma
	var userEndIndex int
	for i := len(tokens) - 1; i >= 0; i-- {
		token := strings.TrimSpace(tokens[i])
		// If this token ends with comma, the group extends backwards
		if strings.HasSuffix(token, ",") {
			continue
		}
		// If the previous token ends with comma, this is part of that group
		if i > 0 && strings.HasSuffix(strings.TrimSpace(tokens[i-1]), ",") {
			continue
		}
		// This token is not part of a comma group
		// Check if we've already found any comma-group tokens
		if i < len(tokens)-1 {
			// We found the boundary - everything from 0 to i is users, i+1 to end is hosts
			userEndIndex = i
			break
		}
		// This is the last token and has no comma - it's the only host
		userEndIndex = i - 1
		break
	}

	if userEndIndex < 0 {
		// All tokens were comma-separated, which shouldn't happen in valid sudoers
		return nil
	}

	// Split into user and host parts
	userTokens := tokens[:userEndIndex+1]
	hostTokens := tokens[userEndIndex+1:]

	// Join and parse
	userPart := strings.Join(userTokens, " ")
	result.users = splitAndTrim(userPart, ",")

	hostPart := strings.Join(hostTokens, " ")
	result.hosts = splitAndTrim(hostPart, ",")

	// Extract tags and commands from remaining
	extractTagsAndCommands(result, remaining)
	return result
}

// extractTagsAndCommands extracts tags and commands from a sudoers line
func extractTagsAndCommands(result *sudoersLine, remaining string) {
	// Extract tags (NOPASSWD, PASSWD, NOEXEC, EXEC, SETENV, NOSETENV, etc.)
	tagRegex := regexp.MustCompile(`\b(NOPASSWD|PASSWD|NOEXEC|EXEC|SETENV|NOSETENV|LOG_INPUT|NOLOG_INPUT|LOG_OUTPUT|NOLOG_OUTPUT|MAIL|NOMAIL|FOLLOW|NOFOLLOW|INTERCEPT|NOINTERCEPT)\s*:\s*`)
	for {
		match := tagRegex.FindStringIndex(remaining)
		if match == nil {
			break
		}
		tag := strings.TrimSpace(remaining[match[0]:match[1]])
		tag = strings.TrimSuffix(tag, ":")
		result.tags = append(result.tags, strings.TrimSpace(tag))
		remaining = remaining[match[1]:]
	}

	// Remaining part is the command specification
	if remaining != "" {
		result.commands = splitCommands(remaining)
	}
}

// smartSplit splits a string by whitespace but respects quoted strings
func smartSplit(s string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	escaped := false

	for _, ch := range s {
		if escaped {
			current.WriteRune(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			current.WriteRune(ch)
			continue
		}

		if ch == '"' {
			inQuote = !inQuote
			current.WriteRune(ch)
			continue
		}

		if ch == ' ' && !inQuote {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(ch)
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// splitCommands splits command specifications by comma
func splitCommands(s string) []string {
	var commands []string
	var current strings.Builder
	inQuote := false
	escaped := false

	for _, ch := range s {
		if escaped {
			current.WriteRune(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			current.WriteRune(ch)
			continue
		}

		if ch == '"' {
			inQuote = !inQuote
			current.WriteRune(ch)
			continue
		}

		if ch == ',' && !inQuote {
			if current.Len() > 0 {
				commands = append(commands, strings.TrimSpace(current.String()))
				current.Reset()
			}
			continue
		}

		current.WriteRune(ch)
	}

	if current.Len() > 0 {
		commands = append(commands, strings.TrimSpace(current.String()))
	}

	return commands
}

// splitAndTrim splits a string by the given separator and trims each part
func splitAndTrim(s string, sep string) []string {
	if s == "" {
		return []string{}
	}

	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
