// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package resources

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/parsers"
)

func (s *lumiParseIni) init(args *lumi.Args) (*lumi.Args, ParseIni, error) {
	if x, ok := (*args)["path"]; ok {
		path, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in parse.ini initialization, it must be a string")
		}

		f, err := s.Runtime.CreateResource("file", "path", path)
		if err != nil {
			return nil, nil, err
		}
		(*args)["file"] = f
		delete(*args, "path")
	}

	return args, nil, nil
}

func (s *lumiParseIni) id() (string, error) {
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

func (s *lumiParseIni) GetDelimiter() (string, error) {
	return "=", nil
}

func (s *lumiParseIni) GetFile() (File, error) {
	// TODO: all of this is a placeholder, in case we initialize the ini resource with content instead of a file
	f, err := s.Runtime.CreateResource("file", "path", "/")
	if err != nil {
		return nil, err
	}
	return f.(File), nil
}

func (s *lumiParseIni) GetContent(file File) (string, error) {
	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes
	err := s.Runtime.WatchAndCompute(file, "content", s, "content")
	if err != nil {
		log.Error().Err(err).Msg("parse.ini> watch+compute failed")
		return "", err
	}

	return file.Content()
}

func (s *lumiParseIni) GetSections(content string, delimiter string) (map[string]interface{}, error) {
	ini := parsers.ParseIni(content, delimiter)

	res := make(map[string]interface{}, len(ini.Fields))
	for k, v := range ini.Fields {
		res[k] = v
	}

	return res, nil
}

func (s *lumiParseIni) GetParams(sections map[string]interface{}) (map[string]interface{}, error) {
	res := sections[""]
	if res == nil {
		return map[string]interface{}{}, nil
	}
	return res.(map[string]interface{}), nil
}
