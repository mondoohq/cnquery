package resources

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
)

func (s *lumiNtpConf) init(args *lumi.Args) (*lumi.Args, NtpConf, error) {
	if x, ok := (*args)["path"]; ok {
		path, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in ntp.conf initialization, it must be a string")
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

const defaultNtpConf = "/etc/ntp.conf"

func (s *lumiNtpConf) id() (string, error) {
	r, err := s.File()
	if err != nil {
		return "", err
	}
	return r.Path()
}

func (s *lumiNtpConf) GetFile() (File, error) {
	f, err := s.Runtime.CreateResource("file", "path", defaultNtpConf)
	if err != nil {
		return nil, err
	}
	return f.(File), nil
}

func (s *lumiNtpConf) GetContent(file File) (string, error) {
	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes
	err := s.Runtime.WatchAndCompute(file, "content", s, "content")
	if err != nil {
		log.Error().Err(err).Msg("ntp.conf> watch+compute failed")
		return "", err
	}

	return file.Content()
}

func (s *lumiNtpConf) GetSettings(content string) ([]interface{}, error) {
	lines := strings.Split(content, "\n")

	settings := []interface{}{}
	var line string
	for i := range lines {
		line = lines[i]
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[0:idx]
		}
		line = strings.Trim(line, " \t\r")

		if line != "" {
			settings = append(settings, line)
		}
	}

	return settings, nil
}

func (s *lumiNtpConf) GetServers(settings []interface{}) ([]interface{}, error) {
	res := []interface{}{}
	var line string
	for i := range settings {
		line = settings[i].(string)
		if strings.HasPrefix(line, "server ") {
			res = append(res, line[7:])
		}
	}

	return res, nil
}

func (s *lumiNtpConf) GetRestrict(settings []interface{}) ([]interface{}, error) {
	res := []interface{}{}
	var line string
	for i := range settings {
		line = settings[i].(string)
		if strings.HasPrefix(line, "restrict ") {
			res = append(res, line[9:])
		}
	}

	return res, nil
}

func (s *lumiNtpConf) GetFudge(settings []interface{}) ([]interface{}, error) {
	res := []interface{}{}
	var line string
	for i := range settings {
		line = settings[i].(string)
		if strings.HasPrefix(line, "fudge ") {
			res = append(res, line[6:])
		}
	}

	return res, nil
}
