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
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
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

// mqlAuditdRulesInternal holds minimal internal state for the resource
type mqlAuditdRulesInternal struct{}

const defaultAuditdRules = "/etc/audit/rules.d"

func initAuditdRules(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// v3.0: No source parameter to handle - connection determines behavior
	return args, nil, nil
}

func (s *mqlAuditdRules) id() (string, error) {
	// Simple path-based ID (source is connection-level implementation detail)
	return s.Path.Data, nil
}

func (s *mqlAuditdRules) path() (string, error) {
	return defaultAuditdRules, nil
}

// controls returns all control rules via connection provider
func (s *mqlAuditdRules) controls(path string) ([]any, error) {
	data, err := s.getAuditRuleData(path)
	if err != nil {
		return nil, err
	}

	s.Controls.Data = data.Controls
	s.Controls.State = plugin.StateIsSet
	return data.Controls, nil
}

// files returns all file rules via connection provider
func (s *mqlAuditdRules) files(path string) ([]any, error) {
	data, err := s.getAuditRuleData(path)
	if err != nil {
		return nil, err
	}

	s.Files.Data = data.Files
	s.Files.State = plugin.StateIsSet
	return data.Files, nil
}

// syscalls returns all syscall rules via connection provider
func (s *mqlAuditdRules) syscalls(path string) ([]any, error) {
	data, err := s.getAuditRuleData(path)
	if err != nil {
		return nil, err
	}

	s.Syscalls.Data = data.Syscalls
	s.Syscalls.State = plugin.StateIsSet
	return data.Syscalls, nil
}

// getAuditRuleData delegates to connection's audit rule provider
func (s *mqlAuditdRules) getAuditRuleData(path string) (*shared.AuditRuleData, error) {
	conn, ok := s.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		return nil, fmt.Errorf("connection does not implement shared.Connection interface")
	}

	provider := conn.AuditRuleProvider()
	if provider == nil {
		return nil, fmt.Errorf("connection does not provide audit rule provider")
	}

	// Inject parser function into provider if not already set
	provider.SetParser(s.parseAuditRules)

	// Get rules from provider (handles dual-source logic internally)
	return provider.GetRules(path)
}

// parseAuditRules is the parser function injected into the connection provider
// It parses audit rule content and returns structured data
func (s *mqlAuditdRules) parseAuditRules(content string) (*shared.AuditRuleData, error) {
	// Initialize result data
	data := &shared.AuditRuleData{
		Controls: make([]interface{}, 0),
		Files:    make([]interface{}, 0),
		Syscalls: make([]interface{}, 0),
	}

	// Use existing parse logic
	var errors multierr.Errors
	s.parseInto(content, &data.Controls, &data.Files, &data.Syscalls, &errors)

	if len(errors.Errors) > 0 {
		return nil, fmt.Errorf("failed to parse audit rules: %w", errors.Deduplicate())
	}

	return data, nil
}

// parseInto parses audit rule content into provided slices
func (s *mqlAuditdRules) parseInto(content string, controls, files, syscalls *[]interface{}, errors *multierr.Errors) {
	lines := strings.Split(content, "\n")
	for _, rawline := range lines {
		line := strings.TrimSpace(rawline)
		if line == "" || line[0] == '#' {
			continue
		}

		resourceName := "auditd.rule.control"
		args := map[string]*llx.RawData{}
		rawFields := []string{}
		syscallList := []any{}
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
				syscallList = append(syscallList, v)

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
			*files = append(*files, r)

		case "auditd.rule.syscall":
			args["syscalls"] = llx.ArrayData(syscallList, types.String)

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
			*syscalls = append(*syscalls, r)

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
				*controls = append(*controls, r)
			}
		}
	}
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
