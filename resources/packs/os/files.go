// copyright: 2020, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package os

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.mondoo.io/mondoo/motor/providers/os"

	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/providers"
)

var findTypes = map[string]string{
	"file":      "f",
	"directory": "d",
	"character": "c",
	"block":     "b",
	"socket":    "s",
	"link":      "l",
}

func (l *lumiFilesFind) init(args *lumi.Args) (*lumi.Args, FilesFind, error) {
	if (*args)["xdev"] == nil {
		(*args)["xdev"] = false
	}

	if (*args)["name"] == nil {
		(*args)["name"] = ""
	}

	if (*args)["permissions"] == nil {
		(*args)["permissions"] = int64(0o777)
	}

	return args, nil, nil
}

func octal2string(o int64) string {
	return fmt.Sprintf("%o", o)
}

func (l *lumiFilesFind) id() (string, error) {
	from, err := l.From()
	if err != nil {
		return "", err
	}

	xdev, err := l.Xdev()
	if err != nil {
		return "", err
	}

	typ, err := l.Type()
	if err != nil {
		return "", err
	}

	regex, err := l.Regex()
	if err != nil {
		return "", err
	}

	name, err := l.Name()
	if err != nil {
		return "", err
	}

	permissions, err := l.Permissions()
	if err != nil {
		return "", err
	}

	var id strings.Builder
	id.WriteString(from)
	if !xdev {
		id.WriteString(" -xdev")
	}
	if typ != "" {
		id.WriteString(" type=" + typ)
	}

	if typ != "" {
		id.WriteString(" regex=" + regex)
	}

	if name != "" {
		id.WriteString(" name=" + name)
	}

	if permissions != 0o777 {
		id.WriteString(" permissions=" + octal2string(permissions))
	}

	return id.String(), nil
}

func (l *lumiFilesFind) GetXdev() (bool, error) {
	return false, nil
}

func (l *lumiFilesFind) GetType() (string, error) {
	return "", nil
}

func (l *lumiFilesFind) GetRegex() (string, error) {
	return "", nil
}

func (l *lumiFilesFind) GetName() (string, error) {
	return "", nil
}

func (l *lumiFilesFind) GetPermissions() (int64, error) {
	return 0, nil
}

func (l *lumiFilesFind) GetList() ([]interface{}, error) {
	path, err := l.From()
	if err != nil {
		return nil, err
	}

	xdev, err := l.Xdev()
	if err != nil {
		return nil, err
	}

	typ, err := l.Type()
	if err != nil {
		return nil, err
	}

	var compiledRegexp *regexp.Regexp
	regex, err := l.Regex()
	if err != nil {
		return nil, err
	}
	if len(regex) > 0 {
		compiledRegexp, err = regexp.Compile(regex)
		if err != nil {
			return nil, err
		}
	}

	perm, err := l.Permissions()
	if err != nil {
		return nil, err
	}

	name, err := l.Name()
	if err != nil {
		return nil, err
	}

	osProvider, err := osProvider(l.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	var foundFiles []string
	caps := l.MotorRuntime.Motor.Provider.Capabilities()
	if caps.HasCapability(providers.Capability_FileSearch) {
		fs := osProvider.FS()

		fsSearch, ok := fs.(os.FileSearch)
		if !ok {
			return nil, errors.New("find is not supported for your platform")
		}

		foundFiles, err = fsSearch.Find(path, compiledRegexp, typ)
		if err != nil {
			return nil, err
		}
	} else if caps.HasCapability(providers.Capability_RunCommand) {
		var call strings.Builder
		call.WriteString("find -L ")
		call.WriteString(strconv.Quote(path))

		if !xdev {
			call.WriteString(" -xdev")
		}

		if typ != "" {
			t, ok := findTypes[typ]
			if ok {
				call.WriteString(" -type " + t)
			}
		}

		if regex != "" {
			// TODO: we need to escape regex here
			call.WriteString(" -regex '")
			call.WriteString(regex)
			call.WriteString("'")
		}

		if perm != 0o777 {
			call.WriteString(" -perm -")
			call.WriteString(octal2string(perm))
		}

		if name != "" {
			call.WriteString(" -name ")
			call.WriteString(name)
		}

		rawCmd, err := l.MotorRuntime.CreateResource("command", "command", call.String())
		if err != nil {
			return nil, err
		}

		cmd := rawCmd.(Command)
		out, err := cmd.Stdout()
		if err != nil {
			return nil, err
		}

		foundFiles = strings.Split(strings.Trim(out, " \t\n"), "\n")
	} else {
		return nil, errors.New("find is not supported for your platform")
	}

	files := make([]interface{}, len(foundFiles))
	var filepath string
	for i := range foundFiles {
		filepath = foundFiles[i]
		files[i], err = l.MotorRuntime.CreateResource("file", "path", filepath)
		if err != nil {
			return nil, err
		}
	}

	// return the packages as new entries
	return files, nil
}
