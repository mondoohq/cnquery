// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package resources

import (
	"errors"
	"regexp"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
)

func (s *lumiSshdConfig) init(args *lumi.Args) (*lumi.Args, SshdConfig, error) {
	if x, ok := (*args)["path"]; ok {
		path, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in sshd.config initialization, it must be a string")
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

const defaultSshdConfig = "/etc/ssh/sshd_config"

func (s *lumiSshdConfig) id() (string, error) {
	r, err := s.File()
	if err != nil {
		return "", err
	}
	return r.Path()
}

func (s *lumiSshdConfig) GetFile() (File, error) {
	f, err := s.Runtime.CreateResource("file", "path", defaultSshdConfig)
	if err != nil {
		return nil, err
	}
	return f.(File), nil
}

func (s *lumiSshdConfig) GetContent(file File) (string, error) {
	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes
	err := s.Runtime.WatchAndCompute(file, "content", s, "content")
	if err != nil {
		log.Error().Err(err).Msg("sshd.config> watch+compute failed")
	}

	return file.Content()
}

func (s *lumiSshdConfig) GetParams(content string) (map[string]interface{}, error) {
	re := regexp.MustCompile("(?m:^([[:alpha:]]+)\\s+(.*))")
	m := re.FindAllStringSubmatch(content, -1)
	res := make(map[string]interface{})
	for _, mm := range m {
		res[mm[1]] = mm[2]
	}

	return res, nil
}
