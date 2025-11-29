// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package vault

import (
	"encoding/json"
	"strings"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	"google.golang.org/protobuf/proto"
)

type Resolver interface {
	GetCredential(cred *Credential) (*Credential, error)
}

//go:generate protoc --plugin=protoc-gen-go=../../../scripts/protoc/protoc-gen-go --plugin=protoc-gen-rangerrpc=../../../scripts/protoc/protoc-gen-rangerrpc --plugin=protoc-gen-go-vtproto=../../../scripts/protoc/protoc-gen-go-vtproto --proto_path=../../../:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. --go-vtproto_out=. --go-vtproto_opt=paths=source_relative --go-vtproto_opt=features=marshal+unmarshal+size+clone vault.proto

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
			// if binary is used, it needs to be over-written from outside
			Secret: x.Data,
		}
	default:
		err = errors.New("unknown secret encoding")
	}

	if err != nil {
		return nil, errors.Wrap(err, "unknown secret format")
	}

	cred.SecretId = x.Key
	cred.PreProcess()

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
		secretData, err = cred.MarshalVT()
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
