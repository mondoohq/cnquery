package vault

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strings"
)

// PreProcess converts a more user-friendly configuration into a standard secrets form:
// eg. { user: "chris", password: "pwd"} will be converted to { type: "password", user: "chris", secret: "<byte pwd>"}
func (cred *Credential) PreProcess() {
	// load private key pem into secret
	if cred.PrivateKey != "" {
		cred.Secret = []byte(cred.PrivateKey)
		cred.PrivateKey = ""
		cred.Type = CredentialType_private_key
	}

	// NOTE: it is possible that private keys hold an additional password, therefore we only
	// copy the password into the secret when the credential type is password
	if (cred.Type == CredentialType_undefined || cred.Type == CredentialType_password) && cred.Password != "" {
		cred.Secret = []byte(cred.Password)
		cred.Password = ""
		cred.Type = CredentialType_password
	}
}

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

// UnmarshalJSON parses either an int or a string representation of
// SecretEncoding into the struct
func (s *SecretEncoding) UnmarshalJSON(data []byte) error {
	// check if we have a number
	var code int32
	err := json.Unmarshal(data, &code)
	if err == nil {
		*s = SecretEncoding(code)
	}
	if err != nil {
		var name string
		err = json.Unmarshal(data, &name)
		code, ok := SecretEncoding_value["encoding_"+strings.ToLower(strings.TrimSpace(name))]
		if !ok {
			return errors.New("unknown type value: " + string(data))
		}
		*s = SecretEncoding(code)
	}
	return nil
}

// MarshalJSON returns the JSON representation of SecretEncoding
// NOTE: we do not use pointers here to ensure its converted properly
// even if the struct is used directly
func (s SecretEncoding) MarshalJSON() ([]byte, error) {
	v, ok := SecretEncoding_name[int32(s)]
	if !ok {
		return nil, errors.New("could not marshal CredentialType")
	}
	return json.Marshal(v)
}
