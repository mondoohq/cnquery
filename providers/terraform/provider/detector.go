// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"

	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/terraform/connection"
)

func (s *Service) detect(asset *inventory.Asset, conn *connection.Connection) error {
	var p *inventory.Platform
	switch conn.Type() {
	case "state":
		p = &inventory.Platform{
			Name:    "terraform-state",
			Title:   "Terraform State",
			Family:  []string{"terraform"},
			Kind:    "code",
			Runtime: "terraform",
		}
	case "plan":
		p = &inventory.Platform{
			Name:    "terraform-plan",
			Title:   "Terraform Plan",
			Family:  []string{"terraform"},
			Kind:    "code",
			Runtime: "terraform",
		}
	default:
		p = &inventory.Platform{
			Name:    "terraform-hcl",
			Title:   "Terraform HCL",
			Family:  []string{"terraform"},
			Kind:    "code",
			Runtime: "terraform",
		}
	}
	asset.Platform = p

	projectPath := asset.Connections[0].Options["path"]
	absPath, _ := filepath.Abs(projectPath)
	h := sha256.New()
	h.Write([]byte(absPath))
	hash := hex.EncodeToString(h.Sum(nil))
	platformID := "//platformid.api.mondoo.app/runtime/terraform/hash/" + hash
	asset.Connections[0].PlatformId = platformID

	return nil
}
