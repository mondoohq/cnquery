package llx

import (
	"go.mondoo.io/mondoo/checksums"
	"go.mondoo.io/mondoo/types"
)

// AddChunk to the list of chunks
func (l *CodeV1) AddChunk(c *Chunk) {
	l.Checksums[l.ChunkIndex()+1] = c.ChecksumV1(l)
	l.Code = append(l.Code, c)
}

// RefreshChunkChecksum if something changed about it
func (l *CodeV1) RefreshChunkChecksum(c *Chunk) {
	var ref int32 = -1

	for i := len(l.Code) - 1; i >= 0; i-- {
		if l.Code[i] == c {
			ref = int32(i)
			break
		}
	}

	if ref != -1 {
		l.Checksums[ref+1] = c.ChecksumV1(l)
	}
}

// RemoveLastChunk from the current code
func (l *CodeV1) RemoveLastChunk() {
	l.Code = l.Code[:len(l.Code)-1]
}

// ChunkIndex is the index of the last chunk that was added
func (l *CodeV1) ChunkIndex() int32 {
	return int32(len(l.Code))
}

func (l *CodeV1) FunctionsIndex() int32 {
	return int32(len(l.Functions))
}

// LastChunk is the last chunk in the list or nil
func (l *CodeV1) LastChunk() *Chunk {
	tl := len(l.Code)
	if tl == 0 {
		return nil
	}
	return l.Code[tl-1]
}

// checksum from this code
func (l *CodeV1) checksum() string {
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
func (l *CodeV1) UpdateID() {
	l.Id = l.checksum()
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

// ComparableLabel takes any arbitrary label and returns the
// operation as a printable string and true if it is a comparable, otherwise "" and false.
func ComparableLabel(label string) (string, bool) {
	if label == "" {
		return "", false
	}

	x := label[0:1]
	if _, ok := comparableOperations[x]; ok {
		return x, true
	}

	x = label[0:2]
	if _, ok := comparableOperations[x]; ok {
		return x, true
	}

	return "", false
}

// RefDatapoints returns the additional datapoints that inform a ref.
// Typically used when writing tests and providing additional data when the test fails.
func (l *CodeV1) RefDatapoints(ref int32) []int32 {
	if assertion, ok := l.Assertions[ref]; ok {
		return assertion.DeprecatedV5Datapoint
	}

	chunk := l.Code[ref-1]

	if chunk.Id == "if" && chunk.Function != nil && len(chunk.Function.Args) != 0 {
		var ok bool
		ref, ok = chunk.Function.Args[0].RefV1()
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

	switch chunk.Id {
	case "$all", "$one", "$any", "$none":
		return []int32{ref - 1}
	}

	if _, ok := ComparableLabel(chunk.Id); !ok {
		return nil
	}

	var res []int32

	// at this point we have a comparable
	// so 2 jobs: check the left, check the right. if it's static, ignore. if not, add
	left := chunk.Function.DeprecatedV5Binding
	if left != 0 {
		leftChunk := l.Code[left-1]
		if leftChunk != nil && !leftChunk.isStatic() {
			res = append(res, left)
		}
	}

	if len(chunk.Function.Args) != 0 {
		rightPrim := chunk.Function.Args[0]
		if rightPrim != nil && types.Type(rightPrim.Type) == types.Ref {
			right, ok := rightPrim.RefV1()
			if ok {
				res = append(res, right)
			}
		}
	}

	return res
}

func (l *CodeV1) entrypoint2assessment(bundle *CodeBundle, ref int32, lookup func(s string) (*RawResult, bool)) *AssessmentItem {
	code := bundle.DeprecatedV5Code
	checksum := code.Checksums[ref]

	checksumRes, ok := lookup(checksum)
	if !ok {
		return nil
	}

	truthy, _ := checksumRes.Data.IsTruthy()

	res := AssessmentItem{
		Checksum:   checksum,
		Entrypoint: uint64(ref), // TODO(jaym): if ref is negative, this will will be wrong
		Success:    truthy,
	}

	if checksumRes.Data.Error != nil {
		res.Error = checksumRes.Data.Error.Error()
	}

	// explicit assessments
	if assertion, ok := bundle.DeprecatedV5Assertions[checksum]; ok {
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

	chunk := l.Code[ref-1]

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
		res.Error = "chunk function cannot be nil"
		return &res
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
			listRef := chunk.Function.DeprecatedV5Binding
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
	left := chunk.Function.DeprecatedV5Binding
	if left != 0 {
		leftChunk := l.Code[left-1]
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
		right, ok := rightPrim.RefV1()
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

// Results2AssessmentV1 converts a list of raw results into an assessment for the query
func Results2AssessmentV1(bundle *CodeBundle, results map[string]*RawResult) *Assessment {
	return Results2AssessmentLookupV1(bundle, func(s string) (*RawResult, bool) {
		r := results[s]
		return r, r != nil
	})
}

// Results2AssessmentLookupV1 creates an assessment for a bundle using a lookup hook to get all results
func Results2AssessmentLookupV1(bundle *CodeBundle, f func(s string) (*RawResult, bool)) *Assessment {
	code := bundle.DeprecatedV5Code

	res := Assessment{
		Success:  true,
		Checksum: code.Id,
	}
	res.Success = true

	for i := range code.Entrypoints {
		ep := code.Entrypoints[i]
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

func (l *CodeV1) refValues(bundle *CodeBundle, ref int32, lookup func(s string) (*RawResult, bool)) []*RawResult {
	checksum := l.Checksums[ref]
	checksumRes, ok := lookup(checksum)
	if ok {
		return []*RawResult{checksumRes}
	}

	chunk := l.Code[ref-1]

	if chunk.Id == "if" && chunk.Function != nil && len(chunk.Function.Args) != 0 {
		// FIXME: we should be checking for the result of the if-condition and then proceed
		// with whatever result is applicable; not poke at possible results

		// function arguments are functions refs to:
		// [1] = the first conditino, [2] = the second condition
		fref, ok := chunk.Function.Args[1].RefV1()
		if ok {
			fun := l.Functions[fref-1]
			part := fun.returnValues(bundle, lookup)
			if len(part) != 0 {
				return part
			}
		}

		fref, ok = chunk.Function.Args[2].RefV1()
		if ok {
			fun := l.Functions[fref-1]
			part := fun.returnValues(bundle, lookup)
			if len(part) != 0 {
				return part
			}
		}
	}

	return nil
}

func (l *CodeV1) returnValues(bundle *CodeBundle, lookup func(s string) (*RawResult, bool)) []*RawResult {
	var res []*RawResult

	for i := range l.Entrypoints {
		ep := l.Entrypoints[i]
		cur := l.refValues(bundle, ep, lookup)
		if cur != nil {
			res = append(res, cur...)
		}
	}

	return res
}

func ReturnValuesV1(bundle *CodeBundle, f func(s string) (*RawResult, bool)) []*RawResult {
	return bundle.DeprecatedV5Code.returnValues(bundle, f)
}
