// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package lrcore

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"text/scanner"
	"unicode/utf8"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
)

// Int number type
type Int int64

// Float number type
type Float float64

// Bool for true/false
type Bool bool

// CommentToken captures a comment along with its source position,
// enabling detection of blank-line gaps between comment groups.
type CommentToken struct {
	Pos  lexer.Position
	Text string `@Comment` //nolint:govet // participle grammar tag
}

var CONTEXT_FIELD = "context"

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

// nolint: govet
type Alias struct {
	Definition SimpleType `@@`
	Type       SimpleType `'=' @@`
}

// LR are MQL resources parsed into an AST
// nolint: govet
type LR struct {
	Comments  []CommentToken `{ @@ }`
	Imports   []string       `{ "import" @String }`
	Options   Map            `{ "option" @(Ident '=' String) }`
	Aliases   []Alias        `{ "alias" @@ }`
	Resources []*Resource    `{ @@ }`
	imports   map[string]map[string]struct{}
	packPaths map[string]string
	aliases   map[string]*Resource
}

// Resource in LR
// nolint: govet
type Resource struct {
	Comments    []CommentToken `{ @@ }`
	IsPrivate   bool           `@"private"?`
	IsExtension bool           `@"extend"?`
	ID          string         `@Ident { @'.' @Ident }`
	Defaults    string         ` ( '@' "defaults" '(' @String ')' )? `
	Context     string         ` ( '@' "context" '(' @String ')' )? `
	Maturity    string         ` ( '@' "maturity" '(' @String ')' )? `
	ListType    *SimplListType `[ '{' [ @@ ]`
	Body        *ResourceDef   `@@ '}' ]`
	title       string
	desc        string
}

// gets the path for the field of the resource, e.g
// for resource A and field B this would be A.B
func (r Resource) GetFieldPaths() []string {
	if r.Body == nil {
		return []string{}
	}

	res := []string{}

	for _, f := range r.Body.Fields {
		if f.BasicField != nil {
			fullyQualifiedAccessor := fmt.Sprintf("%s.%s", r.ID, f.BasicField.ID)
			res = append(res, fullyQualifiedAccessor)
		}
	}
	return res
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
	Fields []*Field `{ @@ }`
}

// ResourceDef carrying the definition of the field
// nolint: govet
type Field struct {
	Comments   []CommentToken `{ @@ }`
	Init       *Init          `( @@ `
	Embeddable *Embeddable    `| @@`
	BasicField *BasicField    `| @@ )?`
}

// Init field definition
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

// Basic field definition of a resource
// nolint: govet
type BasicField struct {
	ID         string     `@Ident?`
	Args       *FieldArgs `[ '(' @@ ')' ]`
	Maturity   string     ` ( '@' "maturity" '(' @String ')' )? `
	Type       Type       `[ @@ ]`
	isEmbedded bool
}

