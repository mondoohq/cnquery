// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann
// author: Tim Smith

package resources

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/os/resources/parsers"
	"go.mondoo.com/cnquery/v12/types"
	"go.mondoo.com/cnquery/v12/utils/multierr"
)

type mqlJournaldConfigInternal struct {
	lock sync.Mutex
}

func initJournaldConfig(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' in journald.config initialization, it must be a string")
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

const defaultJournaldConfig = "/etc/systemd/journald.conf"

func (s *mqlJournaldConfig) id() (string, error) {
	file := s.GetFile()
	if file.Error != nil {
		return "", file.Error
	}

	return file.Data.Path.Data, nil
}

func (s *mqlJournaldConfig) file() (*mqlFile, error) {
	f, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(defaultJournaldConfig),
	})
	if err != nil {
		return nil, err
	}
	return f.(*mqlFile), nil
}

func (s *mqlJournaldConfig) parse(file *mqlFile) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if file == nil {
		return errors.New("no base journald config file to read")
	}

	content := file.GetContent()
	if content.Error != nil {
		return content.Error
	}

	ini := parsers.ParseIni(content.Data, "=")

	if len(ini.Fields) == 0 {
		s.Sections.Data = []any{}
		s.Sections.State = plugin.StateIsSet
		return nil
	}

	var errs multierr.Errors
	sectionResources := []any{}

	for sectionName, sectionData := range ini.Fields {
		fields, ok := sectionData.(map[string]any)
		if !ok {
			errs.Add(fmt.Errorf("failed to parse section '%s' (invalid data)", sectionName))
			continue
		}

		paramResources := []any{}
		for k, v := range fields {
			if val, ok := v.(string); ok {
				if slices.Contains(journaldDowncaseKeywords, k) {
					val = strings.ToLower(val)
				}
				// journald.config.section.param
				param, err := CreateResource(s.MqlRuntime, ResourceJournaldConfigSectionParam, map[string]*llx.RawData{
					"name":  llx.StringData(k),
					"value": llx.StringData(val),
				})
				if err != nil {
					errs.Add(fmt.Errorf("failed to create param resource for '%s' in section '%s': %w", k, sectionName, err))
					continue
				}
				paramResources = append(paramResources, param)
			} else {
				errs.Add(fmt.Errorf("can't parse field '%s' in section '%s', value is %+v", k, sectionName, v))
			}
		}

		section, err := CreateResource(s.MqlRuntime, ResourceJournaldConfigSection, map[string]*llx.RawData{
			"name":   llx.StringData(sectionName),
			"params": llx.ArrayData(paramResources, types.Resource(ResourceJournaldConfigSectionParam)),
		})
		if err != nil {
			errs.Add(fmt.Errorf("failed to create section resource for '%s': %w", sectionName, err))
			continue
		}

		sectionResources = append(sectionResources, section)
	}

	s.Sections.Data = sectionResources
	s.Sections.State = plugin.StateIsSet
	s.Sections.Error = errs.Deduplicate()
	return s.Sections.Error
}

func (s *mqlJournaldConfig) sections(file *mqlFile) ([]any, error) {
	if err := s.parse(file); err != nil {
		return nil, err
	}
	return s.Sections.Data, s.Sections.Error
}

func (s *mqlJournaldConfigSection) id() (string, error) {
	name := s.GetName()
	if name.Error != nil {
		return "", name.Error
	}
	return ResourceJournaldConfigSection + ":" + name.Data, nil
}

func (s *mqlJournaldConfigSectionParam) id() (string, error) {
	name := s.GetName()
	if name.Error != nil {
		return "", name.Error
	}
	value := s.GetValue()
	if value.Error != nil {
		return "", value.Error
	}
	return ResourceJournaldConfigSectionParam + ":" + name.Data + "=" + value.Data, nil
}

// These are the boolean options in journald.conf which are case insensitive
// See https://www.man7.org/linux/man-pages/man5/journald.conf.5.html
var journaldDowncaseKeywords = []string{
	"Compress",
	"Seal",
	"ForwardToSyslog",
	"ForwardToKMsg",
	"ForwardToConsole",
	"ForwardToWall",
	"ForwardToSocket",
	"ReadKMsg",
	"Audit",
}
