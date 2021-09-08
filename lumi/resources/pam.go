package resources

import (
	"errors"
	"io/ioutil"
	"strconv"
	"strings"

	"go.mondoo.io/mondoo/checksums"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/pam"
)

const (
	defaultPamConf = "/etc/pam.conf"
	defaultPamDir  = "/etc/pam.d"
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

func (se *lumiPamConfServiceEntry) id() (string, error) {
	ptype, err := se.PamType()
	if err != nil {
		return "", err
	}
	mod, err := se.Module()
	if err != nil {
		return "", err
	}

	s, err := se.Service()
	if err != nil {
		return "", err
	}
	ln, err := se.LineNumber()
	if err != nil {
		return "", err
	}
	lnstr := strconv.FormatInt(ln, 10)
	id := s + ptype + mod + lnstr
	return id, nil
}

func (s *lumiPamConf) getFiles(confPath string) ([]interface{}, error) {
	// check if the pam.d directory or pam config file exists
	lumiFile, err := s.Runtime.CreateResource("file", "path", confPath)
	if err != nil {
		return nil, err
	}
	f := lumiFile.(File)
	exists, err := f.Exists()
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(" could not load pam configuration: " + confPath)
	}

	fp, err := f.Permissions()
	if err != nil {
		return nil, err
	}
	isDir, err := fp.IsDirectory()
	if err != nil {
		return nil, err
	}
	if isDir {
		return s.getConfDFiles(confPath)
	} else {
		return []interface{}{f.(File)}, nil
	}
}

func (s *lumiPamConf) getConfDFiles(confD string) ([]interface{}, error) {
	files, err := s.Runtime.CreateResource("files.find", "from", confD, "type", "file")
	if err != nil {
		return nil, err
	}

	list, err := files.(FilesFind).List()
	if err != nil {
		return nil, err
	}
	return list, nil
}

// GetFiles is called when the user has not provided a custom path. Otherwise files are set in the init
// method and this function is never called then since the data is already cached.
func (s *lumiPamConf) GetFiles() ([]interface{}, error) {
	// check if the pam.d directory exists and is a directory
	// according to the pam spec, pam prefers the directory if it  exists over the single file config
	// see http://www.linux-pam.org/Linux-PAM-html/sag-configuration.html
	lumiFile, err := s.Runtime.CreateResource("file", "path", defaultPamDir)
	if err != nil {
		return nil, err
	}
	f := lumiFile.(File)
	exists, err := f.Exists()
	if err != nil {
		return nil, err
	}

	if exists {
		return s.getFiles(defaultPamDir)
	} else {
		return s.getFiles(defaultPamConf)
	}
}

func (s *lumiPamConf) GetContent(files []interface{}) (string, error) {
	var res strings.Builder
	var notReadyError error = nil

	for i := range files {
		file := files[i].(File)

		path, err := file.Path()
		if err != nil {
			return "", err
		}
		f, err := s.Runtime.Motor.Transport.FS().Open(path)
		if err != nil {
			return "", err
		}

		raw, err := ioutil.ReadAll(f)
		f.Close()
		if err != nil {
			return "", err
		}

		res.WriteString(string(raw))
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

		path, err := file.Path()
		if err != nil {
			return nil, err
		}
		f, err := s.Runtime.Motor.Transport.FS().Open(path)
		if err != nil {
			return nil, err
		}

		raw, err := ioutil.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, err
		}

		contents[path] = string(raw)
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

func (s *lumiPamConf) GetEntries(files []interface{}) (map[string]interface{}, error) {
	contents := map[string]string{}
	var notReadyError error = nil

	for i := range files {
		file := files[i].(File)

		path, err := file.Path()
		if err != nil {
			return nil, err
		}
		f, err := s.Runtime.Motor.Transport.FS().Open(path)
		if err != nil {
			return nil, err
		}

		raw, err := ioutil.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, err
		}

		contents[path] = string(raw)
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
				entry, err := pam.ParseLine(line)
				if err != nil {
					return nil, err
				}
				pamEntry, err := s.Runtime.CreateResource("pam.conf.serviceEntry",
					"service", basename,
					"lineNumber", int64(i), //Used for ID
					"pamType", entry.PamType,
					"control", entry.Control,
					"module", entry.Module,
					"options", entry.Options,
				)
				if err != nil {
					return nil, err
				}
				settings = append(settings, pamEntry.(PamConfServiceEntry))
			}
		}

		services[basename] = settings
	}

	return services, nil
}
