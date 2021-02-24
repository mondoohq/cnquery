package types

import (
	"encoding/json"
	"strings"
)

// Type information
type Type string

// MarshalJSON generates escapes the \u0000 string for postgres
// Otherwise we are not able to store the compile code as json blob in pg since
// llx and type use \x00 or \u0000. This is not allowed in Postgres json blobs
// see https://www.postgresql.org/docs/9.4/release-9-4-1.html
func (typ Type) MarshalJSON() ([]byte, error) {
	newVal := strings.ReplaceAll(string(typ), "\u0000", "\\u0000")
	return json.Marshal(newVal)
}

// UnmarshalJSON reverts MarshalJSON data arrays to its base type.
func (typ *Type) UnmarshalJSON(data []byte) error {
	var d string
	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}
	reverted := strings.ReplaceAll(d, "\\u0000", "\u0000")
	*typ = Type(reverted)
	return nil
}

const (
	byteUnset byte = iota
	byteAny
	byteNil
	byteRef
	byteBool
	byteInt
	byteFloat
	byteString
	byteRegex
	byteTime
	byteDict
	byteScore
	byteBlock
	byteArray = 1<<4 + iota - 4 // set to 25 to avoid breaking changes
	byteMap
	byteResource
	byteFunction
)

// Empty type is one whose type information is not available at all
const Empty Type = ""

const (
	// Unset type indicates that the type has not yet been set
	Unset = Type(rune(byteUnset))
	// Any type indicating an untyped value that can have any type
	Any = Type(rune(byteAny))
	// Nil for the empty type
	Nil = Type(rune(byteNil))
	// Ref for internal code chunk references
	Ref = Type(rune(byteRef))
	// Bool for the booleans true and false
	Bool = Type(rune(byteBool))
	// Int for integers
	Int = Type(rune(byteInt))
	// Float for any decimal values
	Float = Type(rune(byteFloat))
	// String for strings
	String = Type(rune(byteString))
	// Regex for regular expressions
	Regex = Type(rune(byteRegex))
	// Time for date and time
	Time = Type(rune(byteTime))
	// Dict for storing hierarchical simple key-value assignemnts
	Dict = Type(rune(byteDict))
	// Score for evaluations
	Score = Type(rune(byteScore))
	// Block evaluation results
	Block = Type(rune(byteBlock))
	// ArrayLike is the underlying type of all arrays
	ArrayLike = Type(rune(byteArray))
	// MapLike is the underlying type of all maps
	MapLike = Type(rune(byteMap))
	// ResourceLike is the underlying type of all resources
	ResourceLike = Type(rune(byteResource))
	// FunctionLike is the underlying type of all functions
	FunctionLike = Type(rune(byteFunction))
)

// IsEmpty returns true if the type has no information
func (typ Type) IsEmpty() bool {
	return typ == ""
}

// Array for list of values
func Array(typ Type) Type {
	return ArrayLike + typ
}

// IsArray checks if this type is an array
func (typ Type) IsArray() bool {
	return typ[0] == byteArray
}

// Map for an assocation of keys and values
func Map(key, value Type) Type {
	if key != String && key != Int {
		panic("Unsupported map on key type " + key.Label())
	}
	return MapLike + key + value
}

// IsMap checks if this type is a map
func (typ Type) IsMap() bool {
	return typ[0] == byteMap
}

// Resource for complex data structures
func Resource(name string) Type {
	return ResourceLike + Type(name)
}

// IsResource checks if this type is a map
func (typ Type) IsResource() bool {
	if typ.IsEmpty() {
		return false
	}
	return typ[0] == byteResource
}

// Function for functions
func Function(required rune, args []Type) Type {
	var sig string
	for _, arg := range args {
		sig += string(arg) + "\x00"
	}
	return FunctionLike + Type(required) + Type(sig)
}

// IsFunction checks if this type is a map
func (typ Type) IsFunction() bool {
	return typ[0] == byteFunction
}

// Underlying returns the basic type, e.g. types.MapLike instead of types.Map(..)
func (typ Type) Underlying() Type {
	return Type(typ[0])
}

// Enforce makes sure that both types are the same, and returns the common
// type and true if they are, false otherwise. If one of the types is not
// yet set, use the other type instead. If neither are set return the unset type.
func Enforce(left, right Type) (Type, bool) {
	if left[0] == byteUnset {
		return right, true
	}
	if left != right {
		return right, false
	}
	return right, true
}

// Child returns the child type of arrays and maps
func (typ Type) Child() Type {
	switch typ[0] {
	case byteDict:
		return Dict
	case byteArray:
		return typ[1:]
	case byteMap:
		return typ[2:]
	}
	panic("cannot determine child type of " + typ.Label())
}

// Key returns the key type of a map
func (typ Type) Key() Type {
	if typ[0] != byteMap {
		panic("cannot retrieve key type of non-map type " + typ.Label())
	}
	return Type(typ[1])
}

// ResourceName return the name of a resource
func (typ Type) ResourceName() string {
	switch typ[0] {
	case byteResource:
		return string(typ[1:])
	}
	panic("cannot determine type name of " + typ.Label())
}

var labels = map[byte]string{
	byteUnset:  "unset",
	byteAny:    "any",
	byteNil:    "null",
	byteRef:    "ref",
	byteBool:   "bool",
	byteInt:    "int",
	byteFloat:  "float",
	byteString: "string",
	byteRegex:  "regex",
	byteTime:   "time",
	byteDict:   "dict",
	byteScore:  "score",
	byteBlock:  "block",
}

var labelfun map[byte]func(Type) string

func init() {
	labelfun = map[byte]func(Type) string{
		byteArray:    func(s Type) string { return "[]" + s.Label() },
		byteMap:      func(s Type) string { return "map[" + Type(s[0]).Label() + "]" + s[1:].Label() },
		byteResource: func(s Type) string { return string(s) },
		byteFunction: func(f Type) string { return "function(..??..)" },
	}
}

// Label provides a user-friendly type label
func (typ Type) Label() string {
	if typ == "" {
		return "EMPTY"
	}

	if typ[0]&'\xf0' == '\x00' {
		h, ok := labels[typ[0]]
		if !ok {
			panic("cannot find label for simple type " + typ)
		}
		return h
	}

	h, ok := labelfun[typ[0]]
	if !ok {
		panic("cannot find label for complex type " + typ)
	}
	return h(typ[1:])
}
