// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package proto

//go:generate protoc --proto_path=../../:. --go_out=. --go_opt=paths=source_relative  --go-grpc_out=. --go-grpc_opt=paths=source_relative --go-vtproto_out=. --go-vtproto_opt=paths=source_relative --go-vtproto_opt=features=marshal+unmarshal+size cnquery.proto
