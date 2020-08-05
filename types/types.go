package types

// Type information
type Type string

// Unspecified indicates that this type has not been specified
const Unspecified Type = ""

const (
	byteAny byte = 1 + iota
	byteNil
	byteRef
	byteBool
	byteInt
	byteFloat
	byteString
	byteRegex
	byteTime
	byteDict
	byteArray = 1<<4 + iota - 2 // set to 24 to avoid breaking changes
	byteMap
	byteResource
	byteFunction
)

const (
	// Any type indicating an untyped value that can have any type
	Any = Type(byteAny)
	// Nil for the empty type
	Nil = Type(byteNil)
	// Ref for internal code chunk references
	Ref = Type(byteRef)
	// Bool for the booleans true and false
	Bool = Type(byteBool)
	// Int for integers
	Int = Type(byteInt)
	// Float for any decimal values
	Float = Type(byteFloat)
	// String for strings
	String = Type(byteString)
	// Regex for regular expressions
	Regex = Type(byteRegex)
	// Time for date and time
	Time = Type(byteTime)
	// Dict for storing hierarchical simple key-value assignemnts
	Dict = Type(byteDict)
	// ArrayLike is the underlying type of all arrays
	ArrayLike = Type(byteArray)
	// MapLike is the underlying type of all arrays
	MapLike = Type(byteMap)
	// ResourceLike is the underlying type of all arrays
	ResourceLike = Type(byteResource)
	// FunctionLike is the underlying type of all arrays
	FunctionLike = Type(byteFunction)
)

// IsEmpty returns true if the type has no information
func (typ Type) IsEmpty() bool {
	return typ == ""
}

// Array for list of values
func Array(typ Type) Type {
	return Type(byteArray) + typ
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
	return Type(byteMap) + key + value
}

// IsMap checks if this type is a map
func (typ Type) IsMap() bool {
	return typ[0] == byteMap
}

// Resource for complex data structures
func Resource(name string) Type {
	return Type(byteResource) + Type(name)
}

// IsResource checks if this type is a map
func (typ Type) IsResource() bool {
	return typ[0] == byteResource
}

// Function for functions
func Function(required rune, args []Type) Type {
	var sig string
	for _, arg := range args {
		sig += string(arg) + "\x00"
	}
	return Type(byteFunction) + Type(required) + Type(sig)
}

// IsFunction checks if this type is a map
func (typ Type) IsFunction() bool {
	return typ[0] == byteFunction
}

// Underlying returns the basic type, e.g. types.MapLike instead of types.Map(..)
func (typ Type) Underlying() Type {
	return Type(typ[0])
}

// Child returns the child type of arrays and maps
func (typ Type) Child() Type {
	switch typ[0] {
	case byteArray:
		return typ[1:]
	case byteMap:
		return typ[2:]
	}
	panic("Cannot determine child type of " + typ.Label())
}

// Key returns the key type of a map
func (typ Type) Key() Type {
	if typ[0] != byteMap {
		panic("Cannot retrieve key type of non-map type " + typ.Label())
	}
	return Type(typ[1])
}

// Name return the name of a resource
func (typ Type) Name() string {
	switch typ[0] {
	case byteResource:
		return string(typ[1:])
	}
	panic("Cannot determine type name of " + typ.Label())
}

var labels = map[byte]string{
	byteAny:    "any",
	byteNil:    "nil",
	byteRef:    "ref",
	byteBool:   "bool",
	byteInt:    "int",
	byteFloat:  "float",
	byteString: "string",
	byteRegex:  "regex",
	byteTime:   "time",
	byteDict:   "dict",
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
			panic("Cannot find label for simple type " + typ)
		}
		return h
	}

	h, ok := labelfun[typ[0]]
	if !ok {
		panic("Cannot find label for complex type " + typ)
	}
	return h(typ[1:])
}