// Field definition of a embeddable field resource
// nolint: govet
type Embeddable struct {
	Type  string  `"embed" @Ident { @'.' @Ident }`
	Alias *string `("as" @Ident)?`
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

func (r *Resource) GetInitFields() []*Init {
	inits := []*Init{}
	for _, f := range r.Body.Fields {
		if f.Init != nil {
			inits = append(inits, f.Init)
		}
	}
	return inits
}

func SanitizeComments(raw []CommentToken) []CommentToken {
	todoStart := -1
	for i := range raw {
		if raw[i].Text != "" {
			raw[i].Text = strings.Trim(raw[i].Text[2:], " \t\n")
		}
		if todoStart == -1 && strings.HasPrefix(raw[i].Text, "TODO") {
			todoStart = i
		}
	}
	if todoStart != -1 {
		raw = raw[0:todoStart]
	}
	return raw
}

// lastCommentGroup returns only the final contiguous group of comments,
// splitting on blank-line gaps (non-consecutive source lines). This prevents
// section-separator comment blocks from bleeding into resource/field titles.
func lastCommentGroup(comments []CommentToken) []CommentToken {
	if len(comments) <= 1 {
		return comments
	}
	lastGap := 0
	for i := 1; i < len(comments); i++ {
		if comments[i].Pos.Line != comments[i-1].Pos.Line+1 {
			lastGap = i
		}
	}
	return comments[lastGap:]
}

func extractTitleAndDescription(raw []CommentToken) (string, string) {
	if len(raw) == 0 {
		return "", ""
	}
	title := raw[0].Text
	// Skip the mandatory blank `//` separator between title and description
	// (enforced by validateDocCommentStructure), so it doesn't show up as a
	// leading space in the joined description.
	rest := raw[1:]
	if len(rest) > 0 && rest[0].Text == "" {
		rest = rest[1:]
	}
	parts := make([]string, len(rest))
	for i, c := range rest {
		parts[i] = c.Text
	}
	desc := strings.Join(parts, " ")
	return title, desc
}

// MaxTitleLength caps the rune count of a doc-comment title line. Titles
// render in CLI tables, auto-complete prompts, and the website resource
// docs, where a sprawling title looks bad and wrecks layout. Descriptions
// have no length cap.
const MaxTitleLength = 150

// validateDocCommentStructure enforces the doc-comment shape for resources
// and fields:
//   - 0 lines: nothing to validate.
//   - 1+ lines: line 1 is the title; it must be at most MaxTitleLength runes.
//   - 2+ lines: line 2 MUST be a blank `//` separator (Text == ""), so the
//     title stays a single line and the description starts cleanly. This
//     prevents accidentally truncated titles when a long one-liner wraps
//     onto two source lines.
//
// `context` is included verbatim in the error to identify the offending
// resource or field (e.g. "resource aws.billing.budget" or
// "field aws.billing.budget.budgetType").
func validateDocCommentStructure(comments []CommentToken, context string) error {
	if len(comments) == 0 {
		return nil
	}

	var errs []error

	if n := utf8.RuneCountInString(comments[0].Text); n > MaxTitleLength {
		errs = append(errs, fmt.Errorf(
			"%s: doc-comment title is %d characters (line %d), max is %d - "+
				"titles render in CLI tables, auto-complete, and the website docs, so keep them short; "+
				"move the rest into the description (a blank `//` followed by the longer text)",
			context, n, comments[0].Pos.Line, MaxTitleLength,
		))
	}

	if len(comments) >= 2 && comments[1].Text != "" {
		errs = append(errs, fmt.Errorf(
			"%s: doc-comment has %d lines but is missing the required blank `//` separator after the title (line %d) - "+
				"either collapse the comment to a single line, or insert a blank `//` line between the title and the description",
			context, len(comments), comments[0].Pos.Line,
		))
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
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
	var validationErrs []error
	for i := range res.Resources {
		resource := res.Resources[i]
		resource.Comments = SanitizeComments(resource.Comments)
		resource.Comments = lastCommentGroup(resource.Comments)
		if verr := validateDocCommentStructure(resource.Comments, "resource "+resource.ID); verr != nil {
			validationErrs = append(validationErrs, verr)
		}
		resource.title, resource.desc = extractTitleAndDescription(resource.Comments)
		resource.Comments = nil

		// List types have an implicit list field
		if resource.ListType != nil {
			t := resource.ListType.Type.Type
			args := resource.ListType.Args

			// args of nil tell the compiler that this field needs to be pre-populated
			// however for list we don't have this logic, it is always computed
			if args == nil {
				args = &FieldArgs{}
			}

			field := &BasicField{
				ID:   "list",
				Args: args,
				Type: Type{ListType: &ListType{Type: Type{SimpleType: &SimpleType{t}}}},
			}

			resource.Body.Fields = append(resource.Body.Fields, &Field{BasicField: field})
		}

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
			if arr[j].BasicField == nil && arr[j].Embeddable == nil && arr[j].Init == nil {
				arr[j], arr[ptr-1] = arr[ptr-1], arr[j]
				ptr--
			}
		}
		if ptr < len(arr) {
			resource.Body.Fields = arr[:ptr]
		}

		for i, f := range resource.Body.Fields {
			if f.Embeddable == nil {
				continue
			}
			var name string
			if f.Embeddable.Alias != nil {
				name = *f.Embeddable.Alias
			} else {
				// use the first part of the type name as a id, i.e. os for os.any
				// this won't work if there're are multiple embedded resources without aliases that share the same package, i.e os.any and os.base
				name = strings.Split(f.Embeddable.Type, ".")[0]
			}

			if name == CONTEXT_FIELD {
				return nil, errors.New("'" + CONTEXT_FIELD + "' field already exists on resource " + resource.ID)
			}

			newField := &Field{
				Comments: f.Comments,
				BasicField: &BasicField{
					ID:         name,
					Type:       Type{SimpleType: &SimpleType{f.Embeddable.Type}},
					Args:       &FieldArgs{},
					isEmbedded: true,
				},
			}
			resource.Body.Fields[i] = newField
		}

		if resource.Context != "" {
			// Synthetic token: Pos is intentionally zero-value since this comment
			// has no source location. Safe for lastCommentGroup (single-element
			// slice short-circuits before Pos is read).
			resource.Body.Fields = append(resource.Body.Fields, &Field{
				Comments: []CommentToken{{Text: "# Contextual info, where this resource is located and defined"}},
				BasicField: &BasicField{
					ID:         CONTEXT_FIELD,
					Args:       &FieldArgs{},
					Type:       Type{SimpleType: &SimpleType{resource.Context}},
					isEmbedded: false,
				},
			})
		}
	}

	if len(validationErrs) > 0 {
		return res, errors.Join(append([]error{err}, validationErrs...)...)
	}
	return res, err
}

// returns duplicate resources where duplicate means that one path leads to more than one field
// causing ambiguity. An example minimal LR that would cause duplicates is:
//
//	A {
//	  B A.B
//	}
//
//	A.B {
//	  value string
//	}
//
// in the case above 'A.B` could be interpreted as accessing the property 'B' of the resource 'A'
// or as accessing the resource 'A.B' directly.
func (lr *LR) GetDuplicates() []string {
	dups := []string{}
	seen := map[string]struct{}{}
	// first populate with the resource names (ids), so we don't have fields that
	// are the same as resource names
	for _, r := range lr.Resources {
		seen[r.ID] = struct{}{}
	}

	for _, r := range lr.Resources {
		fields := r.GetFieldPaths()
		for _, f := range fields {
			if _, ok := seen[f]; ok {
				dups = append(dups, f)
			}
			seen[f] = struct{}{}
		}
	}

	return dups
}
