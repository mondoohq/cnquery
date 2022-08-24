// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package core

import (
	"encoding/json"
	"errors"
	"strings"

	"go.mondoo.com/cnquery/checksums"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core/parsers"
	"go.mondoo.com/cnquery/resources/packs/core/plist"
	"sigs.k8s.io/yaml"
)

func (s *mqlParse) id() (string, error) {
	return "", nil
}

func (s *mqlParseIni) init(args *resources.Args) (*resources.Args, ParseIni, error) {
	if x, ok := (*args)["path"]; ok {
		path, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in parse.ini initialization, it must be a string")
		}

		f, err := s.MotorRuntime.CreateResource("file", "path", path)
		if err != nil {
			return nil, nil, err
		}
		(*args)["file"] = f
		delete(*args, "path")
	}

	return args, nil, nil
}

func (s *mqlParseIni) id() (string, error) {
	r, err := s.File()
	if err != nil {
		return "", err
	}

	path, err := r.Path()
	if err != nil {
		return "", err
	}

	del, err := s.Delimiter()
	if err != nil {
		return "", err
	}

	return path + del, nil
}

func (s *mqlParseIni) GetDelimiter() (string, error) {
	return "=", nil
}

func (s *mqlParseIni) GetFile() (File, error) {
	// TODO: all of this is a placeholder, in case we initialize the ini resource with content instead of a file
	f, err := s.MotorRuntime.CreateResource("file", "path", "/")
	if err != nil {
		return nil, err
	}
	return f.(File), nil
}

func (s *mqlParseIni) GetContent(file File) (string, error) {
	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes
	err := s.MotorRuntime.WatchAndCompute(file, "content", s, "content")
	if err != nil {
		return "", err
	}

	return file.Content()
}

func (s *mqlParseIni) GetSections(content string, delimiter string) (map[string]interface{}, error) {
	ini := parsers.ParseIni(content, delimiter)

	res := make(map[string]interface{}, len(ini.Fields))
	for k, v := range ini.Fields {
		res[k] = v
	}

	return res, nil
}

func (s *mqlParseIni) GetParams(sections map[string]interface{}) (map[string]interface{}, error) {
	res := sections[""]
	if res == nil {
		return map[string]interface{}{}, nil
	}
	return res.(map[string]interface{}), nil
}

func (s *mqlParseJson) init(args *resources.Args) (*resources.Args, ParseJson, error) {
	rawPath := (*args)["path"]

	if rawPath != nil {
		path := rawPath.(string)

		f, err := s.MotorRuntime.CreateResource("file", "path", path)
		if err != nil {
			return nil, nil, err
		}
		(*args)["file"] = f
		delete(*args, "path")

	} else if x, ok := (*args)["content"]; ok {
		content := x.(string)
		virtualPath := "in-memory://" + checksums.New.Add(content).String()
		f, err := s.MotorRuntime.CreateResource("file", "path", virtualPath, "content", content, "exists", true)
		if err != nil {
			return nil, nil, err
		}
		(*args)["file"] = f

	} else {
		return nil, nil, errors.New("missing 'path' or 'content' for parse.json initialization")
	}

	return args, nil, nil
}

func (s *mqlParseJson) id() (string, error) {
	r, err := s.File()
	if err != nil {
		return "", err
	}

	path, err := r.Path()
	if err != nil {
		return "", err
	}

	return path, nil
}

func (s *mqlParseJson) GetFile() (File, error) {
	// TODO: all of this is a placeholder, in case we initialize the ini resource with content instead of a file
	f, err := s.MotorRuntime.CreateResource("file", "path", "/")
	if err != nil {
		return nil, err
	}
	return f.(File), nil
}

func (s *mqlParseJson) GetContent(file File) (string, error) {
	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes
	err := s.MotorRuntime.WatchAndCompute(file, "content", s, "content")
	if err != nil {
		return "", err
	}

	return file.Content()
}

func (s *mqlParseJson) GetParams(content string) (interface{}, error) {
	if content == "" {
		return nil, nil
	}

	var res interface{}
	if err := json.Unmarshal([]byte(content), &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (s *mqlParseYaml) init(args *resources.Args) (*resources.Args, ParseJson, error) {
	rawPath := (*args)["path"]

	if rawPath != nil {
		path := rawPath.(string)

		f, err := s.MotorRuntime.CreateResource("file", "path", path)
		if err != nil {
			return nil, nil, err
		}
		(*args)["file"] = f
		delete(*args, "path")

	} else if x, ok := (*args)["content"]; ok {
		content := x.(string)
		virtualPath := "in-memory://" + checksums.New.Add(content).String()
		f, err := s.MotorRuntime.CreateResource("file", "path", virtualPath, "content", content, "exists", true)
		if err != nil {
			return nil, nil, err
		}
		(*args)["file"] = f
	}

	return args, nil, nil
}

func (s *mqlParseYaml) id() (string, error) {
	r, err := s.File()
	if err != nil {
		return "", err
	}

	path, err := r.Path()
	if err != nil {
		return "", err
	}

	return path, nil
}

func (s *mqlParseYaml) GetFile() (File, error) {
	// NOTE: this code should never be reached since the file field is initialized via init
	return nil, errors.New("no file available")
}

func (s *mqlParseYaml) GetContent(file File) (string, error) {
	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes
	err := s.MotorRuntime.WatchAndCompute(file, "content", s, "content")
	if err != nil {
		return "", err
	}

	return file.Content()
}

func (s *mqlParseYaml) GetParams(content string) (map[string]interface{}, error) {
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

func (s *mqlParsePlist) init(args *resources.Args) (*resources.Args, ParseJson, error) {
	if x, ok := (*args)["path"]; ok {
		path, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' in parse.json initialization, it must be a string")
		}

		f, err := s.MotorRuntime.CreateResource("file", "path", path)
		if err != nil {
			return nil, nil, err
		}
		(*args)["file"] = f
		delete(*args, "path")
	} else if z, ok := (*args)["content"]; ok {
		content := z.(string)
		virtualPath := "in-memory://" + checksums.New.Add(content).String()
		f, err := s.MotorRuntime.CreateResource("file", "path", virtualPath, "content", content, "exists", true)
		if err != nil {
			return nil, nil, err
		}
		(*args)["file"] = f
	} else {
		return nil, nil, errors.New("missing 'path' argument for parse.json initialization")
	}

	return args, nil, nil
}

func (s *mqlParsePlist) id() (string, error) {
	r, err := s.File()
	if err != nil {
		return "", err
	}

	path, err := r.Path()
	if err != nil {
		return "", err
	}

	return path, nil
}

func (s *mqlParsePlist) GetFile() (File, error) {
	// TODO: all of this is a placeholder, in case we initialize the plist resource with content instead of a file
	f, err := s.MotorRuntime.CreateResource("file", "path", "/")
	if err != nil {
		return nil, err
	}
	return f.(File), nil
}

func (s *mqlParsePlist) GetContent(file File) (string, error) {
	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes
	err := s.MotorRuntime.WatchAndCompute(file, "content", s, "content")
	if err != nil {
		return "", err
	}

	return file.Content()
}

func (s *mqlParsePlist) GetParams(content string) (map[string]interface{}, error) {
	return plist.Decode(strings.NewReader(content))
}
