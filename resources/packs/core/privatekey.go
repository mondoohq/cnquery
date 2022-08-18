package core

import "errors"

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
