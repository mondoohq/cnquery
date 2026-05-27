// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

var _ plugin.Connection = (*BicepConnection)(nil)

type BicepConnection struct {
	plugin.Connection
	Conf            *inventory.Config
	asset           *inventory.Asset
	path            string
	bicepFiles      []*BicepFile
	bicepParamFiles []*BicepParamFile
	armTemplate     *ARMTemplate
}

// BicepFile holds a Bicep source file path and its raw content.
type BicepFile struct {
	Path    string
	Content string
}

// BicepParamFile holds a `.bicepparam` parameter file path and its raw content.
type BicepParamFile struct {
	Path    string
	Content string
}

// ARMTemplate holds a parsed ARM template JSON.
type ARMTemplate struct {
	Schema         string                     `json:"$schema"`
	ContentVersion string                     `json:"contentVersion"`
	Parameters     map[string]json.RawMessage `json:"parameters"`
	Variables      map[string]json.RawMessage `json:"variables"`
	Resources      []json.RawMessage          `json:"resources"`
	Outputs        map[string]json.RawMessage `json:"outputs"`
}

func NewBicepConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*BicepConnection, error) {
	if len(asset.Connections) == 0 {
		return nil, errors.New("no connections configured on asset")
	}

	conn := &BicepConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	cc := asset.Connections[0]
	bicepPath := cc.Options["path"]
	conn.path = bicepPath

	fi, err := os.Stat(bicepPath)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		files, err := loadBicepFiles(bicepPath)
		if err != nil {
			return nil, err
		}
		conn.bicepFiles = files
		paramFiles, err := loadBicepParamFiles(bicepPath)
		if err != nil {
			return nil, err
		}
		conn.bicepParamFiles = paramFiles
		// Check for ARM template JSON in the directory
		conn.armTemplate = findARMTemplate(bicepPath)
		if len(files) == 0 && len(paramFiles) == 0 && conn.armTemplate == nil {
			return nil, errors.New("no .bicep, .bicepparam, or ARM template JSON files found at " + bicepPath)
		}
	} else if strings.HasSuffix(bicepPath, ".json") {
		// Direct ARM template JSON
		tmpl, err := loadARMTemplate(bicepPath)
		if err != nil {
			return nil, err
		}
		conn.armTemplate = tmpl
	} else if strings.HasSuffix(bicepPath, ".bicepparam") {
		// Single .bicepparam parameter file
		content, err := os.ReadFile(bicepPath)
		if err != nil {
			return nil, err
		}
		conn.bicepParamFiles = []*BicepParamFile{{Path: bicepPath, Content: string(content)}}
	} else {
		// Single .bicep file
		content, err := os.ReadFile(bicepPath)
		if err != nil {
			return nil, err
		}
		conn.bicepFiles = []*BicepFile{{Path: bicepPath, Content: string(content)}}
	}

	return conn, nil
}

func (c *BicepConnection) Name() string {
	return "bicep"
}

func (c *BicepConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *BicepConnection) BicepFiles() []*BicepFile {
	return c.bicepFiles
}

func (c *BicepConnection) BicepParamFiles() []*BicepParamFile {
	return c.bicepParamFiles
}

func (c *BicepConnection) ARMTemplate() *ARMTemplate {
	return c.armTemplate
}

func (c *BicepConnection) Path() string {
	return c.path
}

func loadBicepFiles(dir string) ([]*BicepFile, error) {
	var files []*BicepFile

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".bicep") {
			content, err := os.ReadFile(path)
			if err != nil {
				log.Warn().Err(err).Str("path", path).Msg("failed to read bicep file")
				return nil
			}
			files = append(files, &BicepFile{Path: path, Content: string(content)})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func loadBicepParamFiles(dir string) ([]*BicepParamFile, error) {
	var files []*BicepParamFile

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".bicepparam") {
			content, err := os.ReadFile(path)
			if err != nil {
				log.Warn().Err(err).Str("path", path).Msg("failed to read bicepparam file")
				return nil
			}
			files = append(files, &BicepParamFile{Path: path, Content: string(content)})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func findARMTemplate(dir string) *ARMTemplate {
	// Look for common ARM template filenames
	candidates := []string{
		"azuredeploy.json",
		"mainTemplate.json",
		"template.json",
		"main.json",
	}
	for _, name := range candidates {
		path := filepath.Join(dir, name)
		if tmpl, err := loadARMTemplate(path); err == nil {
			return tmpl
		}
	}
	return nil
}

func loadARMTemplate(path string) (*ARMTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tmpl ARMTemplate
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return nil, err
	}

	// Verify it looks like an ARM template
	if tmpl.Schema == "" || !strings.Contains(tmpl.Schema, "deploymentTemplate") {
		return nil, errors.New("not an ARM template: " + path)
	}

	return &tmpl, nil
}
