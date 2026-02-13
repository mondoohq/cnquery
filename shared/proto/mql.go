// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package proto

//go:generate protoc --plugin=protoc-gen-go=../../scripts/protoc/protoc-gen-go --plugin=protoc-gen-go-grpc=../../scripts/protoc/protoc-gen-go-grpc --plugin=protoc-gen-go-vtproto=../../scripts/protoc/protoc-gen-go-vtproto --proto_path=../../:. --go_out=. --go_opt=paths=source_relative  --go-grpc_out=. --go-grpc_opt=paths=source_relative --go-vtproto_out=. --go-vtproto_opt=paths=source_relative --go-vtproto_opt=features=marshal+unmarshal+size mql.proto
