// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"math/rand"
	"time"

	"go.mondoo.com/cnquery/v11/cli/progress"
	"go.mondoo.com/cnquery/v11/explorer"
	"go.mondoo.com/cnquery/v11/providers"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream"
)

//go:generate protoc --proto_path=../../:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. cnquery_explorer_scan.proto

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
