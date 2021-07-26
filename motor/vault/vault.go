package vault

import (
	"strings"

	"go.mondoo.io/mondoo/falcon/codes"
	"go.mondoo.io/mondoo/falcon/status"
	"go.mondoo.io/mondoo/motor/transports"
	"google.golang.org/protobuf/proto"
)

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --falcon_out=. vault.proto

func EscapeSecretID(key string) string {
	return strings.TrimPrefix(key, "//")
}

var NotFoundError = status.Error(codes.NotFound, "secret not found")

func NewSecret(cred *transports.Credential) (*Secret, error) {
	// TODO: we also encode the ID, this may not be a good approach
	secretData, err := proto.Marshal(cred)
	if err != nil {
		return nil, err
	}

	return &Secret{
		Key:  cred.SecretId,
		Data: secretData,
	}, nil
}

func NewCredential(sec *Secret) (*transports.Credential, error) {
	var cred transports.Credential
	err := proto.Unmarshal(sec.Data, &cred)
	if err != nil {
		return nil, err
	}
	cred.SecretId = sec.Key
	return &cred, nil
}
