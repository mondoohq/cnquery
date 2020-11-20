package vault

import "strings"

//go:generate protoc --proto_path=$GOPATH/src:. --gofast_out=. --falcon_out=. vault.proto

func Mrn2secretKey(key string) string {
	return strings.TrimLeft(key, "//")
}
