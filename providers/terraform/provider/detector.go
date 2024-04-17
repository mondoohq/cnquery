// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/terraform/connection"
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

	// we always prefer the git url since it is more reliable
	url, ok := asset.Connections[0].Options["ssh-url"]
	if ok {
		domain, org, repo, err := parseSSHURL(url)
		if err != nil {
			return err
		}
		platformID := "//platformid.api.mondoo.app/runtime/terraform/domain/" + domain + "/org/" + org + "/repo/" + repo
		asset.Connections[0].PlatformId = platformID
		asset.PlatformIds = []string{platformID}
		asset.Name = "Terraform Static Analysis " + org + "/" + repo
		return nil
	}

	projectPath, ok := asset.Connections[0].Options["path"]
	if ok {
		absPath, _ := filepath.Abs(projectPath)
		h := sha256.New()
		h.Write([]byte(absPath))
		hash := hex.EncodeToString(h.Sum(nil))
		platformID := "//platformid.api.mondoo.app/runtime/terraform/hash/" + hash
		asset.Connections[0].PlatformId = platformID
		asset.PlatformIds = []string{platformID}
		asset.Name = "Terraform Static Analysis " + parseNameFromPath(projectPath)
		return nil
	}

	return errors.New("could not determine platform id for Terraform asset")
}

func parseNameFromPath(file string) string {
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
			name = parseNameFromPath(abspath)
		}
	}

	return name
}

func parseSSHURL(url string) (string, string, string, error) {
	parts := strings.Split(url, "@")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("malformed URL")
	}

	// Get the provider
	providerParts := strings.Split(parts[1], ":")
	if len(providerParts) != 2 {
		return "", "", "", fmt.Errorf("malformed URL")
	}
	provider := providerParts[0]

	// Now split the second part at the slash to separate the org and repo
	orgRepoParts := strings.Split(providerParts[1], "/")

	// The repo name is the last part after the split. It includes .git,
	// so we remove that
	repo := strings.TrimSuffix(orgRepoParts[len(orgRepoParts)-1], ".git")

	return provider, orgRepoParts[0], repo, nil
}
