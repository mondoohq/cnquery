package vault

import "strings"

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --falcon_out=. vault.proto

func EscapeSecretID(key string) string {
	return strings.TrimPrefix(key, "//")
}
