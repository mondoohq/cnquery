// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/ansible/play"
	"go.mondoo.com/mql/v13/providers/ansible/project"
)

var _ plugin.Connection = (*AnsibleConnection)(nil)

// AnsibleConnection connects to either a single playbook file or a whole
// Ansible project directory. The path's type at connect time decides the mode:
// a file populates playbook (single-playbook analysis, the original behavior),
// a directory populates proj (project-wide static analysis).
type AnsibleConnection struct {
	plugin.Connection
	Conf     *inventory.Config
	asset    *inventory.Asset
	path     string
	isDir    bool
	playbook play.Playbook
	proj     *project.Project
}

func NewAnsibleConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*AnsibleConnection, error) {
	if asset == nil || len(asset.Connections) == 0 {
		return nil, errors.New("ansible: no connection options for asset")
	}
	cc := asset.Connections[0]
	path := ""
	if cc.Options != nil {
		path = cc.Options["path"]
	}
	if path == "" {
		return nil, errors.New("ansible: no playbook path provided (set the `path` option)")
	}

	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("ansible: cannot access %q: %w", path, err)
	}

	conn := &AnsibleConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		path:       path,
		isDir:      fi.IsDir(),
	}

	if fi.IsDir() {
		proj, err := project.Load(path)
		if err != nil {
			return nil, fmt.Errorf("ansible: cannot load project %q: %w", path, err)
		}
		conn.proj = proj
		return conn, nil
	}

	playbook, err := loadPlaybookFile(path)
	if err != nil {
		return nil, err
	}
	conn.playbook = playbook
	return conn, nil
}

func loadPlaybookFile(path string) (play.Playbook, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ansible: cannot open playbook %q: %w", path, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("ansible: cannot read playbook %q: %w", path, err)
	}

	playbook, err := play.DecodePlaybook(data)
	if err != nil {
		return nil, fmt.Errorf("ansible: cannot decode playbook %q: %w", path, err)
	}
	return playbook, nil
}

func (c *AnsibleConnection) Name() string {
	return "ansible"
}

func (c *AnsibleConnection) Asset() *inventory.Asset {
	return c.asset
}

// IsProject reports whether the connection targets a project directory.
func (c *AnsibleConnection) IsProject() bool {
	return c.isDir
}

// Playbook returns the parsed single playbook (file mode). It is empty in
// project mode.
func (c *AnsibleConnection) Playbook() play.Playbook {
	return c.playbook
}

// Project returns the parsed project model (directory mode), or nil in file
// mode.
func (c *AnsibleConnection) Project() *project.Project {
	return c.proj
}

// BaseDir is the directory that relative include/import paths in the connected
// playbook resolve against: the project root in directory mode, or the
// playbook file's directory in file mode.
func (c *AnsibleConnection) BaseDir() string {
	if c.isDir {
		return c.path
	}
	return filepath.Dir(c.path)
}
