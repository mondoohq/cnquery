// copyright: 2020, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package resources

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
)

var findTypes = map[string]string{
	"file":      "f",
	"directory": "d",
	"character": "c",
	"block":     "b",
	"socket":    "s",
	"link":      "l",
}

func initFilesFind(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if args["permissions"] == nil {
		args["permissions"] = llx.IntData(int64(0o777))
	}

	return args, nil, nil
}

func octal2string(o int64) string {
	return fmt.Sprintf("%o", o)
}

func (l *mqlFilesFind) id() (string, error) {
	var id strings.Builder
	id.WriteString(l.From.Data)
	if !l.Xdev.Data {
		id.WriteString(" -xdev")
	}
	if l.Type.Data != "" {
		id.WriteString(" type=" + l.Type.Data)
	}

	if l.Regex.Data != "" {
		id.WriteString(" regex=" + l.Regex.Data)
	}

	if l.Name.Data != "" {
		id.WriteString(" name=" + l.Name.Data)
	}

	if l.Permissions.Data != 0o777 {
		id.WriteString(" permissions=" + octal2string(l.Permissions.Data))
	}

	return id.String(), nil
}

func (l *mqlFilesFind) list() ([]interface{}, error) {
	var err error
	var compiledRegexp *regexp.Regexp
	if len(l.Regex.Data) > 0 {
		compiledRegexp, err = regexp.Compile(l.Regex.Data)
		if err != nil {
			return nil, err
		}
	}

	var foundFiles []string
	conn := l.MqlRuntime.Connection.(shared.Connection)
	if conn.Capabilities().Has(shared.Capability_FindFile) {
		fs := conn.FileSystem()
		fsSearch, ok := fs.(shared.FileSearch)
		if !ok {
			return nil, errors.New("find is not supported for your platform")
		}

		foundFiles, err = fsSearch.Find(l.From.Data, compiledRegexp, l.Type.Data)
		if err != nil {
			return nil, err
		}
	} else if conn.Capabilities().Has(shared.Capability_RunCommand) {
		var call strings.Builder
		call.WriteString("find -L ")
		call.WriteString(strconv.Quote(l.From.Data))

		if !l.Xdev.Data {
			call.WriteString(" -xdev")
		}

		if l.Type.Data != "" {
			t, ok := findTypes[l.Type.Data]
			if ok {
				call.WriteString(" -type " + t)
			}
		}

		if l.Regex.Data != "" {
			// TODO: we need to escape regex here
			call.WriteString(" -regex '")
			call.WriteString(l.Regex.Data)
			call.WriteString("'")
		}

		if l.Permissions.Data != 0o777 {
			call.WriteString(" -perm -")
			call.WriteString(octal2string(l.Permissions.Data))
		}

		if l.Name.Data != "" {
			call.WriteString(" -name ")
			call.WriteString(l.Name.Data)
		}

		rawCmd, err := CreateResource(l.MqlRuntime, "command", map[string]*llx.RawData{
			"command": llx.StringData(call.String()),
		})
		if err != nil {
			return nil, err
		}

		cmd := rawCmd.(*mqlCommand)
		out := cmd.GetStdout()
		if out.Error != nil {
			return nil, out.Error
		}

		lines := strings.TrimSpace(out.Data)
		if lines == "" {
			foundFiles = []string{}
		} else {
			foundFiles = strings.Split(lines, "\n")
		}
	} else {
		return nil, errors.New("find is not supported for your platform")
	}

	files := make([]interface{}, len(foundFiles))
	var filepath string
	for i := range foundFiles {
		filepath = foundFiles[i]
		files[i], err = CreateResource(l.MqlRuntime, "file", map[string]*llx.RawData{
			"path": llx.StringData(filepath),
		})
		if err != nil {
			return nil, err
		}
	}

	// return the packages as new entries
	return files, nil
}
