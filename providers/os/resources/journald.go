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

	unit, err := parsers.ParseUnit(content.Data)
	if err != nil {
		return fmt.Errorf("failed to parse journald config: %w", err)
	}

	if len(unit.Sections) == 0 {
		s.Sections.Data = []any{}
		s.Sections.State = plugin.StateIsSet
		return nil
	}

	var errs multierr.Errors
	sectionResources := []any{}

	for _, unitSection := range unit.Sections {
		paramResources := []any{}
		for _, unitParam := range unitSection.Params {
			val := unitParam.Value
			// Apply downcase logic for boolean keywords
			if slices.Contains(journaldDowncaseKeywords, unitParam.Name) {
				val = strings.ToLower(val)
			}

			param, err := CreateResource(s.MqlRuntime, ResourceJournaldConfigSectionParam, map[string]*llx.RawData{
				"name":  llx.StringData(unitParam.Name),
				"value": llx.StringData(val),
			})
			if err != nil {
				errs.Add(fmt.Errorf("failed to create param resource for '%s' in section '%s': %w", unitParam.Name, unitSection.Name, err))
				continue
			}
			paramResources = append(paramResources, param)
		}

		section, err := CreateResource(s.MqlRuntime, ResourceJournaldConfigSection, map[string]*llx.RawData{
			"name":   llx.StringData(unitSection.Name),
			"params": llx.ArrayData(paramResources, types.Resource(ResourceJournaldConfigSectionParam)),
		})
		if err != nil {
			errs.Add(fmt.Errorf("failed to create section resource for '%s': %w", unitSection.Name, err))
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
