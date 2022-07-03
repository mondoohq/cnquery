package llx

import (
	"sort"

	"go.mondoo.io/mondoo/checksums"
	"go.mondoo.io/mondoo/types"
)

func (x *CodeBundle) IsV2() bool {
	return x.CodeV2 != nil
}

func (x *CodeBundle) FilterResults(results map[string]*RawResult) map[string]*RawResult {
	filteredResults := map[string]*RawResult{}

	if x.IsV2() {
		for i := range x.CodeV2.Checksums {
			checksum := x.CodeV2.Checksums[i]

			res := results[checksum]
			if res != nil {
				filteredResults[checksum] = res
			}
		}
	} else {
		for i := range x.DeprecatedV5Code.Checksums {
			checksum := x.DeprecatedV5Code.Checksums[i]

			res := results[checksum]
			if res != nil {
				filteredResults[checksum] = res
			}
		}
	}

	return filteredResults
}

func (b *Block) ChunkIndex() uint32 {
	return uint32(len(b.Chunks))
}

func absRef(blockRef uint64, relRef uint32) uint64 {
	return (blockRef & 0xFFFFFFFF00000000) | uint64(relRef)
}

func (b *Block) TailRef(blockRef uint64) uint64 {
	return absRef(blockRef, b.ChunkIndex())
}

// LastChunk is the last chunk in the list or nil
func (b *Block) LastChunk() *Chunk {
	max := len(b.Chunks)
	if max == 0 {
		return nil
	}
	return b.Chunks[max-1]
}

// AddChunk to the list of chunks
func (b *Block) AddChunk(code *CodeV2, blockRef uint64, c *Chunk) {
	nuRef := b.TailRef(blockRef) + 1
	code.Checksums[nuRef] = c.ChecksumV2(blockRef, code)
	b.Chunks = append(b.Chunks, c)
}

func (b *Block) AddArgumentPlaceholder(code *CodeV2, blockRef uint64, typ types.Type, checksum string) {
	b.AddChunk(code, blockRef, &Chunk{
		Call:      Chunk_PRIMITIVE,
		Primitive: &Primitive{Type: string(typ)},
	})
	code.Checksums[b.TailRef(blockRef)] = checksum
	b.Parameters++
}

// PopChunk removes the last chunk from the block and returns it
func (b *Block) PopChunk() *Chunk {
	if len(b.Chunks) == 0 {
		return nil
	}

	max := len(b.Chunks)
	last := b.Chunks[max-1]
	b.Chunks = b.Chunks[:max-1]
	return last
}

// ChunkIndex is the index of the last chunk that was added
func (l *CodeV2) TailRef(blockRef uint64) uint64 {
	return l.Block(blockRef).TailRef(blockRef)
}

// Retrieve a chunk for the given ref
func (l *CodeV2) Chunk(ref uint64) *Chunk {
	return l.Block(ref).Chunks[uint32(ref)-1]
}

// Retrieve a block for the given ref
func (l *CodeV2) Block(ref uint64) *Block {
	return l.Blocks[uint32(ref>>32)-1]
}

// AddBlock adds a new block at the end of this code and returns its ref
func (l *CodeV2) AddBlock() (*Block, uint64) {
	block := &Block{}
	l.Blocks = append(l.Blocks, block)
	return block, uint64(len(l.Blocks)) << 32
}

func (c *CodeV2) Entrypoints() []uint64 {
	if len(c.Blocks) == 0 {
		return []uint64{}
	}

	return c.Blocks[0].Entrypoints
}

func (c *CodeV2) Datapoints() []uint64 {
	if len(c.Blocks) == 0 {
		return []uint64{}
	}

	return c.Blocks[0].Datapoints
}

// DereferencedBlockType returns the type of a block, which is a specific
// type if it is a single-value block
func (l *CodeV2) DereferencedBlockType(b *Block) types.Type {
	if len(b.Entrypoints) != 1 {
		return types.Block
	}

	ep := b.Entrypoints[0]
	chunk := b.Chunks[(ep-1)&0xFFFFFFFF]
	return chunk.DereferencedTypeV2(l)
}

