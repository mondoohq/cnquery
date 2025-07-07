// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package etl

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-vtproto_out=. --go-vtproto_opt=paths=source_relative --go-vtproto_opt=features=marshal+unmarshal+size etl.proto
