package vault

import (
	"encoding/json"
	"strings"

	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/falcon/codes"
	"go.mondoo.io/mondoo/falcon/status"
	"google.golang.org/protobuf/proto"
)

//go:generate protoc --proto_path=$PWD:. --go_out=. --go_opt=paths=source_relative --falcon_out=. vault.proto

func EscapeSecretID(key string) string {
	return strings.TrimPrefix(key, "//")
}

var NotFoundError = status.Error(codes.NotFound, "secret not found")

// Credential parses the secret data and creates a credential
func (x *Secret) Credential() (*Credential, error) {
	var cred Credential
	var err error

	switch x.Encoding {
	case SecretEncoding_encoding_proto:
		err = proto.Unmarshal(x.Data, &cred)
	case SecretEncoding_encoding_json:
		err = json.Unmarshal(x.Data, &cred)
	case SecretEncoding_encoding_binary:
		cred = Credential{
			// if binary is used, it needs to be overwriten from outside
			Secret: x.Data,
		}
	default:
		err = errors.New("unknown secret encoding")
	}

	if err != nil {
		return nil, errors.Wrap(err, "unknown secret format")
	}

	cred.SecretId = x.Key
	return &cred, nil
}

func NewSecret(cred *Credential, encoding SecretEncoding) (*Secret, error) {
	// TODO: we also encode the ID, this may not be a good approach
	var secretData []byte
	var err error

	switch encoding {
	case SecretEncoding_encoding_json:
		secretData, err = json.Marshal(cred)
	case SecretEncoding_encoding_proto:
		secretData, err = proto.Marshal(cred)
	default:
		return nil, errors.New("unknown secret encoding")
	}

	if err != nil {
		return nil, err
	}

	return &Secret{
		Key:      cred.SecretId,
		Data:     secretData,
		Encoding: encoding,
	}, nil
}
