// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package resources

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode"

	"go.mondoo.com/cnquery/v12/checksums"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/os/resources/parsers"
	"go.mondoo.com/cnquery/v12/types"
	"go.mondoo.com/cnquery/v12/utils/multierr"
)

type mqlAuditdConfigInternal struct {
	lock sync.Mutex
}

func initAuditdConfig(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' in auditd.config initialization, it must be a string")
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

const defaultAuditdConfig = "/etc/audit/auditd.conf"

func (s *mqlAuditdConfig) id() (string, error) {
	file := s.GetFile()
	if file.Error != nil {
		return "", file.Error
	}

	return file.Data.Path.Data, nil
}

func (s *mqlAuditdConfig) file() (*mqlFile, error) {
	f, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(defaultAuditdConfig),
	})
	if err != nil {
		return nil, err
	}
	return f.(*mqlFile), nil
}

func (s *mqlAuditdConfig) parse(file *mqlFile) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if file == nil {
		return errors.New("no base auditd config file to read")
	}

	content := file.GetContent()
	if content.Error != nil {
		return content.Error
	}

	ini := parsers.ParseIni(content.Data, "=")

	res := make(map[string]any, len(ini.Fields))
	s.Params.Data = res
	s.Params.State = plugin.StateIsSet

	if len(ini.Fields) == 0 {
		return nil
	}

	root := ini.Fields[""]
	if root == nil {
		s.Params.Error = errors.New("failed to parse auditd config")
		return s.Params.Error
	}

	fields, ok := root.(map[string]any)
	if !ok {
		s.Params.Error = errors.New("failed to parse auditd config (invalid data retrieved)")
		return s.Params.Error
	}

	var errs multierr.Errors
	for k, v := range fields {
		key := strings.ToLower(k)
		if s, ok := v.(string); ok {
			if slices.Contains(auditdDowncaseKeywords, key) {
				res[key] = strings.ToLower(s)
			} else {
				res[key] = s
			}
		} else {
			errs.Add(fmt.Errorf("can't parse field '"+s+"', value is %+v", v))
		}
	}

	s.Params.Error = errs.Deduplicate()
	return s.Params.Error
}

func (s *mqlAuditdConfig) params(file *mqlFile) (map[string]any, error) {
	return nil, s.parse(file)
}

var auditdDowncaseKeywords = []string{
	"local_events",
	"write_logs",
	"log_format",
	"flush",
	"max_log_file_action",
	"verify_email",
	"space_left_action",
	"admin_space_left_action",
	"disk_full_action",
	"disk_error_action",
	"use_libwrap",
	"enable_krb5",
	"overflow_action",
}

type mqlAuditdRulesInternal struct {
	lock      sync.Mutex
	loaded    bool
	loadError error

	// Dual-source storage
	filesystemLock   sync.Mutex
	filesystemLoaded bool
	filesystemData   struct {
		controls []interface{}
		files    []interface{}
		syscalls []interface{}
	}
	filesystemError error

	runtimeLock   sync.Mutex
	runtimeLoaded bool
	runtimeData   struct {
		controls []interface{}
		files    []interface{}
		syscalls []interface{}
	}
	runtimeError error
}

const (
	defaultAuditdRules = "/etc/audit/rules.d"
	defaultSource      = "both"
)

func initAuditdRules(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// Set default source if not provided
	if _, ok := args["source"]; !ok {
		args["source"] = llx.StringData(defaultSource)
	} else {
		// Validate source value
		if x, ok := args["source"]; ok {
			source, ok := x.Value.(string)
			if !ok {
				return nil, nil, errors.New("wrong type for 'source' in auditd.rules initialization, it must be a string")
			}
			if source != "filesystem" && source != "runtime" && source != "both" {
				return nil, nil, errors.New("source must be 'filesystem', 'runtime', or 'both'")
			}
		}
	}
	return args, nil, nil
}

func (s *mqlAuditdRules) id() (string, error) {
	// Include both path and source in the ID to ensure different source parameters
	// create separate resource instances
	return s.Path.Data + "\x00" + s.Source.Data, nil
}

func (s *mqlAuditdRules) path() (string, error) {
	return defaultAuditdRules, nil
}

