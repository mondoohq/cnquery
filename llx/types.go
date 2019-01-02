package llx

import (
	"fmt"

	"go.mondoo.io/mondoo/types"
)

func (p *Primitive) typeString(typ types.Type, stack *Code) string {
	switch typ.Underlying() {
	case types.Ref:
		idx := bytes2int(p.Value)
		ref := stack.Code[idx-1]
		return ref.Primitive.typeString(ref.Type(stack), stack)
	case types.ArrayLike:
		return "[]" + p.typeString(typ.Child(), stack)
	case types.MapLike:
		return "map[" + p.typeString(typ.Key(), stack) + "]" + p.typeString(typ.Child(), stack)
	default:
		return typ.Label()
	}
}

// TypeString for the dereferenced type label of this primitive
func (p *Primitive) TypeString(stack *Code) string {
	return p.typeString(types.Type(p.Type), stack)
}

// Type for the dereferenced type of this chunk
// Finds the real type after looking at either the primitive or function
func (c *Chunk) Type(stack *Code) types.Type {
	if c.Call == Chunk_PRIMITIVE {
		return types.Type(c.Primitive.Type)
	}
	if c.Function == nil {
		return types.Any
	}
	return types.Type(c.Function.Type)
}

// turns any "ref" types into whatever they are referencing
func (p *Primitive) dereferenceType(typ types.Type, stack *Code) types.Type {
	switch typ.Underlying() {
	case types.Ref:
		idx := bytes2int(p.Value)
		fmt.Printf("ref; %d %#v", idx, stack.Code)
		ref := stack.Code[idx-1]
		return ref.Primitive.dereferenceType(ref.Type(stack), stack)
	case types.ArrayLike:
		k := p.dereferenceType(typ.Child(), stack)
		return types.Array(k)
	case types.MapLike:
		k := p.dereferenceType(typ.Key(), stack)
		v := p.dereferenceType(typ.Child(), stack)
		return types.Map(k, v)
	default:
		return typ
	}
}

// ArrayType for the given list of primitives
func ArrayType(arr []*Primitive, stack *Code) types.Type {
	if arr == nil || len(arr) == 0 {
		return types.Array(types.Any)
	}

	sharedType := arr[0].dereferenceType(types.Type(arr[0].Type), stack)
	for i := 1; i < len(arr); i++ {
		// we are done if not all elements have the same type
		ct := arr[i].dereferenceType(types.Type(arr[i].Type), stack)
		if ct != sharedType {
			return types.Array(types.Any)
		}
	}

	return types.Array(sharedType)
}
