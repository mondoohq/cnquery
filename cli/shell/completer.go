// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shell

import (
	"go.mondoo.com/cnquery/v12"
	"go.mondoo.com/cnquery/v12/mqlc"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/resources"
)

// Suggestion represents a completion suggestion for the shell
type Suggestion struct {
	Text        string // The completion text
	Description string // Description shown in popup
}

// Completer is an auto-complete helper for the shell
type Completer struct {
	schema      resources.ResourcesSchema
	features    cnquery.Features
	queryPrefix func() string
}

// NewCompleter creates a new Mondoo completer object
func NewCompleter(schema resources.ResourcesSchema, features cnquery.Features, queryPrefix func() string) *Completer {
	return &Completer{
		schema:      schema,
		features:    features,
		queryPrefix: queryPrefix,
	}
}

// Complete returns suggestions for the given input text
func (c *Completer) Complete(text string) []Suggestion {
	if text == "" {
		return nil
	}

	var query string
	if c.queryPrefix != nil {
		query = c.queryPrefix()
	}
	query += text

	bundle, _ := mqlc.Compile(query, nil, mqlc.NewConfig(c.schema, c.features))
	if bundle == nil || len(bundle.Suggestions) == 0 {
		return nil
	}

	res := make([]Suggestion, len(bundle.Suggestions))
	for i := range bundle.Suggestions {
		cur := bundle.Suggestions[i]
		res[i] = Suggestion{
			Text:        cur.Field,
			Description: cur.Title,
		}
	}

	return res
}
