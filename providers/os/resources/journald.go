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

	res := make(map[string]any, len(ini.Fields))
	s.Params.Data = res
	s.Params.State = plugin.StateIsSet

	if len(ini.Fields) == 0 {
		return nil
	}

	root := ini.Fields["Journal"]
	if root == nil {
		s.Params.Error = errors.New("failed to parse journald config")
		return s.Params.Error
	}

	fields, ok := root.(map[string]any)
	if !ok {
		s.Params.Error = errors.New("failed to parse journald config (invalid data retrieved)")
		return s.Params.Error
	}

	var errs multierr.Errors
	for k, v := range fields {
		if s, ok := v.(string); ok {
			if slices.Contains(journaldDowncaseKeywords, k) {
				res[k] = strings.ToLower(s)
			} else {
				res[k] = s
			}
		} else {
			errs.Add(fmt.Errorf("can't parse field '"+s+"', value is %+v", v))
		}
	}

	s.Params.Error = errs.Deduplicate()
	return s.Params.Error
}

func (s *mqlJournaldConfig) params(file *mqlFile) (map[string]any, error) {
	return nil, s.parse(file)
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
