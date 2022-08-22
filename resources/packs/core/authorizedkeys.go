package core

import (
	"errors"
	"path"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/core/authorizedkeys"
)

func (u *mqlUser) GetAuthorizedkeys() (Authorizedkeys, error) {
	// fmt.Println("determine user authorized key file")
	home, err := u.Home()
	if err != nil {
		return nil, err
	}

	// TODO: we may need to handle ".ssh/authorized_keys2" too
	authorizedKeysPath := path.Join(home, ".ssh", "authorized_keys")
	ak, err := u.MotorRuntime.CreateResource("authorizedkeys", "path", authorizedKeysPath)
	if err != nil {
		return nil, err
	}
	return ak.(Authorizedkeys), nil
}

func (ake *mqlAuthorizedkeysEntry) id() (string, error) {
	file, err := ake.File()
	if err != nil {
		return "", err
	}

	path, err := file.Path()
	if err != nil {
		return "", err
	}

	line, err := ake.Line()
	if err != nil {
		return "", err
	}

	// composed of filepath + line number
	return path + ":" + strconv.FormatInt(line, 10), nil
}

func (ake *mqlAuthorizedkeysEntry) GetLabel() (string, error) {
	// NOTE: content can be overridden in constructor only
	return "", nil
}

func (ake *mqlAuthorizedkeysEntry) GetOptions() ([]string, error) {
	// NOTE: content can be overridden in constructor only
	return []string{}, nil
}

func (s *mqlAuthorizedkeys) init(args *resources.Args) (*resources.Args, Authorizedkeys, error) {
	// resolve path to file
	if x, ok := (*args)["path"]; ok {
		path, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in authorizedkeys initialization, it must be a string")
		}

		f, err := s.MotorRuntime.CreateResource("file", "path", path)
		if err != nil {
			return nil, nil, err
		}
		(*args)["file"] = f
	}

	return args, nil, nil
}

func authorizedkeysid(path string) string {
	return "authorizedkeys:" + path
}

func (a *mqlAuthorizedkeys) id() (string, error) {
	r, err := a.File()
	if err != nil {
		return "", err
	}
	path, err := r.Path()
	if err != nil {
		return "", err
	}

	return authorizedkeysid(path), nil
}

func (a *mqlAuthorizedkeys) GetFile() (File, error) {
	path, err := a.Path()
	if err != nil {
		return nil, err
	}

	f, err := a.MotorRuntime.CreateResource("file", "path", path)
	if err != nil {
		return nil, err
	}
	return f.(File), nil
}

func (a *mqlAuthorizedkeys) GetContent(file File) (string, error) {
	exists, err := file.Exists()
	if err != nil {
		return "", err
	}

	if !exists {
		return "", nil
	}

	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes
	err = a.MotorRuntime.WatchAndCompute(file, "content", a, "content")
	if err != nil {
		return "", err
	}

	return file.Content()
}

func (a *mqlAuthorizedkeys) GetList(file File, content string) ([]interface{}, error) {
	res := []interface{}{}

	exists, err := file.Exists()
	if err != nil {
		return res, err
	}

	if !exists {
		return res, nil
	}

	log.Debug().Msg("autorizedkeys> list...")

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

		ae, err := a.MotorRuntime.CreateResource("authorizedkeys.entry",
			"line", entry.Line,
			"type", entry.Key.Type(),
			"key", entry.Base64Key(),
			"label", entry.Label,
			"options", opts,
			"file", file,
		)
		if err != nil {
			return nil, err
		}

		res = append(res, ae.(AuthorizedkeysEntry))
	}

	return res, nil
}
