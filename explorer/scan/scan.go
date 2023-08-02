package scan

import (
	"context"
	"math/rand"
	"time"

	"go.mondoo.com/cnquery/cli/progress"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/providers"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/providers-sdk/v1/vault"
)

//go:generate protoc --proto_path=../../:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. cnquery_explorer_scan.proto

func init() {
	rand.Seed(time.Now().UnixNano())
}

type AssetJob struct {
	DoRecord         bool
	UpstreamConfig   upstream.UpstreamConfig
	Asset            *inventory.Asset
	Bundle           *explorer.Bundle
	QueryPackFilters []string
	Props            map[string]string
	Ctx              context.Context
	CredsResolver    vault.Resolver
	Reporter         Reporter
	runtime          *providers.Runtime
	ProgressReporter progress.Progress
}
