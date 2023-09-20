// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/terraform/connection"
)

func (s *Service) detect(asset *inventory.Asset, conn *connection.Connection) error {
	var p *inventory.Platform
	connType := asset.Connections[0].Type
	switch connType {
	case StateConnectionType:
		p = &inventory.Platform{
			Name:    "terraform-state",
			Title:   "Terraform State",
			Family:  []string{"terraform"},
			Kind:    "code",
			Runtime: "terraform",
		}
	case PlanConnectionType:
		p = &inventory.Platform{
			Name:    "terraform-plan",
			Title:   "Terraform Plan",
			Family:  []string{"terraform"},
			Kind:    "code",
			Runtime: "terraform",
		}
	case HclConnectionType:
		fallthrough
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
	asset.PlatformIds = []string{platformID}

	name := ""
	if projectPath != "" {
		// manifest parent directory name
		name = projectNameFromPath(projectPath)
	}
	asset.Name = "Terraform Static Analysis " + name

	return nil
}

func projectNameFromPath(file string) string {
	// if it is a local file (which may not be true)
	name := ""
	fi, err := os.Stat(file)
	if err == nil {
		if fi.IsDir() && fi.Name() != "." {
			name = "directory " + fi.Name()
		} else if fi.IsDir() {
			name = fi.Name()
		} else {
			name = filepath.Base(fi.Name())
			extension := filepath.Ext(name)
			name = strings.TrimSuffix(name, extension)
		}
	} else {
		// it is not a local file, so we try to be a bit smart
		name = path.Base(file)
		extension := path.Ext(name)
		name = strings.TrimSuffix(name, extension)
	}

	// if the path is . we read the current directory
	if name == "." {
		abspath, err := filepath.Abs(name)
		if err == nil {
			name = projectNameFromPath(abspath)
		}
	}

	return name
}
