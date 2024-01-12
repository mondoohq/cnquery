// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package resources

import (
	"encoding/json"
	"errors"
	"strings"

	"go.mondoo.com/cnquery/v10/checksums"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/os/resources/parsers"
	"go.mondoo.com/cnquery/v10/providers/os/resources/plist"
	"sigs.k8s.io/yaml"
)

func fileFromPathOrContent(runtime *plugin.Runtime, args map[string]*llx.RawData) error {
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return errors.New("Wrong type for 'path' it must be a string")
		}

		f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path": llx.StringData(path),
		})
		if err != nil {
			return err
		}
		args["file"] = llx.ResourceData(f, "file")
		delete(args, "path")
	} else {
		if x, ok := args["content"]; ok {
			content := x.Value.(string)
			virtualPath := "in-memory://" + checksums.New.Add(content).String()
			f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
				"path":    llx.StringData(virtualPath),
				"content": llx.StringData(content),
				"exists":  llx.BoolTrue,
			})
			if err != nil {
				return err
			}
			args["file"] = llx.ResourceData(f, "file")
		}
	}
	return nil
}

func initParseIni(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if err := fileFromPathOrContent(runtime, args); err != nil {
		return nil, nil, err
	}

	if _, ok := args["delimiter"]; !ok {
		args["delimiter"] = llx.StringData("=")
	}

	return args, nil, nil
}

func (s *mqlParseIni) id() (string, error) {
	if s.File.Data == nil {
		return "", errors.New("no file provided for parse.ini")
	}

	file := s.File.Data
	del := s.Delimiter.Data
	return file.Path.Data + del, nil
}

func (s *mqlParseIni) content(file *mqlFile) (string, error) {
	c := file.GetContent()
	return c.Data, c.Error
}

func (s *mqlParseIni) sections(content string, delimiter string) (map[string]interface{}, error) {
	ini := parsers.ParseIni(content, delimiter)

	res := make(map[string]interface{}, len(ini.Fields))
	for k, v := range ini.Fields {
		res[k] = v
	}

	return res, nil
}

func (s *mqlParseIni) params(sections map[string]interface{}) (map[string]interface{}, error) {
	res := sections[""]
	if res == nil {
		return map[string]interface{}{}, nil
	}
	return res.(map[string]interface{}), nil
}

func initParseJson(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if err := fileFromPathOrContent(runtime, args); err != nil {
		return nil, nil, err
	}

	return args, nil, nil
}

func (s *mqlParseJson) id() (string, error) {
	if s.File.Data == nil {
		return "", errors.New("no file provided for parse.json")
	}

	file := s.File.Data
	return file.Path.Data, nil
}

func (s *mqlParseJson) content(file *mqlFile) (string, error) {
	c := file.GetContent()
	return c.Data, c.Error
}

func (s *mqlParseJson) params(content string) (interface{}, error) {
	if content == "" {
		return nil, nil
	}

	var res interface{}
	if err := json.Unmarshal([]byte(content), &res); err != nil {
		return nil, err
	}
	return res, nil
}

func initParseYaml(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if err := fileFromPathOrContent(runtime, args); err != nil {
		return nil, nil, err
	}

	return args, nil, nil
}

func (s *mqlParseYaml) id() (string, error) {
	if s.File.Data == nil {
		return "", errors.New("no file provided for parse.yaml")
	}

	file := s.File.Data
	return file.Path.Data, nil
}

func (s *mqlParseYaml) content(file *mqlFile) (string, error) {
	c := file.GetContent()
	return c.Data, c.Error
}

