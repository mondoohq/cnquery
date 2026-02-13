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

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/resources/parsers"
	"go.mondoo.com/mql/v13/types"
	"go.mondoo.com/mql/v13/utils/multierr"
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

// parses the journald config file and creates the resources
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

	filePath := file.GetPath()
	if filePath.Error != nil {
		return filePath.Error
	}

	var errs multierr.Errors
	sectionResources := []any{}

	for i, unitSection := range unit.Sections {
		sectionID := fmt.Sprintf("%s/%s/%d", filePath.Data, unitSection.Name, i)
		paramResources := []any{}

		for j, unitParam := range unitSection.Params {
			val := unitParam.Value
			if slices.Contains(journaldDowncaseKeywords, unitParam.Name) {
				val = strings.ToLower(val)
			}

			paramID := fmt.Sprintf("%s/%s/%d", sectionID, unitParam.Name, j)
			param, err := CreateResource(s.MqlRuntime, ResourceJournaldConfigSectionParam, map[string]*llx.RawData{
				"__id":  llx.StringData(paramID),
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
			"__id":   llx.StringData(sectionID),
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

// returns the sections of the journald config, eg [Journal], [Upload], etc
func (s *mqlJournaldConfig) sections(file *mqlFile) ([]any, error) {
	if err := s.parse(file); err != nil {
		return nil, err
	}
	return s.Sections.Data, s.Sections.Error
}

// params is deprecated, use sections instead
func (s *mqlJournaldConfig) params(file *mqlFile) (map[string]any, error) {
	if err := s.parse(file); err != nil {
		return nil, err
	}

	// For backward compatibility, return the [Journal] section's params as a map
	for _, sectionAny := range s.Sections.Data {
		section := sectionAny.(*mqlJournaldConfigSection)
		name := section.GetName()
		if name.Error != nil {
			continue
		}

		if name.Data != "Journal" {
			continue
		}

		params := section.GetParams()
		if params.Error != nil {
			return nil, params.Error
		}

		result := make(map[string]any, len(params.Data))
		for _, paramAny := range params.Data {
			param := paramAny.(*mqlJournaldConfigSectionParam)
			paramName := param.GetName()
			paramValue := param.GetValue()
			if paramName.Error != nil || paramValue.Error != nil {
				continue
			}
			result[paramName.Data] = paramValue.Data
		}
		return result, nil
	}

	return map[string]any{}, nil
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
