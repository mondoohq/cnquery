package vault

import (
	"strings"

	"go.mondoo.io/mondoo/falcon/codes"
	"go.mondoo.io/mondoo/falcon/status"
)

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --falcon_out=. vault.proto

func EscapeSecretID(key string) string {
	return strings.TrimPrefix(key, "//")
}

var NotFoundError = status.Error(codes.NotFound, "secret not found")
