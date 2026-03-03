// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/resources/logrotate"
	"go.mondoo.com/mql/v13/types"
)

const (
	defaultLogrotateConf = "/etc/logrotate.conf"
	defaultLogrotateDir  = "/etc/logrotate.d"
)

func initLogrotate(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return args, nil, nil
}

func (l *mqlLogrotate) id() (string, error) {
	return "logrotate", nil
}

func (le *mqlLogrotateEntry) id() (string, error) {
	file := le.File.Data
	lineNum := strconv.FormatInt(le.LineNumber.Data, 10)

	return file.Path.Data + ":" + lineNum + ":" + le.Path.Data, nil
}

// files discovers logrotate configuration files from the main config and logrotate.d directory.
func (l *mqlLogrotate) files() ([]any, error) {
	var allFiles []any

	// Add main logrotate.conf
	mainFile, err := CreateResource(l.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(defaultLogrotateConf),
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

	// Check logrotate.d directory
	dirFile, err := CreateResource(l.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(defaultLogrotateDir),
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
		files, err := CreateResource(l.MqlRuntime, "files.find", map[string]*llx.RawData{
			"from": llx.StringData(defaultLogrotateDir),
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

		for i := range list.Data {
			file := list.Data[i].(*mqlFile)
			basename := file.GetBasename()
			if basename.Error != nil {
				continue
			}

			// Skip common backup/temp file extensions that logrotate ignores
			name := basename.Data
			if strings.HasSuffix(name, ".bak") || strings.HasSuffix(name, ".old") ||
				strings.HasSuffix(name, ".rpmsave") || strings.HasSuffix(name, ".rpmorig") ||
				strings.HasSuffix(name, ".dpkg-old") || strings.HasSuffix(name, ".dpkg-new") ||
				strings.HasSuffix(name, ".dpkg-dist") || strings.HasSuffix(name, "~") {
				continue
			}

			allFiles = append(allFiles, file)
		}
	}

	return allFiles, nil
}

// globalConfig parses all config files and returns the global directives.
func (l *mqlLogrotate) globalConfig(files []any) (map[string]any, error) {
	merged := make(map[string]any)

	for i := range files {
		file := files[i].(*mqlFile)
		content := file.GetContent()
		if content.Error != nil {
			continue
		}

		global, _ := logrotate.ParseContent(file.Path.Data, content.Data)
		for k, v := range global {
			merged[k] = v
		}
	}

	return merged, nil
}

// entries parses all config files and returns logrotate entries.
func (l *mqlLogrotate) entries(files []any) ([]any, error) {
	var allEntries []any
	var errs []error

	for i := range files {
		file := files[i].(*mqlFile)

		content := file.GetContent()
		if content.Error != nil {
			errs = append(errs, fmt.Errorf("failed to read %s: %w", file.Path.Data, content.Error))
			continue
		}

		entries, err := parseLogrotateContent(l.MqlRuntime, file, content.Data)
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

func parseLogrotateContent(runtime *plugin.Runtime, file *mqlFile, content string) ([]any, error) {
	_, parsed := logrotate.ParseContent(file.Path.Data, content)
	var entries []any

	for _, e := range parsed {
		configMap := make(map[string]any, len(e.Config))
		for k, v := range e.Config {
			configMap[k] = v
		}

		entry, err := CreateResource(runtime, "logrotate.entry", map[string]*llx.RawData{
			"file":       llx.ResourceData(file, "file"),
			"lineNumber": llx.IntData(int64(e.LineNumber)),
			"path":       llx.StringData(e.Path),
			"config":     llx.MapData(configMap, types.String),
		})
		if err != nil {
			return nil, err
		}

		entries = append(entries, entry.(*mqlLogrotateEntry))
	}

	return entries, nil
}
