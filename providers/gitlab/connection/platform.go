// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"strconv"

	"github.com/xanzy/go-gitlab"
)

func (c *GitLabConnection) Identifier() (string, error) {
	grp, err := c.Group()
	if err != nil {
		return "", err
	}

	return "//platformid.api.mondoo.app/runtime/gitlab/group/" + strconv.Itoa(grp.ID), nil
}

func (c *GitLabConnection) Group() (*gitlab.Group, error) {
	grp, _, err := c.Client().Groups.GetGroup(c.GroupPath, nil)
	if err != nil {
		return nil, err
	}
	return grp, err
}
