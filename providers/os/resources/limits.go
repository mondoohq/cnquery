// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
)

const (
	defaultLimitsFile = "/etc/security/limits.conf"
	defaultLimitsDir  = "/etc/security/limits.d"
)

var (
	// Regular expression for parsing limits entries
	// Format: <domain> <type> <item> <value>
	limitsEntryRegex = regexp.MustCompile(`^(\S+)\s+(soft|hard|-)\s+(\S+)\s+(\S+)`)
)

func initLimits(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return args, nil, nil
}

func (l *mqlLimits) id() (string, error) {
	return "limits", nil
}

func (le *mqlLimitsEntry) id() (string, error) {
	file := le.File.Data
	lineNum := strconv.FormatInt(le.LineNumber.Data, 10)

	// Create unique ID from file path and line number
	id := file + ":" + lineNum

	return id, nil
}

// files returns the list of limits configuration files
func (l *mqlLimits) files() ([]any, error) {
	var allFiles []any

	// Add main limits file
	mainFile, err := CreateResource(l.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(defaultLimitsFile),
	})
	if err != nil {
		return nil, err
	}
	f := mainFile.(*mqlFile)
	exists := f.GetExists()
	if exists.Error != nil {
		return nil, exists.Error
	}

	if exists.Data {
		allFiles = append(allFiles, f)
	}

	// Check if limits.d directory exists
	dirFile, err := CreateResource(l.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(defaultLimitsDir),
	})
	if err != nil {
		return nil, err
	}
	dir := dirFile.(*mqlFile)
	dirExists := dir.GetExists()
	if dirExists.Error != nil {
		return nil, dirExists.Error
	}

	if dirExists.Data {
		// Get all files from limits.d directory
		files, err := CreateResource(l.MqlRuntime, "files.find", map[string]*llx.RawData{
			"from": llx.StringData(defaultLimitsDir),
			"type": llx.StringData("file"),
		})
		if err != nil {
			return nil, err
		}

		ff := files.(*mqlFilesFind)
		list := ff.GetList()
		if list.Error != nil {
			return nil, list.Error
		}

		// Filter for .conf files from limits.d
		for i := range list.Data {
			file := list.Data[i].(*mqlFile)
			basename := file.GetBasename()
			if basename.Error != nil {
				continue
			}

			// Only include .conf files
			if strings.HasSuffix(basename.Data, ".conf") {
				allFiles = append(allFiles, file)
			}
		}
	}

	return allFiles, nil
}

// entries parses all limits files and returns structured entries
func (l *mqlLimits) entries(files []any) ([]any, error) {
	conn := l.MqlRuntime.Connection.(shared.Connection)

	var allEntries []any
	var errs []error

	for i := range files {
		file := files[i].(*mqlFile)

		f, err := conn.FileSystem().Open(file.Path.Data)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to open %s: %w", file.Path.Data, err))
			continue
		}

		raw, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to read %s: %w", file.Path.Data, err))
			continue
		}

		entries, err := parseLimitsContent(l.MqlRuntime, file.Path.Data, string(raw))
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to parse %s: %w", file.Path.Data, err))
			continue
		}

		allEntries = append(allEntries, entries...)
	}

	if len(errs) > 0 {
		return allEntries, errors.Join(errs...)
	}

	return allEntries, nil
}

// limitsEntry represents a parsed limits entry (used for testing)
type limitsEntry struct {
	File       string
	LineNumber int
	Domain     string
	Type       string
	Item       string
	Value      string
}

// parseLimitsLines parses the content of a limits file and returns structured entries
// This function is separated from resource creation for testability
func parseLimitsLines(filePath string, content string) []limitsEntry {
	var entries []limitsEntry
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		actualLineNum := lineNum + 1

		// Skip empty lines and comments
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse the line using regex
		matches := limitsEntryRegex.FindStringSubmatch(line)
		if matches == nil {
			// Invalid format, skip
			continue
		}

		entries = append(entries, limitsEntry{
			File:       filePath,
			LineNumber: actualLineNum,
			Domain:     matches[1],
			Type:       matches[2],
			Item:       matches[3],
			Value:      matches[4],
		})
	}

	return entries
}

// parseLimitsContent parses the content of a limits file and creates MQL resources
func parseLimitsContent(runtime *plugin.Runtime, filePath string, content string) ([]any, error) {
	parsed := parseLimitsLines(filePath, content)
	var entries []any

	for _, e := range parsed {
		entry, err := CreateResource(runtime, "limits.entry", map[string]*llx.RawData{
			"file":       llx.StringData(e.File),
			"lineNumber": llx.IntData(int64(e.LineNumber)),
			"domain":     llx.StringData(e.Domain),
			"type":       llx.StringData(e.Type),
			"item":       llx.StringData(e.Item),
			"value":      llx.StringData(e.Value),
		})
		if err != nil {
			return nil, err
		}

		entries = append(entries, entry.(*mqlLimitsEntry))
	}

	return entries, nil
}
