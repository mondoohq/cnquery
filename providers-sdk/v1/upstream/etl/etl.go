// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package etl

//go:generate protoc --plugin=protoc-gen-go=../../../../scripts/protoc/protoc-gen-go --plugin=protoc-gen-go-vtproto=../../../../scripts/protoc/protoc-gen-go-vtproto --go_out=. --go_opt=paths=source_relative --go-vtproto_out=. --go-vtproto_opt=paths=source_relative --go-vtproto_opt=features=marshal+unmarshal+size etl.proto