func (block *Block) checksum(l *CodeV2) string {
	c := checksums.New
	for i := range block.Entrypoints {
		c = c.Add(l.Checksums[block.Entrypoints[i]])
	}
	return c.String()
}

// checksum from this code
func (l *CodeV2) checksum() string {
	checksum := checksums.New

	for i := range l.Blocks {
		checksum = checksum.Add(l.Blocks[i].checksum(l))
	}

	assertionRefs := make([]uint64, 0, len(l.Assertions))
	for k := range l.Assertions {
		assertionRefs = append(assertionRefs, k)
	}
	sort.Slice(assertionRefs, func(i, j int) bool { return assertionRefs[i] < assertionRefs[j] })
	for _, ref := range assertionRefs {
		checksum = checksum.Add(l.Checksums[ref])
	}

	if len(l.Blocks) == 0 || len(l.Blocks[0].Entrypoints) == 0 {
		// Why do we even handle this case? Because at this point we still have
		// all raw entrypoints, which may get shuffled around in the step after this.
		// This also means entrypoints aren't sanitized. We may not have any.
		//
		// TODO: review this behavior!
		// We may want to do the entrypoint handling earlier.
		//panic("received a code without any entrypoints")
	}

	return checksum.String()
}

// UpdateID of the piece of code
func (l *CodeV2) UpdateID() {
	l.Id = l.checksum()
}

// RefDatapoints returns the additional datapoints that inform a ref.
// Typically used when writing tests and providing additional data when the test fails.
func (l *CodeV2) RefDatapoints(ref uint64) []uint64 {
	if assertion, ok := l.Assertions[ref]; ok {
		return assertion.Refs
	}

	chunk := l.Chunk(ref)

	if chunk.Id == "if" && chunk.Function != nil && len(chunk.Function.Args) != 0 {
		var ok bool
		ref, ok = chunk.Function.Args[0].RefV2()
		if !ok {
			return nil
		}
		chunk = l.Chunk(ref)
	}

	if chunk.Id == "" {
		return nil
	}

	// nothing to do for primitives (unclear if we need to investigate refs here)
	if chunk.Call != Chunk_FUNCTION || chunk.Function == nil {
		return nil
	}

	switch chunk.Id {
	case "$all", "$one", "$any", "$none":
		return []uint64{ref - 1}
	}

	if _, ok := ComparableLabel(chunk.Id); !ok {
		return nil
	}

	var res []uint64

	// at this point we have a comparable
	// so 2 jobs: check the left, check the right. if it's static, ignore. if not, add
	left := chunk.Function.Binding
	if left != 0 {
		leftChunk := l.Chunk(left)
		if leftChunk != nil && !leftChunk.isStatic() {
			res = append(res, left)
		}
	}

	if len(chunk.Function.Args) != 0 {
		rightPrim := chunk.Function.Args[0]
		if rightPrim != nil && types.Type(rightPrim.Type) == types.Ref {
			right, ok := rightPrim.RefV2()
			if ok {
				res = append(res, right)
			}
		}
	}

	return res
}

func (l *CodeV2) refValues(bundle *CodeBundle, ref uint64, lookup func(s string) (*RawResult, bool)) []*RawResult {
	checksum := l.Checksums[ref]
	checksumRes, ok := lookup(checksum)
	if ok {
		return []*RawResult{checksumRes}
	}

	chunk := l.Chunk(ref)

	if chunk.Id == "if" && chunk.Function != nil && len(chunk.Function.Args) != 0 {
		// FIXME: we should be checking for the result of the if-condition and then proceed
		// with whatever result is applicable; not poke at possible results

		// function arguments are functions refs to:
		// [1] = the first condition, [2] = the second condition
		fref, ok := chunk.Function.Args[1].RefV2()
		if ok {
			if part, ok := lookup(l.Checksums[fref]); ok {
				return []*RawResult{part}
			}
		}

		fref, ok = chunk.Function.Args[2].RefV2()
		if ok {
			if part, ok := lookup(l.Checksums[fref]); ok {
				return []*RawResult{part}
			}
		}
	}

	return nil
}

