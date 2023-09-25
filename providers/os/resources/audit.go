// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strconv"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
)

func (c *mqlAuditCvss) id() (string, error) {
	if c.Score.Error != nil {
		return "", c.Score.Error
	}
	score := c.Score.Data

	if c.Vector.Error != nil {
		return "", c.Vector.Error
	}
	vector := c.Vector.Data
	return "cvss/" + strconv.FormatFloat(score, 'f', 2, 64) + "/vector/" + vector, nil
}

func (c *mqlAuditAdvisory) id() (string, error) {
	return c.Mrn.Data, c.Mrn.Error
}

func (c *mqlAuditCve) id() (string, error) {
	return c.Mrn.Data, c.Mrn.Error
}

func initAuditAdvisory(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) == 7 {
		return args, nil, nil
	}

	if _, ok := args["mrn"]; !ok {
		return args, nil, fmt.Errorf("Initialized \"audit.advisory\" resource without a \"mrn\". This field is required.")
	}

	return args, nil, nil
}

func initAuditCve(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) == 8 {
		return args, nil, nil
	}

	if _, ok := args["mrn"]; !ok {
		return args, nil, fmt.Errorf("Initialized \"audit.cve\" resource without a \"mrn\". This field is required.")
	}

	return args, nil, nil
}

func initAuditCvss(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) == 2 {
		return args, nil, nil
	}

	if _, ok := args["score"]; !ok {
		return args, nil, fmt.Errorf("Initialized \"audit.cvss\" resource without a \"score\". This field is required.")
	}

	return args, nil, nil
}
