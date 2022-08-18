// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package lr

import (
	"io"
	"strings"
	"text/scanner"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
)

// Int number type
type Int int64

// Float number type
type Float float64

// Bool for true/false
type Bool bool

// Capture a Bool type for participle
func (b *Bool) Capture(values []string) error {
	*b = values[0] == "true"
	return nil
}

type Map map[string]string

func (m *Map) Capture(values []string) error {
	if len(values) == 0 {
		return nil
	}

	if *m == nil {
		*m = map[string]string{}
	}
	(*m)[values[0]] = values[2]
	return nil
}

// LR are lumi resources parsed into an AST
// nolint: govet
type LR struct {
	Imports   []string    `{ "import" @String }`
	Options   Map         `{ "option" @(Ident '=' String) }`
	Resources []*Resource `{ @@ }`
	imports   map[string]map[string]struct{}
	packPaths map[string]string
}

// Resource in LR
// nolint: govet
type Resource struct {
	Comments  []string       `{ @Comment }`
	IsPrivate bool           `@"private"?`
	ID        string         `@Ident { @'.' @Ident }`
	ListType  *SimplListType `[ '{' [ @@ ]`
	Body      *ResourceDef   `@@ '}' ]`
	title     string
	desc      string
}

// nolint: govet
type Type struct {
	MapType    *MapType    `( @@ |`
	ListType   *ListType   ` @@ |`
	SimpleType *SimpleType ` @@ )`
}

// nolint: govet
type SimplListType struct {
	Type SimpleType `'[' ']' @@`
	Args *FieldArgs `[ '(' @@ ')' ]`
}

// nolint: govet
type ListType struct {
	Type Type `'[' ']' @@`
}

// nolint: govet
type MapType struct {
	Key   SimpleType `'map' '[' @@ `
	Value Type       `']' @@`
}

// nolint: govet
type SimpleType struct {
	Type string `@Ident { @'.' @Ident }`
}

// ResourceDef carrying the definition of the resource
// nolint: govet
type ResourceDef struct {
	Inits  []*Init  `{ ( @@ `
	Fields []*Field `| @@ ) }`
}

// Init calls
// nolint: govet
type Init struct {
	Args []TypedArg `'init' '(' @@ { ',' @@ } ')'`
}

// TypedArg is an argument with a type
// nolint: govet
type TypedArg struct {
	ID       string `@Ident`
	Optional bool   `@'?'?`
	Type     Type   ` @@`
}

// Field definition of a resource
// nolint: govet
type Field struct {
	Comments []string   `{ @Comment }`
	ID       string     `@Ident?`
	Args     *FieldArgs `[ '(' @@ ')' ]`
	Type     Type       `[ @@ ]`
}

// Args list of arguments
// nolint: govet
type FieldArgs struct {
	List []SimpleType `[ @@ { ',' @@ } ]`
}

// LEXER

type lrLexer struct{}

func (l *lrLexer) Lex(r io.Reader) (lexer.Lexer, error) {
	var scannerObj scanner.Scanner
	lexerObj := lexer.LexWithScanner(r, &scannerObj)
	scannerObj.Mode ^= scanner.SkipComments
	return lexerObj, nil
}

func (l *lrLexer) Symbols() map[string]rune {
	return map[string]rune{
		"EOF":       scanner.EOF,
		"Char":      scanner.Char,
		"Ident":     scanner.Ident,
		"Int":       scanner.Int,
		"Float":     scanner.Float,
		"String":    scanner.String,
		"RawString": scanner.RawString,
		"Comment":   scanner.Comment,
	}
}

func extractComments(raw []string) (string, string) {
	if len(raw) == 0 {
		return "", ""
	}

	for i := range raw {
		if raw[i] != "" {
			raw[i] = strings.Trim(raw[i][2:], " \t\n")
		}
	}

	title, rest := raw[0], raw[1:]
	desc := strings.Join(rest, " ")

	return title, desc
}

// Parse the input leise string to an AST
func Parse(input string) (*LR, error) {
	res := &LR{}

	var lexer lrLexer
	parser := participle.MustBuild(&LR{},
		participle.Lexer(&lexer),
	)

	err := parser.Parse(strings.NewReader(input), res)

	// clean up the parsed results
	for i := range res.Resources {
		resource := res.Resources[i]

		resource.title, resource.desc = extractComments(resource.Comments)
		resource.Comments = nil

		if resource.Body == nil {
			continue
		}
		if len(resource.Body.Fields) == 0 {
			continue
		}

		// eliminate fields that are comment-only (no ID)
		arr := resource.Body.Fields
		ptr := len(arr)
		for j := 0; j < ptr; j++ {
			if arr[j].ID == "" {
				arr[j], arr[ptr-1] = arr[ptr-1], arr[j]
				ptr--
			}
		}
		if ptr < len(arr) {
			resource.Body.Fields = arr[:ptr]
		}
	}

	return res, err
}
