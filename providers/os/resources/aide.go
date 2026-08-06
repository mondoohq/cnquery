// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"path"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/types"
)

var (
	// aideConfigCandidates are the root configuration locations, in the order
	// they are probed. Debian-family packages use the first, RedHat-family and
	// SUSE the second.
	aideConfigCandidates = []string{
		"/etc/aide/aide.conf",
		"/etc/aide.conf",
		"/usr/local/etc/aide.conf",
	}

	aideBinaries = []string{
		"/usr/sbin/aide",
		"/usr/bin/aide",
		"/usr/local/bin/aide",
	}
)

type mqlAideInternal struct {
	lock        sync.Mutex
	loaded      atomic.Bool
	cfg         *aideConfig
	parsedFiles []*mqlFile
	err         error
}

func (a *mqlAide) id() (string, error) {
	return "aide", nil
}

func (a *mqlAide) fs() (afero.Fs, error) {
	conn, ok := a.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		return nil, errors.New("aide is not supported on this connection")
	}
	return conn.FileSystem(), nil
}

// load parses the configuration once and shares the result across every field,
// so a query touching params, rules, and the database reads the files a single
// time.
func (a *mqlAide) load() (*aideConfig, []*mqlFile, error) {
	if a.loaded.Load() {
		return a.cfg, a.parsedFiles, a.err
	}

	a.lock.Lock()
	defer a.lock.Unlock()
	if a.loaded.Load() {
		return a.cfg, a.parsedFiles, a.err
	}

	cfg, files, err := a.readConfig()
	a.cfg, a.parsedFiles, a.err = cfg, files, err
	a.loaded.Store(true)

	return cfg, files, err
}

func (a *mqlAide) readConfig() (*aideConfig, []*mqlFile, error) {
	fs, err := a.fs()
	if err != nil {
		return nil, nil, err
	}

	cfg := newAideConfig()
	files := []*mqlFile{}

	root := a.findConfigFile(fs)
	if root == "" {
		return cfg, files, nil
	}

	// a configuration can include a file more than once; parsing it twice would
	// duplicate every rule it holds
	visited := map[string]struct{}{}

	read := func(filePath string) (aideIncludeFile, bool) {
		if _, seen := visited[filePath]; seen {
			return aideIncludeFile{}, false
		}
		visited[filePath] = struct{}{}

		file, err := newFile(a.MqlRuntime, filePath)
		if err != nil {
			log.Debug().Err(err).Str("file", filePath).Msg("aide> cannot create file resource")
			return aideIncludeFile{}, false
		}

		content, err := fileContentOrEmpty(file)
		if err != nil {
			log.Debug().Err(err).Str("file", filePath).Msg("aide> cannot read configuration file")
			return aideIncludeFile{}, false
		}

		files = append(files, file)
		return aideIncludeFile{Path: filePath, Content: content}, true
	}

	var resolve aideIncludeResolver = func(target string) []aideIncludeFile {
		res := []aideIncludeFile{}

		isDir, err := afero.IsDir(fs, target)
		if err != nil {
			log.Debug().Err(err).Str("target", target).Msg("aide> cannot stat include target")
			return res
		}

		if !isDir {
			if included, ok := read(target); ok {
				res = append(res, included)
			}
			return res
		}

		entries, err := afero.ReadDir(fs, target)
		if err != nil {
			log.Debug().Err(err).Str("target", target).Msg("aide> cannot list include directory")
			return res
		}

		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			names = append(names, entry.Name())
		}
		// AIDE reads the entries of an include directory in sorted order
		sort.Strings(names)

		for _, name := range names {
			if included, ok := read(path.Join(target, name)); ok {
				res = append(res, included)
			}
		}
		return res
	}

	rootFile, ok := read(root)
	if !ok {
		return cfg, files, nil
	}

	parseAideConfig(cfg, rootFile.Path, rootFile.Content, 0, resolve)

	return cfg, files, nil
}

func (a *mqlAide) findConfigFile(fs afero.Fs) string {
	for _, candidate := range aideConfigCandidates {
		exists, err := afero.Exists(fs, candidate)
		if err != nil {
			log.Debug().Err(err).Str("path", candidate).Msg("aide> cannot check path")
			continue
		}
		if exists {
			return candidate
		}
	}
	return ""
}

func (a *mqlAide) installed() (bool, error) {
	fs, err := a.fs()
	if err != nil {
		return false, err
	}

	if a.findConfigFile(fs) != "" {
		return true, nil
	}

	for _, binary := range aideBinaries {
		exists, err := afero.Exists(fs, binary)
		if err != nil {
			continue
		}
		if exists {
			return true, nil
		}
	}

	return false, nil
}

