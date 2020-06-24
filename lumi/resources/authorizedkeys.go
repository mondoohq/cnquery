package resources

import (
	"errors"
	"path"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/authorizedkeys"
)

func (u *lumiUser) GetAuthorizedkeys() (Authorizedkeys, error) {
	// fmt.Println("determine user authorized key file")
	home, err := u.Home()

	// TODO: we may need to handle ".ssh/authorized_keys2" too
	authorizedKeysPath := path.Join(home, ".ssh", "authorized_keys")
	ak, err := u.Runtime.CreateResource("authorizedkeys", "path", authorizedKeysPath)
	if err != nil {
		return nil, err
	}
	return ak.(Authorizedkeys), nil
}

func (ake *lumiAuthorizedkeysEntry) id() (string, error) {
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

func (ake *lumiAuthorizedkeysEntry) GetLabel() (string, error) {
	// NOTE: content can be overridden in constructor only
	return "", nil
}

func (ake *lumiAuthorizedkeysEntry) GetOptions() ([]string, error) {
	// NOTE: content can be overridden in constructor only
	return []string{}, nil
}

func (s *lumiAuthorizedkeys) init(args *lumi.Args) (*lumi.Args, Authorizedkeys, error) {
	// resolve path to file
	if x, ok := (*args)["path"]; ok {
		path, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in authorizedkeys initialization, it must be a string")
		}

		f, err := s.Runtime.CreateResource("file", "path", path)
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

func (a *lumiAuthorizedkeys) id() (string, error) {
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

func (a *lumiAuthorizedkeys) GetFile() (File, error) {
	path, err := a.Path()
	if err != nil {
		return nil, err
	}

	f, err := a.Runtime.CreateResource("file", "path", path)
	if err != nil {
		return nil, err
	}
	return f.(File), nil
}

func (a *lumiAuthorizedkeys) GetContent(file File) (string, error) {
	exists, err := file.Exists()
	if err != nil {
		return "", err
	}

	if !exists {
		return "", nil
	}

	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes
	err = a.Runtime.WatchAndCompute(file, "content", a, "content")
	if err != nil {
		log.Error().Err(err).Msg("authorizedkeys> watch+compute failed")
	}

	return file.Content()
}

func (a *lumiAuthorizedkeys) GetList(file File, content string) ([]interface{}, error) {
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

		ae, err := a.Runtime.CreateResource("authorizedkeys.entry",
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
