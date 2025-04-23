// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package resources

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/os/resources/parsers"
	"go.mondoo.com/cnquery/v11/utils/multierr"
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
