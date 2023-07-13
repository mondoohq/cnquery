package os

import (
	"errors"
	"io/ioutil"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/checksums"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/resources/packs/os/pam"
)

const (
	defaultPamConf = "/etc/pam.conf"
	defaultPamDir  = "/etc/pam.d"
)

func (s *mqlPamConf) init(args *resources.Args) (*resources.Args, PamConf, error) {
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

func (s *mqlPamConf) id() (string, error) {
	files, err := s.Files()
	if err != nil {
		return "", err
	}

	checksum := checksums.New
	for i := range files {
		c, err := files[i].(core.File).Path()
		if err != nil {
			return "", err
		}
		checksum = checksum.Add(c)
	}

	return checksum.String(), nil
}

func (se *mqlPamConfServiceEntry) id() (string, error) {
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

	id := s + "/" + lnstr + "/" + ptype

	// for include mod is empty
	if mod != "" {
		id += "/" + mod
	}

	return id, nil
}

func (s *mqlPamConf) getFiles(confPath string) ([]interface{}, error) {
	// check if the pam.d directory or pam config file exists
	mqlFile, err := s.MotorRuntime.CreateResource("file", "path", confPath)
	if err != nil {
		return nil, err
	}
	f := mqlFile.(core.File)
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
		return []interface{}{f.(core.File)}, nil
	}
}

func (s *mqlPamConf) getConfDFiles(confD string) ([]interface{}, error) {
	files, err := s.MotorRuntime.CreateResource("files.find", "from", confD, "type", "file")
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
func (s *mqlPamConf) GetFiles() ([]interface{}, error) {
	// check if the pam.d directory exists and is a directory
	// according to the pam spec, pam prefers the directory if it  exists over the single file config
	// see http://www.linux-pam.org/Linux-PAM-html/sag-configuration.html
	mqlFile, err := s.MotorRuntime.CreateResource("file", "path", defaultPamDir)
	if err != nil {
		return nil, err
	}
	f := mqlFile.(core.File)
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

func (s *mqlPamConf) GetContent(files []interface{}) (string, error) {
	osProvider, err := osProvider(s.MotorRuntime.Motor)
	if err != nil {
		return "", err
	}

	var res strings.Builder
	var notReadyError error = nil

	for i := range files {
		file := files[i].(core.File)

		path, err := file.Path()
		if err != nil {
			return "", err
		}
		f, err := osProvider.FS().Open(path)
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

func (s *mqlPamConf) GetServices(files []interface{}) (map[string]interface{}, error) {
	osProvider, err := osProvider(s.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	contents := map[string]string{}
	var notReadyError error = nil

	for i := range files {
		file := files[i].(core.File)

		path, err := file.Path()
		if err != nil {
			return nil, err
		}
		f, err := osProvider.FS().Open(path)
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

func (s *mqlPamConf) GetEntries(files []interface{}) (map[string]interface{}, error) {
	osProvider, err := osProvider(s.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	contents := map[string]string{}
	var notReadyError error = nil

	for i := range files {
		file := files[i].(core.File)

		path, err := file.Path()
		if err != nil {
			return nil, err
		}
		f, err := osProvider.FS().Open(path)
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

			entry, err := pam.ParseLine(line)
			if err != nil {
				return nil, err
			}

			// empty lines parse as empty object
			if entry == nil {
				continue
			}

			pamEntry, err := s.MotorRuntime.CreateResource("pam.conf.serviceEntry",
				"service", basename,
				"lineNumber", int64(i), // Used for ID
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

		services[basename] = settings
	}

	return services, nil
}
