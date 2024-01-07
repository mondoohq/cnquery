// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package types

import (
	"encoding/json"
	"strings"
	"time"
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
	byteEmpty
	byteSemver
	byteArray = 1<<4 + iota - 6 // set to 25 to avoid breaking changes
	byteMap
	byteResource
	byteFunction
	byteStringSlice
	byteRange
)

// NoType type is one whose type information is not available at all
const NoType Type = ""

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
	// Dict for storing hierarchical simple key-value assignments
	Dict = Type(rune(byteDict))
	// Score for evaluations
	Score = Type(rune(byteScore))
	// Block evaluation results
	Block = Type(rune(byteBlock))
	// Empty value
	Empty = Type(rune(byteEmpty))
	// Semver value
	Semver = Type(rune(byteSemver))
	// ArrayLike is the underlying type of all arrays
	ArrayLike = Type(rune(byteArray))
	// MapLike is the underlying type of all maps
	MapLike = Type(rune(byteMap))
	// ResourceLike is the underlying type of all resources
	ResourceLike = Type(rune(byteResource))
	// FunctionLike is the underlying type of all functions
	FunctionLike = Type(rune(byteFunction))

	// StringSlice is used to represent special function for searching strings.
	// Users are never exposed to this type directly and it is not documented
	// as a primitive. It serves as a way to array functions on top of strings,
	// which is required for cases where a `dict` can represent both an array
	// and a string (as well as other things) at the same time. Functions like
	// `contains` are defined on both arrays and strings with different behavior.
	// This types allows us to keep the compiler and execution simple, while
	// handling the runtime distinction for dict.
	StringSlice = Type(rune(byteStringSlice))

	// Range represents a range of content. This can be a number of lines
	// or lines and columns combined. We use a special type for a very
	// efficient storage and transmission structure.
	Range = Type(rune(byteRange))
)

// NotSet returns true if the type has no information
func (typ Type) NotSet() bool {
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

// Map for an association of keys and values
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
	if typ.NotSet() {
		return false
	}
	return typ[0] == byteResource
}

// ContainsResource checks if this or any child type has a resource
func (typ Type) ContainsResource() bool {
	for {
		if typ.IsResource() {
			return true
		}

		if !typ.IsArray() && !typ.IsMap() {
			return false
		}

		typ = typ.Child()
	}
}

// Function for creating a function type signature
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
// type and true if they are, false otherwise (and the right type).
// - if one of the types is not yet set, use the other type instead.
// - if neither are set return the unset type.
// - goes into child types to see if either is unset
func Enforce(left, right Type) (Type, bool) {
	var i int
	for ; i < len(left) && i < len(right); i++ {
		if left[i] == right[i] {
			continue
		}

		if right[i] == byteUnset || right[i] == byteNil {
			return left, true
		}
		if left[i] == byteUnset || left[i] == byteNil {
			return right, true
		}
	}

	return right, len(left) == len(right)
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

// ResourceName return the name of a resource. Has to be a resource type,
// otherwise this call panics.
func (typ Type) ResourceName() string {
	if typ[0] == byteResource {
		return string(typ[1:])
	}
	panic("cannot determine type name of " + typ.Label())
}

var labels = map[byte]string{
	byteUnset:       "unset",
	byteAny:         "any",
	byteNil:         "null",
	byteRef:         "ref",
	byteBool:        "bool",
	byteInt:         "int",
	byteFloat:       "float",
	byteString:      "string",
	byteRegex:       "regex",
	byteTime:        "time",
	byteDict:        "dict",
	byteScore:       "score",
	byteBlock:       "block",
	byteEmpty:       "empty",
	byteSemver:      "semver",
	byteStringSlice: "stringslice",
	byteRange:       "range",
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

	h, ok := labels[typ[0]]
	if ok {
		return h
	}

	hf, ok := labelfun[typ[0]]
	if !ok {
		panic("cannot find label for type " + typ)
	}
	return hf(typ[1:])
}

// Equal provides a set of function for a range of types to test if 2 values
// of that type are equal
var Equal = map[Type]func(interface{}, interface{}) bool{
	Bool: func(left, right interface{}) bool {
		return left.(bool) == right.(bool)
	},
	Int: func(left, right interface{}) bool {
		return left.(int64) == right.(int64)
	},
	Float: func(left, right interface{}) bool {
		return left.(float64) == right.(float64)
	},
	String: func(left, right interface{}) bool {
		return left.(string) == right.(string)
	},
	Regex: func(left, right interface{}) bool {
		return left.(string) == right.(string)
	},
	Time: func(left, right interface{}) bool {
		l := left.(*time.Time)
		r := right.(*time.Time)
		if l == nil || r == nil {
			return false
		}
		return l.Equal(*r)
	},
	// types.Dict: func(left, right interface{}) bool {},
	Score: func(left, right interface{}) bool {
		return left.(int32) == right.(int32)
	},
}
