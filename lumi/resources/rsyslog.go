package resources

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/fasthash/fnv1a"
	"go.mondoo.io/mondoo/lumi"
)

func (s *lumiRsyslogConf) init(args *lumi.Args) (*lumi.Args, RsyslogConf, error) {
	if x, ok := (*args)["path"]; ok {
		path, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in initialization, it must be a string")
		}

		files, err := s.getFiles(path)
		if err != nil {
			return nil, nil, err
		}

		(*args)["files"] = files
		delete(*args, "path")
	}

	return args, nil, nil
}

const defaultRsyslogConf = "/etc/rsyslog.conf"

func (s *lumiRsyslogConf) id() (string, error) {
	files, err := s.Files()
	if err != nil {
		return "", err
	}

	checksum := fnv1a.Init64
	for i := range files {
		c, err := files[i].(File).Path()
		if err != nil {
			return "", err
		}
		checksum = fnv1a.AddString64(checksum, c)
	}

	return checksum2string(checksum), nil
}

func (s *lumiRsyslogConf) getFiles(confPath string) ([]interface{}, error) {
	if !strings.HasSuffix(confPath, ".conf") {
		return nil, errors.New("failed to initialize, path must end in `.conf` so we can find files in `.d` directory")
	}

	f, err := s.Runtime.CreateResource("file", "path", confPath)
	if err != nil {
		return nil, err
	}

	confD := confPath[0:len(confPath)-5] + ".d"
	files, err := s.Runtime.CreateResource("files.find", "from", confD, "type", "file")
	if err != nil {
		return nil, err
	}

	list, err := files.(FilesFind).List()
	if err != nil {
		return nil, err
	}

	list = append([]interface{}{f.(File)}, list...)
	return list, nil
}

func (s *lumiRsyslogConf) GetFiles() ([]interface{}, error) {
	return s.getFiles(defaultRsyslogConf)
}

func (s *lumiRsyslogConf) GetContent(files []interface{}) (string, error) {
	var res strings.Builder
	var notReadyError error = nil

	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes
	for i := range files {
		file := files[i].(File)

		err := s.Runtime.WatchAndCompute(file, "content", s, "content")
		if err != nil {
			log.Error().Err(err).Msg("[rsyslog.conf]> watch+compute failed for file.content")
		}

		content, err := file.Content()
		if err != nil {
			notReadyError = lumi.NotReadyError{}
		}

		res.WriteString(content)
		res.WriteString("\n")
	}

	if notReadyError != nil {
		return "", notReadyError
	}

	return res.String(), nil
}

func (s *lumiRsyslogConf) GetSettings(content string) ([]interface{}, error) {
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