// loadBothSources loads and merges rules from both filesystem and runtime
// When source="both", this implements intelligent fallback:
// - On live systems (with run-command capability): loads both and requires logical AND
// - On non-live systems (no capability): gracefully falls back to filesystem only
// - If auditctl command doesn't exist: gracefully falls back to filesystem only
func (s *mqlAuditdRules) loadBothSources(path string) error {
	var fsErr, rtErr error

	// Load filesystem rules
	fsErr = s.loadFilesystemRules(path)

	// Check if runtime is even available
	hasRuntime := s.hasRunCommandCapability()

	if hasRuntime {
		// Load runtime rules
		rtErr = s.loadRuntimeRules()

		// Check if runtime error is "command not found" - if so, treat as not available
		isCommandNotFound := rtErr != nil && (strings.Contains(rtErr.Error(), "command not found") ||
			strings.Contains(rtErr.Error(), "executable file not found"))

		if isCommandNotFound {
			// auditctl not installed - fall back to filesystem only
			if fsErr != nil {
				return fmt.Errorf("failed to load audit rules from filesystem: %w", fsErr)
			}
			return nil
		}

		// Logical AND: both must succeed on live systems (unless command doesn't exist)
		if fsErr != nil && rtErr != nil {
			return fmt.Errorf("failed to load audit rules from both filesystem and runtime: [filesystem: %v, runtime: %v]", fsErr, rtErr)
		}
		if fsErr != nil {
			return fmt.Errorf("failed to load audit rules from filesystem: %w", fsErr)
		}
		if rtErr != nil {
			return fmt.Errorf("failed to load audit rules from runtime: %w", rtErr)
		}
	} else {
		// Non-live system: only filesystem rules (current behavior)
		// This maintains backward compatibility
		if fsErr != nil {
			return fmt.Errorf("failed to load audit rules from filesystem: %w", fsErr)
		}
	}

	// Rules are loaded in separate storage
	return nil
}

// load is a compatibility wrapper that uses the old behavior (filesystem only)
// Deprecated: use loadBySource instead
func (s *mqlAuditdRules) load(path string) error {
	return s.loadFilesystemRules(path)
}

func parseKeyVal(line string) (string, string, int) {
	runes := []rune(line)
	i := 0

	// invalid prefix
	if line[i] != '-' {
		for ; i < len(runes); i++ {
			if unicode.IsSpace(runes[i]) {
				break
			}
		}
		for ; i < len(runes); i++ {
			if !unicode.IsSpace(runes[i]) {
				break
			}
		}
		return "", "", i
	}

	if len(line) < 2 {
		return "", "", len(line)
	}
	if line[1] == '-' {
		i = 2
	} else {
		i = 1
	}

	for ; i < len(runes); i++ {
		if unicode.IsSpace(runes[i]) {
			break
		}
	}
	if i == len(runes) {
		return line, "", i
	}
	keyend := i

	for ; i < len(runes); i++ {
		if !unicode.IsSpace(runes[i]) {
			break
		}
	}
	valstart := i
	for ; i < len(runes); i++ {
		if unicode.IsSpace(runes[i]) {
			break
		}
	}
	valend := i

	for ; i < len(runes); i++ {
		if !unicode.IsSpace(runes[i]) {
			break
		}
	}

	return line[:keyend], line[valstart:valend], i
}

// Make sure this regex matches the most complete form first (ie >=) before
// matching the shorter forms (ie =)
var reOperator = regexp.MustCompile(`(!=|<=|>=|=|>|<)`)

func (s *mqlAuditdRules) parse(content string, errors *multierr.Errors) {
	s.Syscalls.State = plugin.StateIsSet
	s.Files.State = plugin.StateIsSet
	s.Controls.State = plugin.StateIsSet

	lines := strings.Split(content, "\n")
	for _, rawline := range lines {
		line := strings.TrimSpace(rawline)
		if line == "" || line[0] == '#' {
			continue
		}

		resourceName := "auditd.rule.control"
		args := map[string]*llx.RawData{}
		rawFields := []string{}
		syscalls := []any{}
		other := [][2]string{}

		for line != "" {
			k, v, idx := parseKeyVal(line)
			line = line[idx:]

			switch k {
			case "-a":
				resourceName = "auditd.rule.syscall"
				arr := strings.SplitN(v, ",", 2)
				args["action"] = llx.StringData(arr[0])
				args["list"] = llx.StringData(arr[1])

			case "-F":
				rawFields = append(rawFields, v)

			case "-w":
				resourceName = "auditd.rule.file"
				args["path"] = llx.StringData(v)

			case "-k":
				args["keyname"] = llx.StringData(v)

			case "-p":
				args["permissions"] = llx.StringData(v)

			case "-S":
				syscalls = append(syscalls, v)

			default:
				other = append(other, [2]string{k, v})
			}
		}

		switch resourceName {
		case "auditd.rule.file":
			if _, ok := args["keyname"]; !ok {
				args["keyname"] = llx.StringData("")
			}

			r, err := CreateResource(s.MqlRuntime, resourceName, args)
			if err != nil {
				errors.Add(err)
				continue
			}
			s.Files.Data = append(s.Files.Data, r)

		case "auditd.rule.syscall":
			args["syscalls"] = llx.ArrayData(syscalls, types.String)

			fields := make([]any, len(rawFields))
			for i, raw := range rawFields {
				op := reOperator.FindString(raw)
				if op == "" {
					fields[i] = map[string]any{"key": raw}
					continue
				}
				// it must exist according to the preceding statement
				idx := strings.Index(raw, op)
				fields[i] = map[string]any{
					"key":   raw[0:idx],
					"op":    raw[idx : idx+len(op)],
					"value": raw[idx+len(op):],
				}
			}
			args["fields"] = llx.ArrayData(fields, types.Dict)

			if _, ok := args["keyname"]; !ok {
				args["keyname"] = llx.StringData("")
			}

			r, err := CreateResource(s.MqlRuntime, resourceName, args)
			if err != nil {
				errors.Add(err)
				continue
			}
			s.Syscalls.Data = append(s.Syscalls.Data, r)

		default:
			for io := range other {
				r, err := CreateResource(s.MqlRuntime, resourceName, map[string]*llx.RawData{
					"flag":  llx.StringData(other[io][0]),
					"value": llx.StringData(other[io][1]),
				})
				if err != nil {
					errors.Add(err)
					continue
				}
				s.Controls.Data = append(s.Controls.Data, r)
			}
		}
	}
}

