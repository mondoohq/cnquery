package llx

import (
	"encoding/base64"
	"encoding/binary"

	"go.mondoo.io/mondoo/types"
	"golang.org/x/crypto/blake2b"
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

// Type of this chunk, by looking at either the primitive or function.
// The type is not be dereferenced! (i.e. ref's remain)
func (c *Chunk) Type(stack *Code) types.Type {
	// call: primitive
	if c.Call == Chunk_PRIMITIVE {
		return types.Type(c.Primitive.Type)
	}

	// call: function
	if c.Function != nil {
		return types.Type(c.Function.Type)
	}

	// call: property
	if c.Primitive != nil {
		return types.Type(c.Primitive.Type)
	}

	return types.Any
}

// DereferencedType of this chunk, resolved if it is a reference to anything.
func (c *Chunk) DereferencedType(stack *Code) types.Type {
	if c.Call == Chunk_PRIMITIVE {
		return c.Primitive.dereferenceType(types.Type(c.Primitive.Type), stack)
	}

	if c.Function != nil {
		return types.Type(c.Function.Type)
	}

	if c.Primitive != nil {
		return c.Primitive.dereferenceType(types.Type(c.Primitive.Type), stack)
	}

	return types.Any
}

// Checksum computes the checksum of this chunk
func (c *Chunk) Checksum(code *Code) string {
	data := []byte(c.Id)

	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(c.Call))
	data = append(data, b...)

	if c.Primitive != nil {
		cs := c.Primitive.checksum(code)
		data = append(data, cs...)
	}

	if c.Function != nil {
		data = append(data, c.Function.checksum(code)...)
	}

	hash := blake2b.Sum512(data)
	return base64.StdEncoding.EncodeToString(hash[:])
}

func (f *Function) checksum(code *Code) []byte {
	res := []byte(f.Type)

	if f.Binding != 0 {
		ref := code.Checksums[f.Binding]
		if ref == "" {
			panic("cannot compute checksum for chunk, it doesn't seem to reference a function on the stack")
		}
		res = append(res, ref...)
	}

	for i := range f.Args {
		cs := f.Args[i].checksum(code)
		res = append(res, cs...)
	}

	return res
}

func (p *Primitive) checksum(code *Code) []byte {
	ref, ok := p.Ref()
	if ok {
		typ := types.Type(p.Type)
		if typ == types.Ref {
			refChecksum, ok := code.Checksums[int32(ref)]
			if !ok {
				panic("llx> cannot compute checksum for primitive, it doesn't seem to reference a variable on the stack")
			}

			return []byte(refChecksum)
		}

		if typ.Underlying() == types.FunctionLike {
			refFunction := code.Functions[int32(ref)-1]
			if !ok {
				panic("llx> cannot compute checksum for primitive, it doesn't seem to reference a function on the stack")
			}

			return []byte(refFunction.Id)
		}

		panic("llx> received a reference of an unknown type in trying to calculate the checksum")
	}

	res := []byte(p.Type)
	res = append(res, p.Value...)

	for i := range p.Array {
		entry := p.Array[i]
		res = append(res, entry.checksum(code)...)
	}

	for k, v := range p.Map {
		res = append(res, k...)
		res = append(res, v.checksum(code)...)
	}

	return res
}

// turns any "ref" types into whatever they are referencing
func (p *Primitive) dereferenceType(typ types.Type, stack *Code) types.Type {
	switch typ.Underlying() {
	case types.Ref:
		idx := bytes2int(p.Value)
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
		return types.Array(types.Unset)
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
