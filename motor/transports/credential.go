package transports

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strings"
)

// UnmarshalJSON parses either an int or a string representation of
// CredentialType into the struct
func (s *CredentialType) UnmarshalJSON(data []byte) error {
	// check if we have a number
	var code int32
	err := json.Unmarshal(data, &code)
	if err == nil {
		*s = CredentialType(code)
	}
	if err != nil {
		var name string
		err = json.Unmarshal(data, &name)
		code, ok := CredentialType_value[strings.TrimSpace(name)]
		if !ok {
			return errors.New("unknown type value: " + string(data))
		}
		*s = CredentialType(code)
	}
	return nil
}

// MarshalJSON returns the JSON representation of CredentialType
// NOTE: we do not use pointers here to ensure its converted properly
// even if the struct is used directly
func (s CredentialType) MarshalJSON() ([]byte, error) {
	v, ok := CredentialType_name[int32(s)]
	if !ok {
		return nil, errors.New("could not marshal CredentialType")
	}
	return json.Marshal(v)
}

func NewPrivateKeyCredential(user string, pemBytes []byte, password string) *Credential {
	return &Credential{
		Type:     CredentialType_private_key,
		User:     user,
		Secret:   pemBytes,
		Password: password,
	}
}

func NewPrivateKeyCredentialFromPath(user string, path string, password string) (*Credential, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, errors.New("private key does not exist " + path)
	}

	pemBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return NewPrivateKeyCredential(user, pemBytes, password), nil
}

func NewPasswordCredential(user string, password string) *Credential {
	return &Credential{
		Type:   CredentialType_password,
		User:   user,
		Secret: []byte(password),
	}
}

// GetPassword returns the first password in the list
func GetPassword(list []*Credential) (*Credential, error) {
	for i := range list {
		credential := list[i]
		if credential.Type == CredentialType_password {
			return credential, nil
		}
	}
	return nil, errors.New("no password found")
}
