package resources

import (
	"errors"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers/os/resources/authorizedkeys"
)

func (x *mqlAuthorizedkeysEntry) id() (string, error) {
	file := x.File.Data
	if file == nil {
		return "", errors.New("cannot determine authorized keys ID (missing file)")
	}

	path := file.Path.Data
	if path == "" {
		return "", errors.New("cannot determine authorized keys ID (missing file path)")
	}

	return path + ":" + strconv.FormatInt(x.Line.Data, 10), nil
}

func (x *mqlAuthorizedkeys) init(args map[string]*llx.RawData) (map[string]*llx.RawData, *mqlAuthorizedkeys, error) {
	// users may supply only the file or the path. Until we deprecate path in this
	// resource, we have to make sure it gets filled; if we receive a file,
	// set it from the file (for consistency)
	if v, ok := args["file"]; ok {
		file, ok := v.Value.(*mqlFile)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'file' in authorizedkeys initialization, it must be a file")
		}

		args["path"] = llx.StringData(file.Path.Data)
	}

	if path, ok := args["path"]; ok {
		f, err := CreateResource(x.MqlRuntime, "file", map[string]*llx.RawData{
			"path": path,
		})
		if err != nil {
			return nil, nil, err
		}

		args["file"] = llx.ResourceData(f, "file")
	}

	return args, nil, nil
}

func (x *mqlAuthorizedkeys) id() (string, error) {
	file := x.File.Data
	if file == nil {
		return "", errors.New("cannot determine authorized keys ID (missing file)")
	}

	path := file.Path.Data
	if path == "" {
		return "", errors.New("cannot determine authorized keys ID (missing file path)")
	}

	return "authorizedkeys:" + path, nil
}

func (a *mqlAuthorizedkeys) content(file *mqlFile) (string, error) {
	if !file.GetExists().Data {
		return "", file.Exists.Error
	}

	content := file.GetContent()
	return content.Data, content.Error
}

func (x *mqlAuthorizedkeys) list(file *mqlFile, content string) ([]interface{}, error) {
	res := []interface{}{}

	if !file.GetExists().Data {
		return res, file.Exists.Error
	}

	entries, err := authorizedkeys.Parse(strings.NewReader(content))
	if err != nil {
		return nil, err
	}

	for i := range entries {
		entry := entries[i]

		ae, err := CreateResource(x.MqlRuntime, "authorizedkeys.entry", map[string]*llx.RawData{
			"line":    llx.IntData(entry.Line),
			"type":    llx.StringData(entry.Key.Type()),
			"key":     llx.StringData(entry.Base64Key()),
			"label":   llx.StringData(entry.Label),
			"options": llx.ArrayData(llx.TArr2Raw[string](entry.Options), "string"),
			"file":    llx.ResourceData(file, "file"),
		})
		if err != nil {
			return nil, err
		}

		res = append(res, ae.(*mqlAuthorizedkeysEntry))
	}

	return res, nil
}
