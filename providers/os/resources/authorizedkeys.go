package resources

import (
	"errors"
	"strconv"
	"strings"

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

func (x *mqlAuthorizedkeys) init(args map[string]interface{}) (map[string]interface{}, *mqlAuthorizedkeys, error) {
	// users may supply only the file or the path. Until we deprecate path in this
	// resource, we have to make sure it gets filled; if we receive a file,
	// set it from the file (for consistency)
	if v, ok := args["file"]; ok {
		file, ok := v.(*mqlFile)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'file' in authorizedkeys initialization, it must be a file")
		}

		args["path"] = file.Path.Data
	}

	if v, ok := args["path"]; ok {
		path, ok := v.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in authorizedkeys initialization, it must be a string")
		}

		f, err := CreateResource(x.MqlRuntime, "file", map[string]interface{}{
			"path": path,
		})
		if err != nil {
			return nil, nil, err
		}
		args["file"] = f
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

		opts := make([]interface{}, len(entry.Options))
		for j := range entry.Options {
			opts[j] = entry.Options[j]
		}

		ae, err := CreateResource(x.MqlRuntime, "authorizedkeys.entry", map[string]interface{}{
			"line":    entry.Line,
			"type":    entry.Key.Type(),
			"key":     entry.Base64Key(),
			"label":   entry.Label,
			"options": opts,
			"file":    file,
		})
		if err != nil {
			return nil, err
		}

		res = append(res, ae.(*mqlAuthorizedkeysEntry))
	}

	return res, nil
}
