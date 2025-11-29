// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"math/rand"
	"time"

	"go.mondoo.com/cnquery/v12/cli/progress"
	"go.mondoo.com/cnquery/v12/explorer"
	"go.mondoo.com/cnquery/v12/providers"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/upstream"
)

//go:generate protoc --plugin=protoc-gen-go=../../scripts/protoc/protoc-gen-go --plugin=protoc-gen-rangerrpc=../../scripts/protoc/protoc-gen-rangerrpc --plugin=protoc-gen-go-vtproto=../../scripts/protoc/protoc-gen-go-vtproto --proto_path=../../:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. --go-vtproto_out=. --go-vtproto_opt=paths=source_relative --go-vtproto_opt=features=marshal+unmarshal+size cnquery_explorer_scan.proto

func init() {
	rand.Seed(time.Now().UnixNano())
}

type AssetJob struct {
	DoRecord         bool
	UpstreamConfig   *upstream.UpstreamConfig
	Asset            *inventory.Asset
	Bundle           *explorer.Bundle
	QueryPackFilters []string
	Props            map[string]string
	Ctx              context.Context
	Reporter         Reporter
	runtime          *providers.Runtime
	ProgressReporter progress.Progress
}