func (s *mqlAuditdRules) controls(path string, source string) ([]any, error) {
	if err := s.loadBySource(path, source); err != nil {
		return nil, err
	}

	// Populate the TValue field that the auto-generated code expects
	rules := s.getRulesBySource(source, "controls")
	s.Controls.Data = rules
	s.Controls.State = plugin.StateIsSet

	return rules, nil
}

func (s *mqlAuditdRules) files(path string, source string) ([]any, error) {
	if err := s.loadBySource(path, source); err != nil {
		return nil, err
	}

	// Populate the TValue field that the auto-generated code expects
	rules := s.getRulesBySource(source, "files")
	s.Files.Data = rules
	s.Files.State = plugin.StateIsSet

	return rules, nil
}

func (s *mqlAuditdRules) syscalls(path string, source string) ([]any, error) {
	if err := s.loadBySource(path, source); err != nil {
		return nil, err
	}

	// Populate the TValue field that the auto-generated code expects
	rules := s.getRulesBySource(source, "syscalls")
	s.Syscalls.Data = rules
	s.Syscalls.State = plugin.StateIsSet

	return rules, nil
}

// loadBySource loads rules based on the source parameter
func (s *mqlAuditdRules) loadBySource(path string, source string) error {
	switch source {
	case "filesystem":
		return s.loadFilesystemRules(path)
	case "runtime":
		return s.loadRuntimeRules()
	case "both":
		return s.loadBothSources(path)
	default:
		return fmt.Errorf("invalid source '%s', must be 'filesystem', 'runtime', or 'both'", source)
	}
}

// getRulesBySource returns the appropriate rules based on source parameter
func (s *mqlAuditdRules) getRulesBySource(source string, ruleType string) []any {
	switch source {
	case "filesystem":
		return s.getFilesystemRules(ruleType)
	case "runtime":
		return s.getRuntimeRules(ruleType)
	case "both":
		// For "both", we merge the rules from both sources
		// Set-based comparison: union of both sets
		return s.mergeRules(ruleType)
	default:
		return nil
	}
}

// getFilesystemRules returns filesystem rules of the specified type
func (s *mqlAuditdRules) getFilesystemRules(ruleType string) []any {
	switch ruleType {
	case "controls":
		return s.filesystemData.controls
	case "files":
		return s.filesystemData.files
	case "syscalls":
		return s.filesystemData.syscalls
	default:
		return nil
	}
}

// getRuntimeRules returns runtime rules of the specified type
func (s *mqlAuditdRules) getRuntimeRules(ruleType string) []any {
	switch ruleType {
	case "controls":
		return s.runtimeData.controls
	case "files":
		return s.runtimeData.files
	case "syscalls":
		return s.runtimeData.syscalls
	default:
		return nil
	}
}

// mergeRules merges rules from both sources (union for set-based comparison)
// For strict mode with logical AND, both sources must have successfully loaded
// On non-live systems (no runtime capability), returns only filesystem rules
func (s *mqlAuditdRules) mergeRules(ruleType string) []any {
	fsRules := s.getFilesystemRules(ruleType)
	rtRules := s.getRuntimeRules(ruleType)

	// If no runtime rules (non-live system), return only filesystem rules
	if len(rtRules) == 0 {
		return fsRules
	}

	// If both sources have rules, merge them (union for now)
	// The user can query separately if they want to see differences
	result := make([]any, 0, len(fsRules)+len(rtRules))
	result = append(result, fsRules...)
	result = append(result, rtRules...)

	// TODO: Implement deduplication based on rule IDs
	return result
}

func (s *mqlAuditdRuleFile) id() (string, error) {
	var f checksums.Fast
	return f.
		Add(s.Path.Data).
		Add(s.Permissions.Data).
		Add(s.Keyname.Data).
		String(), nil
}

func (s *mqlAuditdRuleControl) id() (string, error) {
	var f checksums.Fast
	return f.
		Add(s.Flag.Data).
		Add(s.Value.Data).
		String(), nil
}

func (s *mqlAuditdRuleSyscall) id() (string, error) {
	var f checksums.Fast
	f = f.
		Add(s.Action.Data).
		Add(s.List.Data).
		Add(s.Keyname.Data)
	for i := range s.Syscalls.Data {
		f = f.Add(s.Syscalls.Data[i].(string))
	}
	for i := range s.Fields.Data {
		c := s.Fields.Data[i].(map[string]any)
		for k, v := range c {
			f = f.Add(k).Add(v.(string))
		}
	}

	return f.String(), nil
}