func (l *CodeV2) returnValues(bundle *CodeBundle, lookup func(s string) (*RawResult, bool)) []*RawResult {
	var res []*RawResult

	if len(l.Blocks) == 0 {
		return res
	}
	block := l.Blocks[0]

	for i := range block.Entrypoints {
		ep := block.Entrypoints[i]
		cur := l.refValues(bundle, ep, lookup)
		if cur != nil {
			res = append(res, cur...)
		}
	}

	return res
}

func ReturnValuesV2(bundle *CodeBundle, f func(s string) (*RawResult, bool)) []*RawResult {
	return bundle.CodeV2.returnValues(bundle, f)
}

// Results2Assessment converts a list of raw results into an assessment for the query
func Results2AssessmentV2(bundle *CodeBundle, results map[string]*RawResult) *Assessment {
	return Results2AssessmentLookupV2(bundle, func(s string) (*RawResult, bool) {
		r := results[s]
		return r, r != nil
	})
}

// Results2AssessmentLookup creates an assessment for a bundle using a lookup hook to get all results
func Results2AssessmentLookupV2(bundle *CodeBundle, f func(s string) (*RawResult, bool)) *Assessment {
	code := bundle.CodeV2

	res := Assessment{
		Success:  true,
		Checksum: code.Id,
	}
	res.Success = true

	entrypoints := code.Entrypoints()
	for i := range entrypoints {
		ep := entrypoints[i]
		cur := code.entrypoint2assessment(bundle, ep, f)
		if cur == nil {
			continue
		}

		res.Results = append(res.Results, cur)
		if !cur.Success {
			res.Success = false
		}

		// We don't want to lose errors
		if cur.IsAssertion || cur.Error != "" {
			res.IsAssertion = true
		}
	}

	if !res.IsAssertion {
		return nil
	}

	return &res
}

