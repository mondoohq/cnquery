// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shell

import (
	"github.com/c-bata/go-prompt"
	"go.mondoo.com/cnquery/v9"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/mqlc"
)

var completerSeparator = string([]byte{'.', ' '})

// Completer is an auto-complete helper for the shell
type Completer struct {
	schema      llx.Schema
	features    cnquery.Features
	queryPrefix func() string
}

// NewCompleter creates a new Mondoo completer object
func NewCompleter(schema llx.Schema, features cnquery.Features, queryPrefix func() string) *Completer {
	return &Completer{
		schema:      schema,
		features:    features,
		queryPrefix: queryPrefix,
	}
}

// CompletePrompt provides suggestions
func (c *Completer) CompletePrompt(doc prompt.Document) []prompt.Suggest {
	if doc.TextBeforeCursor() == "" {
		return []prompt.Suggest{}
	}

	var query string
	if c.queryPrefix != nil {
		query = c.queryPrefix()
	}
	query += doc.TextBeforeCursor()

	bundle, _ := mqlc.Compile(query, nil, mqlc.NewConfig(c.schema, c.features))
	if bundle == nil || len(bundle.Suggestions) == 0 {
		return []prompt.Suggest{}
	}

	res := make([]prompt.Suggest, len(bundle.Suggestions))
	for i := range bundle.Suggestions {
		cur := bundle.Suggestions[i]
		res[i] = prompt.Suggest{
			Text:        cur.Field,
			Description: cur.Title,
		}
	}

	return res

	// Alternatively we can decide to let prompt filter this list of words for us:
	// return prompt.FilterHasPrefix(suggest, doc.GetWordBeforeCursor(), true)
}
