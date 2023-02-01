package scan

import (
	"context"
	"math/rand"
	"time"

	"go.mondoo.com/cnquery/cli/progress"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/vault/credentials_resolver"
	"go.mondoo.com/cnquery/resources"
)

//go:generate protoc --proto_path=../../:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. cnquery_explorer_scan.proto

func init() {
	rand.Seed(time.Now().UnixNano())
}

type AssetJob struct {
	DoRecord         bool
	UpstreamConfig   resources.UpstreamConfig
	Asset            *asset.Asset
	Bundle           *explorer.Bundle
	QueryPackFilters []string
	Props            map[string]string
	Ctx              context.Context
	CredsResolver    credentials_resolver.Resolver
	Reporter         Reporter
	connection       *motor.Motor
	ProgressReporter progress.Progress
}
