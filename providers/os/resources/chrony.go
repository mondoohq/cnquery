// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// chronyConfPaths lists the default chrony configuration file locations.
// RHEL/Fedora/SUSE ship /etc/chrony.conf; Debian/Ubuntu use
// /etc/chrony/chrony.conf. The first one that exists wins.
var chronyConfPaths = []string{
	"/etc/chrony.conf",
	"/etc/chrony/chrony.conf",
}

func initChronyConf(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' in chrony.conf initialization, it must be a string")
		}

		f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path": llx.StringData(path),
		})
		if err != nil {
			return nil, nil, err
		}
		args["file"] = llx.ResourceData(f, "file")
		delete(args, "path")
	}

	return args, nil, nil
}

func (s *mqlChronyConf) id() (string, error) {
	file := s.GetFile()
	if file.Error != nil {
		return "", file.Error
	}
	if file.Data == nil {
		return "", errors.New("cannot get file for chrony.conf")
	}
	return file.Data.Path.Data, nil
}

func (s *mqlChronyConf) file() (*mqlFile, error) {
	for _, candidate := range chronyConfPaths {
		f, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
			"path": llx.StringData(candidate),
		})
		if err != nil {
			return nil, err
		}
		mqlFile := f.(*mqlFile)
		if exists := mqlFile.GetExists(); exists.Error == nil && exists.Data {
			return mqlFile, nil
		}
	}

	// none of the candidates exist; return the primary path so callers can
	// still inspect the (missing) file rather than erroring out
	f, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(chronyConfPaths[0]),
	})
	if err != nil {
		return nil, err
	}
	return f.(*mqlFile), nil
}

func (s *mqlChronyConf) content(file *mqlFile) (string, error) {
	return fileContentOrEmpty(file)
}

// settings strips comments (lines starting with '#' or '!') and blank
// lines, returning the remaining directives trimmed of surrounding
// whitespace.
func (s *mqlChronyConf) settings(content string) ([]any, error) {
	lines := strings.Split(content, "\n")

	settings := []any{}
	for i := range lines {
		line := strings.TrimSpace(lines[i])
		if line == "" || line[0] == '#' || line[0] == '!' {
			continue
		}
		settings = append(settings, line)
	}

	return settings, nil
}

// directiveValues returns the argument portion of every setting whose
// first token matches the given directive (case-insensitive).
func directiveValues(settings []any, directive string) []any {
	res := []any{}
	for i := range settings {
		line, ok := settings[i].(string)
		if !ok {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if !strings.EqualFold(fields[0], directive) {
			continue
		}
		res = append(res, strings.TrimSpace(strings.TrimPrefix(line, fields[0])))
	}
	return res
}

// lastDirectiveValue returns the argument portion of the last occurrence
// of a single-value directive (chrony uses the last setting for these),
// or "" when the directive is absent.
func lastDirectiveValue(settings []any, directive string) string {
	values := directiveValues(settings, directive)
	if len(values) == 0 {
		return ""
	}
	return values[len(values)-1].(string)
}

func (s *mqlChronyConf) servers(settings []any) ([]any, error) {
	return directiveValues(settings, "server"), nil
}

func (s *mqlChronyConf) pools(settings []any) ([]any, error) {
	return directiveValues(settings, "pool"), nil
}

func (s *mqlChronyConf) peers(settings []any) ([]any, error) {
	return directiveValues(settings, "peer"), nil
}

func (s *mqlChronyConf) allow(settings []any) ([]any, error) {
	return directiveValues(settings, "allow"), nil
}

func (s *mqlChronyConf) deny(settings []any) ([]any, error) {
	return directiveValues(settings, "deny"), nil
}

func (s *mqlChronyConf) bindCmdAddresses(settings []any) ([]any, error) {
	return directiveValues(settings, "bindcmdaddress"), nil
}

func (s *mqlChronyConf) keyFile(settings []any) (string, error) {
	return lastDirectiveValue(settings, "keyfile"), nil
}

func (s *mqlChronyConf) makeStep(settings []any) (string, error) {
	return lastDirectiveValue(settings, "makestep"), nil
}

// rtcSync reports whether the bare `rtcsync` directive is present.
func (s *mqlChronyConf) rtcSync(settings []any) (bool, error) {
	for i := range settings {
		line, ok := settings[i].(string)
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(line), "rtcsync") {
			return true, nil
		}
	}
	return false, nil
}