func (a *mqlAide) version() (string, error) {
	// no point running the binary on a host that carries no AIDE at all; the
	// version is simply unknown there
	installed, err := a.installed()
	if err != nil {
		return "", err
	}
	if !installed {
		a.Version.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}

	o, err := CreateResource(a.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("aide --version"),
	})
	if err != nil {
		return "", err
	}
	cmd := o.(*mqlCommand)

	// a backend that cannot run commands, such as an image scan, leaves the
	// version unknown rather than wrong
	exit := cmd.GetExitcode()
	if exit.Error != nil {
		log.Debug().Err(exit.Error).Msg("aide> cannot run aide")
		a.Version.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}

	stdout := cmd.GetStdout()
	if stdout.Error != nil {
		return "", stdout.Error
	}

	// aide reports its version on stderr on some releases and exits non-zero on
	// others, so the output is what decides, not the exit code
	out := stdout.Data
	if out == "" {
		if stderr := cmd.GetStderr(); stderr.Error == nil {
			out = stderr.Data
		}
	}

	version := parseAideVersion(out)
	if version == "" {
		a.Version.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}

	return version, nil
}

func (a *mqlAide) configFile() (*mqlFile, error) {
	fs, err := a.fs()
	if err != nil {
		return nil, err
	}

	root := a.findConfigFile(fs)
	if root == "" {
		a.ConfigFile.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	return newFile(a.MqlRuntime, root)
}

func (a *mqlAide) files() ([]any, error) {
	_, files, err := a.load()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(files))
	for _, file := range files {
		res = append(res, file)
	}
	return res, nil
}

func (a *mqlAide) params() (map[string]any, error) {
	cfg, _, err := a.load()
	if err != nil {
		return nil, err
	}
	return convert.MapToInterfaceMap(cfg.Params), nil
}

func (a *mqlAide) groups() (map[string]any, error) {
	cfg, _, err := a.load()
	if err != nil {
		return nil, err
	}
	return convert.MapToInterfaceMap(cfg.Groups), nil
}

func (a *mqlAide) rules() ([]any, error) {
	cfg, files, err := a.load()
	if err != nil {
		return nil, err
	}

	// reuse the file resources the parse already created rather than building a
	// second one per rule
	byPath := make(map[string]*mqlFile, len(files))
	for _, file := range files {
		byPath[file.Path.Data] = file
	}

	res := make([]any, 0, len(cfg.Rules))
	for i := range cfg.Rules {
		rule := cfg.Rules[i]

		file, ok := byPath[rule.File]
		if !ok {
			file, err = newFile(a.MqlRuntime, rule.File)
			if err != nil {
				return nil, err
			}
			byPath[rule.File] = file
		}

		// the same path can be selected by several lines, so the source location
		// is what keeps the cache key unique
		id := rule.File + ":" + strconv.Itoa(rule.LineNumber) + ":" + rule.Path

		resource, err := CreateResource(a.MqlRuntime, "aide.rule", map[string]*llx.RawData{
			"__id":       llx.StringData(id),
			"path":       llx.StringData(rule.Path),
			"selection":  llx.StringData(rule.Selection),
			"expression": llx.StringData(rule.Expression),
			"attributes": llx.ArrayData(convert.SliceAnyToInterface(rule.Attributes), types.String),
			"lineNumber": llx.IntData(int64(rule.LineNumber)),
			"file":       llx.ResourceData(file, "file"),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, resource)
	}

	return res, nil
}

func (a *mqlAide) database() (*mqlFile, error) {
	return a.databaseFrom("database_in", "database", &a.Database)
}

func (a *mqlAide) newDatabase() (*mqlFile, error) {
	return a.databaseFrom("database_out", "", &a.NewDatabase)
}

// databaseFrom resolves a database setting into a file, falling back to the
// legacy option name when the modern one is absent.
func (a *mqlAide) databaseFrom(option string, legacyOption string, field *plugin.TValue[*mqlFile]) (*mqlFile, error) {
	cfg, _, err := a.load()
	if err != nil {
		return nil, err
	}

	value := cfg.Params[option]
	if value == "" && legacyOption != "" {
		value = cfg.Params[legacyOption]
	}

	dbPath := aideDatabasePath(value)
	if dbPath == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	return newFile(a.MqlRuntime, dbPath)
}
