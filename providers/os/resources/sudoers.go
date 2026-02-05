// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v12/checksums"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/resources/sudoers"
	"go.mondoo.com/cnquery/v12/types"
)

const (
	defaultSudoersFile = "/etc/sudoers"
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
		if matches := sudoers.IncludeRegex.FindStringSubmatch(line); matches != nil {
			includePath := strings.TrimSpace(matches[1])
			if err := s.collectSudoersFiles(conn, includePath, visited, allFiles); err != nil {
				// Continue on error - included file might not exist
				continue
			}
		}

		// Check for @includedir or #includedir
		if matches := sudoers.IncludedirRegex.FindStringSubmatch(line); matches != nil {
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

		parsed := sudoers.ParseUserSpecs(file.Path.Data, string(raw))

		for _, spec := range parsed {
			entry, err := CreateResource(s.MqlRuntime, "sudoers.userSpec", map[string]*llx.RawData{
				"file":        llx.StringData(spec.File),
				"lineNumber":  llx.IntData(int64(spec.LineNumber)),
				"users":       llx.ArrayData(toAnySlice(spec.Users), types.String),
				"hosts":       llx.ArrayData(toAnySlice(spec.Hosts), types.String),
				"runasUsers":  llx.ArrayData(toAnySlice(spec.RunasUsers), types.String),
				"runasGroups": llx.ArrayData(toAnySlice(spec.RunasGroups), types.String),
				"tags":        llx.ArrayData(toAnySlice(spec.Tags), types.String),
				"commands":    llx.ArrayData(toAnySlice(spec.Commands), types.String),
			})
			if err != nil {
				return nil, err
			}

			allEntries = append(allEntries, entry.(*mqlSudoersUserSpec))
		}
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

		parsed := sudoers.ParseDefaults(file.Path.Data, string(raw))

		for _, def := range parsed {
			entry, err := CreateResource(s.MqlRuntime, "sudoers.default", map[string]*llx.RawData{
				"file":       llx.StringData(def.File),
				"lineNumber": llx.IntData(int64(def.LineNumber)),
				"raw":        llx.StringData(def.Raw),
				"scope":      llx.StringData(def.Scope),
				"target":     llx.StringData(def.Target),
				"parameter":  llx.StringData(def.Parameter),
				"value":      llx.StringData(def.Value),
				"operation":  llx.StringData(def.Operation),
				"negated":    llx.BoolData(def.Negated),
			})
			if err != nil {
				return nil, err
			}

			allDefaults = append(allDefaults, entry.(*mqlSudoersDefault))
		}
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

		parsed := sudoers.ParseAliases(file.Path.Data, string(raw))

		for _, alias := range parsed {
			entry, err := CreateResource(s.MqlRuntime, "sudoers.alias", map[string]*llx.RawData{
				"file":       llx.StringData(alias.File),
				"lineNumber": llx.IntData(int64(alias.LineNumber)),
				"type":       llx.StringData(alias.Type),
				"name":       llx.StringData(alias.Name),
				"members":    llx.ArrayData(toAnySlice(alias.Members), types.String),
			})
			if err != nil {
				return nil, err
			}

			allAliases = append(allAliases, entry.(*mqlSudoersAlias))
		}
	}

	return allAliases, nil
}

// toAnySlice converts a []string to []any
func toAnySlice(s []string) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}
