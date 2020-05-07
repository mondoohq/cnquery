// copyright: 2020, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package resources

import (
	"strconv"
	"strings"

	"go.mondoo.io/mondoo/lumi"
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
	if ((*args)["xdev"]) == nil {
		(*args)["xdev"] = false
	}

	return args, nil, nil
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

	var id strings.Builder
	id.WriteString(from)
	if !xdev {
		id.WriteString(" -xdev")
	}
	if typ != "" {
		id.WriteString(" type=" + typ)
	}

	return id.String(), nil
}

func (l *lumiFilesFind) GetXdev() (bool, error) {
	return false, nil
}

func (l *lumiFilesFind) GetType() (string, error) {
	return "", nil
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

	var call strings.Builder
	call.WriteString("find ")
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

	rawCmd, err := l.Runtime.CreateResource("command", "command", call.String())
	if err != nil {
		return nil, err
	}

	cmd := rawCmd.(Command)
	out, err := cmd.Stdout()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.Trim(out, " \t\n"), "\n")
	files := make([]interface{}, len(lines))
	var line string
	for i := range lines {
		line = lines[i]
		files[i], err = l.Runtime.CreateResource("file", "path", line)
		if err != nil {
			return nil, err
		}
	}

	// return the packages as new entries
	return files, nil
}
