package resources

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

func (u *lumiUser) GetSshkeys() ([]interface{}, error) {
	osProvider, err := osProvider(u.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	home, err := u.Home()
	if err != nil {
		return nil, err
	}

	userSshPath := path.Join(home, ".ssh")

	fs := osProvider.FS()
	afutil := afero.Afero{Fs: fs}

	// check if use ssh directory exists
	exists, err := afutil.Exists(userSshPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return res, nil
	}

	filter := []string{"config"}

	// walk dir and search for all private keys
	potentialPrivateKeyFiles := []string{}
	err = afutil.Walk(userSshPath, func(path string, f os.FileInfo, err error) error {
		if f == nil || f.IsDir() {
			return nil
		}

		// eg. matches google_compute_known_hosts and known_hosts
		if strings.HasSuffix(f.Name(), ".pub") || strings.HasSuffix(f.Name(), "known_hosts") {
			return nil
		}

		for i := range filter {
			if f.Name() == filter[i] {
				return nil
			}
		}

		potentialPrivateKeyFiles = append(potentialPrivateKeyFiles, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// iterate over files and check if the content is there
	for i := range potentialPrivateKeyFiles {
		log.Debug().Msg("load content from file")
		path := potentialPrivateKeyFiles[i]
		f, err := fs.Open(path)
		if err != nil {
			return nil, err
		}

		data, err := ioutil.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, err
		}

		content := string(data)

		// check if content contains PRIVATE KEY
		isPrivateKey := strings.Contains(content, "PRIVATE KEY")
		// check if the key is encrypted ENCRYPTED
		isEncrypted := strings.Contains(content, "ENCRYPTED")

		if isPrivateKey {
			upk, err := u.MotorRuntime.CreateResource("privatekey",
				"pem", content,
				"encrypted", isEncrypted,
				"path", path,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, upk.(Privatekey))
		}
	}

	return res, nil
}

func (r *lumiPrivatekey) id() (string, error) {
	// TODO: use path or hash depending on initialization
	path, err := r.Path()
	if err != nil {
		return "", err
	}

	return "privatekey:" + path, nil
}

func (r *lumiPrivatekey) GetPath() (string, error) {
	return "", errors.New("not implemented")
}

func (r *lumiPrivatekey) GetEncrypted() (bool, error) {
	return false, errors.New("not implemented")
}