func (l *CodeV2) entrypoint2assessment(bundle *CodeBundle, ref uint64, lookup func(s string) (*RawResult, bool)) *AssessmentItem {
	code := bundle.CodeV2
	checksum := code.Checksums[ref]

	checksumRes, ok := lookup(checksum)
	if !ok {
		return nil
	}

	truthy, _ := checksumRes.Data.IsTruthy()

	res := AssessmentItem{
		Checksum:   checksum,
		Entrypoint: ref,
		Success:    truthy,
	}

	if checksumRes.Data.Error != nil {
		res.Error = checksumRes.Data.Error.Error()
	}

	// explicit assessments
	if assertion, ok := bundle.Assertions[checksum]; ok {
		res.IsAssertion = true

		if assertion.DecodeBlock {
			sum := assertion.Checksums[0]
			raw, ok := lookup(sum)
			if !ok {
				res.Error = "cannot find required data block for assessment"
				return &res
			}

			x := raw.Result().Data
			if x == nil {
				res.Error = "required data block for assessment is nil"
				return &res
			}

			dataMap := map[string]*Primitive(x.Map)

			cnt := len(assertion.Checksums) - 1
			res.Data = make([]*Primitive, cnt)
			for i := 0; i < cnt; i++ {
				sum = assertion.Checksums[i+1]
				res.Data[i], ok = dataMap[sum]
				if !ok {
					res.Error = "required data field is not in block for assessment"
				}
			}

			res.Template = assertion.Template
			return &res
		}

		data := make([]*Primitive, len(assertion.Checksums))
		for j := range assertion.Checksums {
			sum := assertion.Checksums[j]

			raw, ok := lookup(sum)
			if !ok {
				res.Error = "cannot find required data"
				return &res
			}

			data[j] = raw.Result().Data
		}

		res.Data = data
		res.Template = assertion.Template
		return &res
	}

	chunk := l.Chunk(ref)

	if chunk.Id == "if" {
		// Our current assessment structure cannot handle nesting very well
		// We return nil here for now. Our result printing has good enough
		// information to convey this nesting and what exactly went wrong
		return nil
	}

	if chunk.Call == Chunk_PRIMITIVE {
		res.Actual = chunk.Primitive
		return &res
	}

	if chunk.Call != Chunk_FUNCTION {
		res.Error = "unknown type of chunk"
		return &res
	}

	if chunk.Function == nil {
		// this only happens when we have a call chain that resembles a resource
		// which is used without any init arguments
		chunk.Function = &Function{Type: string(types.Resource(chunk.Id))}
	}

	if chunk.Id == "" {
		res.Error = "chunk has unknown identifier"
		return &res
	}

	switch chunk.Id {
	case "$one", "$all", "$none", "$any":
		res.IsAssertion = true
		res.Operation = chunk.Id[1:]

		if !truthy {
			listRef := chunk.Function.Binding
			list, ok := lookup(code.Checksums[listRef])
			if !ok {
				res.Error = "cannot find value for assessment (" + res.Operation + ")"
				return &res
			}

			res.Actual = list.Result().Data
		} else {
			res.Actual = BoolPrimitive(true)
		}

		return &res
	}

	// FIXME: support child operations inside of block calls "{}" / "${}"

	if label, found := ComparableLabel(chunk.Id); found {
		res.Operation = label
	} else {
		cRes := checksumRes.Result()

		if checksumRes.Data.Type != types.Bool {
			res.Actual = cRes.Data
		} else {
			res.Operation = "=="
			res.Expected = BoolPrimitive(true)
			res.Actual = cRes.Data
			res.IsAssertion = true
		}
		return &res
	}

	res.IsAssertion = true

	// at this point we have a comparable
	// so 2 jobs: check the left, check the right. if it's static, ignore. if not, add
	left := chunk.Function.Binding
	if left != 0 {
		leftChunk := l.Chunk(left)
		if leftChunk == nil {
			res.Actual = &Primitive{
				Type:  string(types.Any),
				Value: []byte("< unknown expected value >"),
			}
		}

		if leftChunk.isStatic() {
			res.Actual = leftChunk.Primitive
		} else {
			leftSum := code.Checksums[left]
			leftRes, ok := lookup(leftSum)
			if !ok {
				res.Actual = nil
			} else {
				res.Actual = leftRes.Result().Data
			}
		}
	}

	if len(chunk.Function.Args) == 0 {
		return &res
	}

	rightPrim := chunk.Function.Args[0]
	if rightPrim == nil {
		res.Expected = &Primitive{
			Type:  string(types.Any),
			Value: []byte("< unknown actual value >"),
		}
	}

	if types.Type(rightPrim.Type) != types.Ref {
		res.Expected = rightPrim
	} else {
		right, ok := rightPrim.RefV2()
		if !ok {
			res.Expected = &Primitive{
				Type:  string(types.Any),
				Value: []byte("< unknown actual value >"),
			}
		} else {
			rightSum := code.Checksums[right]
			rightRes, ok := lookup(rightSum)
			if !ok {
				res.Expected = nil
			} else {
				res.Expected = rightRes.Result().Data
			}
		}
	}

	return &res
}

func Results2Assessment(bundle *CodeBundle, results map[string]*RawResult, useV2Code bool) *Assessment {
	if useV2Code {
		return Results2AssessmentLookupV2(bundle, func(s string) (*RawResult, bool) {
			r := results[s]
			return r, r != nil
		})
	}
	return Results2AssessmentLookupV1(bundle, func(s string) (*RawResult, bool) {
		r := results[s]
		return r, r != nil
	})
}
