package llx

import (
	"go.mondoo.io/mondoo/checksums"
	"go.mondoo.io/mondoo/types"
)

// AddChunk to the list of chunks
func (l *Code) AddChunk(c *Chunk) {
	l.Checksums[l.ChunkIndex()+1] = c.Checksum(l)
	l.Code = append(l.Code, c)
}

// RefreshChunkChecksum if something changed about it
func (l *Code) RefreshChunkChecksum(c *Chunk) {
	var ref int32 = -1

	for i := len(l.Code) - 1; i >= 0; i-- {
		if l.Code[i] == c {
			ref = int32(i)
			break
		}
	}

	if ref != -1 {
		l.Checksums[ref+1] = c.Checksum(l)
	}
}

// RemoveLastChunk from the current code
func (l *Code) RemoveLastChunk() {
	l.Code = l.Code[:len(l.Code)-1]
}

// ChunkIndex is the index of the last chunk that was added
func (l *Code) ChunkIndex() int32 {
	return int32(len(l.Code))
}

func (l *Code) FunctionsIndex() int32 {
	return int32(len(l.Functions))
}

// LastChunk is the last chunk in the list or nil
func (l *Code) LastChunk() *Chunk {
	tl := len(l.Code)
	if tl == 0 {
		return nil
	}
	return l.Code[tl-1]
}

// checksum from this code
func (l *Code) checksum() string {
	checksum := checksums.New
	for i := range l.Entrypoints {
		checksum = checksum.Add(l.Checksums[l.Entrypoints[i]])
	}
	if len(l.Entrypoints) == 0 {
		checksum = checksum.Add(l.Checksums[l.ChunkIndex()])
	}
	return checksum.String()
}

// UpdateID of the piece of code
func (l *Code) UpdateID() {
	l.Id = l.checksum()
}

// AddSuggestion to CodeBundle
func (l *CodeBundle) AddSuggestion(msg string) {
	l.Suggestions = append(l.Suggestions, msg)
}

var comparableOperations = map[string]struct{}{
	"==": {},
	"!=": {},
	">":  {},
	"<":  {},
	">=": {},
	"<=": {},
	"&&": {},
	"||": {},
}

func (c *Chunk) isStatic() bool {
	if c.Call != Chunk_PRIMITIVE {
		return false
	}

	if types.Type(c.Primitive.Type) == types.Ref {
		return false
	}

	return true
}

// RefDatapoints returns the additional datapoints that inform a ref.
// Typically used when writing tests and providing additional data when the test fails.
func (l *Code) RefDatapoints(ref int32) []int32 {
	chunk := l.Code[ref-1]

	if chunk.Id == "if" && chunk.Function != nil && len(chunk.Function.Args) != 0 {
		var ok bool
		ref, ok = chunk.Function.Args[0].Ref()
		if !ok {
			return nil
		}
		chunk = l.Code[ref-1]
	}

	if chunk.Id == "" {
		return nil
	}

	// nothing to do for primitives (unclear if we need to investigate refs here)
	if chunk.Call != Chunk_FUNCTION || chunk.Function == nil {
		return nil
	}

	if _, ok := comparableOperations[chunk.Id[0:1]]; !ok {
		if len(chunk.Id) == 1 {
			return nil
		}
		if _, ok := comparableOperations[chunk.Id[0:2]]; !ok {
			return nil
		}
	}

	var res []int32

	// at this point we have a comparable
	// so 2 jobs: check the left, check the right. if it's static, ignore. if not, add
	left := chunk.Function.Binding
	if left != 0 {
		leftChunk := l.Code[left-1]
		if leftChunk != nil && !leftChunk.isStatic() {
			res = append(res, left)
		}
	}

	if len(chunk.Function.Args) != 0 {
		rightPrim := chunk.Function.Args[0]
		if rightPrim != nil && types.Type(rightPrim.Type) == types.Ref {
			right, ok := rightPrim.Ref()
			if ok {
				res = append(res, right)
			}
		}
	}

	return res
}
