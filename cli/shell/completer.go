// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shell

import (
	"strings"

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
	schema   resources.ResourcesSchema
	features cnquery.Features
}

// NewCompleter creates a new Mondoo completer object
func NewCompleter(schema resources.ResourcesSchema, features cnquery.Features, connectedProviders []string) *Completer {
	return &Completer{
		schema:   schema,
		features: features,
	}
}

// builtinCommands are shell commands that should appear in completions
var builtinCommands = []Suggestion{
	{Text: "exit", Description: "Exit the shell"},
	{Text: "quit", Description: "Exit the shell"},
	{Text: "help", Description: "Show available resources"},
	{Text: "clear", Description: "Clear the screen"},
}

// Complete returns suggestions for the given input text
func (c *Completer) Complete(text string) []Suggestion {
	if text == "" {
		return nil
	}
	var suggestions []Suggestion

	// Check for matching built-in commands first (only at the start of input)
	for _, cmd := range builtinCommands {
		if strings.HasPrefix(cmd.Text, text) {
			suggestions = append(suggestions, cmd)
		}
	}

	bundle, _ := mqlc.Compile(text, nil, mqlc.NewConfig(c.schema, c.features))
	if bundle != nil && len(bundle.Suggestions) > 0 {
		for i := range bundle.Suggestions {
			cur := bundle.Suggestions[i]
			suggestions = append(suggestions, Suggestion{
				Text:        cur.Field,
				Description: cur.Title,
			})
		}
	}

	return suggestions
}
