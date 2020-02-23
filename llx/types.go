package llx

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"

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

// Checksum computes the checksum of this chunk
func (c *Chunk) Checksum(checksums map[int32]string) string {
	data := []byte(c.Id)

	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(c.Call))
	data = append(data, b...)

	if c.Primitive != nil {
		cs := c.Primitive.checksum(checksums)
		data = append(data, cs...)
	}

	if c.Function != nil {
		data = append(data, c.Function.checksum(checksums)...)
	}

	hash := blake2b.Sum512(data)
	return base64.StdEncoding.EncodeToString(hash[:])
}

func (f *Function) checksum(checksums map[int32]string) []byte {
	res := []byte(f.Type)

	if f.Binding != 0 {
		ref := checksums[f.Binding]
		if ref == "" {
			panic("Cannot compute checksum for chunk, it doesn't seem to reference a function on the stack")
		}
		res = append(res, ref...)
	}

	for i := range f.Args {
		cs := f.Args[i].checksum(checksums)
		res = append(res, cs...)
	}

	return res
}

func (p *Primitive) checksum(checksums map[int32]string) []byte {
	ref, ok := p.Ref()
	if ok {
		refChecksum, ok := checksums[int32(ref)]
		if !ok {
			panic("Cannot compute checksum for primitive, it doesn't seem to reference a function on the stack")
		}

		return []byte(refChecksum)
	}

	res := []byte(p.Type)
	res = append(res, p.Value...)

	for i := range p.Array {
		entry := p.Array[i]
		res = append(res, entry.checksum(checksums)...)
	}

	for k, v := range p.Map {
		res = append(res, k...)
		res = append(res, v.checksum(checksums)...)
	}

	return res
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
