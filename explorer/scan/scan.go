package scan

import (
	"context"
	"math/rand"
	"time"

	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/vault"
)

//go:generate protoc --proto_path=../../:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. cnquery_explorer_scan.proto

func init() {
	rand.Seed(time.Now().UnixNano())
}

type AssetJob struct {
	DoRecord         bool
	Asset            *asset.Asset
	Bundle           *explorer.Bundle
	QueryPackFilters []string
	Ctx              context.Context
	GetCredential    func(cred *vault.Credential) (*vault.Credential, error)
	Reporter         Reporter
	connection       *motor.Motor
}
