// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package gql

import (
	"context"

	"github.com/shurcooL/graphql"
	mondoogql "go.mondoo.com/mondoo-go"
)

type Policy struct {
	Mrn          string
	Name         string
	Assigned     bool
	Action       string
	Version      string
	IsPublic     bool
	CreatedAt    string
	UpdatedAt    string
	MqueryCounts struct {
		Total int
	}
}

type PolicyReportSummaryOrder struct {
	Direction string `json:"direction"`
	Field     string `json:"field"`
}

type Node struct {
	Policy Policy
}

type Edge struct {
	Cursor string
	Node   Node
}

type PolicyReportSummariesConnection struct {
	Edges []Edge
}

type SpaceReport struct {
	SpaceMrn              string
	PolicyReportSummaries PolicyReportSummariesConnection `graphql:"policyReportSummaries(first: $first, after: $after, orderBy: $orderBy)"`
}

func (c *MondooClient) GetSpaceReport(mrn string) (*SpaceReport, error) {
	var m struct {
		SpaceReport struct {
			SpaceReport `graphql:"... on SpaceReport"`
		} `graphql:"spaceReport(input: $input)"`
	}
	orderBy := PolicyReportSummaryOrder{
		Direction: "DESC",
		Field:     "TITLE",
	}
	err := c.Query(context.Background(), &m, map[string]interface{}{
		"input":   mondoogql.SpaceReportInput{SpaceMrn: mondoogql.String(mrn)},
		"first":   graphql.Int(10),
		"after":   graphql.String(""),
		"orderBy": orderBy,
	})
	if err != nil {
		return nil, err
	}

	// Map the data from the GraphQL response to the SpaceReport struct
	gqlSpaceReport := &SpaceReport{
		SpaceMrn:              m.SpaceReport.SpaceReport.SpaceMrn,
		PolicyReportSummaries: m.SpaceReport.SpaceReport.PolicyReportSummaries,
	}

	return gqlSpaceReport, nil
}