func (s *mqlParseYaml) params(content string) (map[string]interface{}, error) {
	res := make(map[string](interface{}))

	if content == "" {
		return nil, nil
	}

	err := yaml.Unmarshal([]byte(content), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func initParsePlist(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if err := fileFromPathOrContent(runtime, args); err != nil {
		return nil, nil, err
	}
	return args, nil, nil
}

func (s *mqlParsePlist) id() (string, error) {
	if s.File.Data == nil {
		return "", errors.New("no file provided for parse.plist")
	}

	file := s.File.Data
	return file.Path.Data, nil
}

func (s *mqlParsePlist) content(file *mqlFile) (string, error) {
	c := file.GetContent()
	return c.Data, c.Error
}

func (s *mqlParsePlist) params(content string) (map[string]interface{}, error) {
	return plist.Decode(strings.NewReader(content))
}

func initParseCertificates(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// resolve path to file
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in certificates initialization, it must be a string")
		}

		f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path": llx.StringData(path),
		})
		if err != nil {
			return nil, nil, err
		}
		args["file"] = llx.ResourceData(f, "file")

	} else if x, ok := args["content"]; ok {
		content := x.Value.(string)
		virtualPath := "in-memory://" + checksums.New.Add(content).String()
		f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path":    llx.StringData(virtualPath),
			"content": llx.StringData(content),
			"exists":  llx.BoolTrue,
		})
		if err != nil {
			return nil, nil, err
		}
		args["file"] = llx.ResourceData(f, "file")
		args["path"] = llx.StringData(virtualPath)
	} else {
		return nil, nil, errors.New("missing 'path' or 'content' for parse.json initialization")
	}

	return args, nil, nil
}

func certificatesid(path string) string {
	return "certificates:" + path
}

func (a *mqlParseCertificates) id() (string, error) {
	f := a.File.Data
	if f == nil {
		return "", errors.New("missing file in parse certificate")
	}

	return certificatesid(f.Path.Data), nil
}

func (a *mqlParseCertificates) file() (*mqlFile, error) {
	f, err := CreateResource(a.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(a.Path.Data),
	})
	if err != nil {
		return nil, err
	}
	return f.(*mqlFile), nil
}

func (a *mqlParseCertificates) content(file *mqlFile) (string, error) {
	res := file.GetContent()
	return res.Data, res.Error
}

func (p *mqlParseCertificates) list(content string, path string) ([]interface{}, error) {
	certificates, err := p.MqlRuntime.CreateSharedResource("certificates", map[string]*llx.RawData{
		"pem": llx.StringData(content),
	})
	if err != nil {
		return nil, err
	}

	list, err := p.MqlRuntime.GetSharedData("certificates", certificates.MqlID(), "list")
	if err != nil {
		return nil, err
	}

	return list.Value.([]interface{}), nil
}

func initParseOpenpgp(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// resolve path to file
	if x, ok := args["path"]; ok {
		f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path": x,
		})
		if err != nil {
			return nil, nil, err
		}
		args["file"] = llx.ResourceData(f, "file")

	} else if x, ok := args["content"]; ok {
		content := x.Value.(string)
		virtualPath := "in-memory://" + checksums.New.Add(content).String()
		f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path":    llx.StringData(virtualPath),
			"content": llx.StringData(content),
			"exists":  llx.BoolTrue,
		})
		if err != nil {
			return nil, nil, err
		}
		args["file"] = llx.ResourceData(f, "file")
		args["path"] = llx.StringData(virtualPath)

	} else {
		return nil, nil, errors.New("missing 'path' or 'content' for parse.json initialization")
	}

	return args, nil, nil
}

func (a *mqlParseOpenpgp) id() (string, error) {
	if a.File.Error != nil {
		return "", a.File.Error
	}

	return a.File.Data.Path.Data, nil
}

func (a *mqlParseOpenpgp) content(file plugin.Resource) (string, error) {
	res := file.(*mqlFile).GetContent()
	return res.Data, res.Error
}

func (p *mqlParseOpenpgp) list(content string) ([]interface{}, error) {
	certificates, err := p.MqlRuntime.CreateSharedResource("openpgp.entities", map[string]*llx.RawData{
		"content": llx.StringData(content),
	})
	if err != nil {
		return nil, err
	}

	list, err := p.MqlRuntime.GetSharedData("openpgp.entities", certificates.MqlID(), "list")
	if err != nil {
		return nil, err
	}

	return list.Value.([]interface{}), nil
}
