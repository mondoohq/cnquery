package resources

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/checksums"
	"go.mondoo.io/mondoo/lumi"
)

func (s *lumiPamConf) init(args *lumi.Args) (*lumi.Args, PamConf, error) {
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

const defaultPamConf = "/etc/pam.conf"

func (s *lumiPamConf) id() (string, error) {
	files, err := s.Files()
	if err != nil {
		return "", err
	}

	checksum := checksums.New
	for i := range files {
		c, err := files[i].(File).Path()
		if err != nil {
			return "", err
		}
		checksum = checksum.Add(c)
	}

	return checksum.String(), nil
}

func (s *lumiPamConf) getFiles(confPath string) ([]interface{}, error) {
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

func (s *lumiPamConf) GetFiles() ([]interface{}, error) {
	return s.getFiles(defaultPamConf)
}

func (s *lumiPamConf) GetContent(files []interface{}) (string, error) {
	var res strings.Builder
	var notReadyError error = nil

	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes
	for i := range files {
		file := files[i].(File)

		err := s.Runtime.WatchAndCompute(file, "content", s, "content")
		if err != nil {
			log.Error().Err(err).Msg("[pam.conf]> watch+compute failed for file.content")
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

func (s *lumiPamConf) GetServices(files []interface{}) (map[string]interface{}, error) {
	contents := map[string]string{}
	var notReadyError error = nil

	for i := range files {
		file := files[i].(File)

		err := s.Runtime.WatchAndCompute(file, "content", s, "services")
		if err != nil {
			log.Error().Err(err).Msg("[pam.conf]> watch+compute failed for file.content")
			return nil, err
		}
		err = s.Runtime.WatchAndCompute(file, "basename", s, "services")
		if err != nil {
			log.Error().Err(err).Msg("[pam.conf]> watch+compute failed for file.basename")
			return nil, err
		}

		content, err := file.Content()
		if err != nil {
			notReadyError = lumi.NotReadyError{}
		}

		basename, err := file.Basename()
		if err != nil {
			notReadyError = lumi.NotReadyError{}
		}

		contents[basename] = content
	}

	if notReadyError != nil {
		return nil, notReadyError
	}

	services := map[string]interface{}{}
	for basename, content := range contents {
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

		services[basename] = settings
	}

	return services, nil
}
