package vault

import "strings"

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --falcon_out=. vault.proto

func Mrn2secretKey(key string) string {
	return strings.TrimLeft(key, "//")
}
