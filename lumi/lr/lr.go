// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package lr

import (
	"strings"

	"github.com/alecthomas/participle"
)

var (
	parser = participle.MustBuild(&LR{})
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

// LR are lumi resources parsed into an AST
type LR struct {
	Resources []*Resource `{ @@ }`
}

// Resource in LR
type Resource struct {
	ID       string         `@Ident { @'.' @Ident }`
	ListType *SimplListType `[ '{' [ @@ ]`
	Body     *ResourceDef   `@@ '}' ]`
}

type Type struct {
	MapType    *MapType    `( @@ |`
	ListType   *ListType   ` @@ |`
	SimpleType *SimpleType ` @@ )`
}

type SimplListType struct {
	Type SimpleType `'[' ']' @@`
}

type ListType struct {
	Type Type `'[' ']' @@`
}

type MapType struct {
	Key   SimpleType `'map' '[' @@ `
	Value Type       `']' @@`
}

type SimpleType struct {
	Type string `@Ident { @'.' @Ident }`
}

// ResourceDef carrying the definition of the resource
type ResourceDef struct {
	Inits  []*Init  `{ ( @@ `
	Fields []*Field `| @@ ) }`
}

// Init calls
type Init struct {
	Args []TypedArg `'init' '(' @@ { ',' @@ } ')'`
}

// TypedArg is an argument with a type
type TypedArg struct {
	ID   string `@Ident`
	Type Type   ` @@`
}

// Field definition of a resource
type Field struct {
	ID   string     `@Ident`
	Args *FieldArgs `[ '(' @@ ')' ]`
	Type Type       `[ @@ ]`
}

// Args list of arguments
type FieldArgs struct {
	List []SimpleType `[ @@ { ',' @@ } ]`
}

// Parse the input leise string to an AST
func Parse(input string) (*LR, error) {
	res := &LR{}
	err := parser.Parse(strings.NewReader(input), res)
	return res, err
}
