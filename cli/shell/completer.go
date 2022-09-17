package shell

import (
	"github.com/c-bata/go-prompt"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/resources"
)

var completerSeparator = string([]byte{'.', ' '})

// Completer is an auto-complete helper for the shell
type Completer struct {
	schema      *resources.Schema
	features    cnquery.Features
	queryPrefix func() string
}

// NewCompleter creates a new Mondoo completer object
func NewCompleter(schema *resources.Schema, features cnquery.Features, queryPrefix func() string) *Completer {
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

	bundle, _ := mqlc.Compile(query, c.schema, c.features, nil)
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
