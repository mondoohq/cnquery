// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mqlc

import (
	"errors"
	"regexp"
	"strconv"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/types"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func createLabel(res *llx.CodeBundle, ref uint64, schema llx.Schema) (string, error) {
	code := res.CodeV2
	chunk := code.Chunk(ref)

	if chunk.Call == llx.Chunk_PRIMITIVE {
		// In the case of refs, we want to check for the name of the variable,
		// which is what every final ref should lead to
		if chunk.Primitive.Type == string(types.Ref) {
			if deref, ok := chunk.Primitive.RefV2(); ok {
				ref = deref
			}
		}

		// TODO: better labels if we don't have it as a var
		label := res.Vars[ref]

		return label, nil
	}

	id := chunk.Id
	if chunk.Function == nil {
		return id, nil
	}

	// TODO: workaround to get past the builtin global call
	// this needs proper handling for global calls
	if chunk.Function.Binding == 0 && id != "if" && id != "createResource" {
		return id, nil
	}

	var parentLabel string
	var err error
	if id == "createResource" {
		if ref, ok := chunk.Function.Args[0].RefV2(); ok {
			parentLabel, err = createLabel(res, ref, schema)
			if err != nil {
				return "", err
			}
		}
	} else if chunk.Function.Binding != 0 {
		parentLabel, err = createLabel(res, chunk.Function.Binding, schema)
		if err != nil {
			return "", err
		}
	}

	var label string
	switch id {
	case "[]":
		if len(chunk.Function.Args) != 1 {
			panic("don't know how to extract label data from array access without args")
		}

		arg := chunk.Function.Args[0].RawData()
		idx := arg.Value

		switch arg.Type {
		case types.Int:
			label = "[" + strconv.FormatInt(idx.(int64), 10) + "]"
		case types.String:
			if chunk.Function.Type == string(types.Dict) && isAccessor(idx.(string)) {
				label = idx.(string)
			} else {
				label = "[" + idx.(string) + "]"
			}
		default:
			panic("cannot label array index of type " + arg.Type.Label())
		}
		if parentLabel != "" {
			if label != "" && label[0] == '[' {
				label = parentLabel + label
			} else {
				label = parentLabel + "." + label
			}
		}
	case "{}", "${}":
		label = parentLabel
	case "createResource":
		label = parentLabel + "." + string(chunk.Type())
	case "if":
		label = "if"
	default:
		if x, ok := llx.ComparableLabel(id); ok {
			arg := chunk.Function.Args[0].LabelV2(code)
			label = parentLabel + " " + x + " " + arg
		} else if parentLabel == "" {
			label = id
		} else {
			label = parentLabel + "." + id
		}
	}

	// TODO: figure out why this string includes control characters in the first place
	return stripCtlAndExtFromUnicode(label), nil
}

var reAccessor = regexp.MustCompile(`^[\p{L}\d_]+$`)

func isAccessor(s string) bool {
	return reAccessor.MatchString(s)
}

// Unicode normalization and filtering, see http://blog.golang.org/normalization and
// http://godoc.org/golang.org/x/text/unicode/norm for more details.
func stripCtlAndExtFromUnicode(str string) string {
	isOk := func(r rune) bool {
		return r < 32 || r >= 127
	}
	// The isOk filter is such that there is no need to chain to norm.NFC
	t := transform.Chain(norm.NFKD, transform.RemoveFunc(isOk))
	str, _, _ = transform.String(t, str)
	return str
}

// UpdateLabels for the given code under the schema
func UpdateLabels(res *llx.CodeBundle, schema llx.Schema) error {
	if res == nil || res.CodeV2 == nil {
		return errors.New("cannot create labels without code")
	}

	for i := range res.CodeV2.Blocks {
		err := updateLabels(res, res.CodeV2.Blocks[i], schema)
		if err != nil {
			return err
		}
	}

	// not needed anymore since we have all the info in labels now
	res.Vars = nil

	return nil
}

func updateLabels(res *llx.CodeBundle, block *llx.Block, schema llx.Schema) error {
	datapoints := block.Datapoints
	code := res.CodeV2
	labels := res.Labels.Labels

	// We don't want assertions to become labels. Their data should not be printed
	// regularly but instead be processed through the assertion itself
	if code.Assertions != nil {
		assertionPoints := map[uint64]struct{}{}
		for _, assertion := range code.Assertions {
			for j := range assertion.Refs {
				assertionPoints[assertion.Refs[j]] = struct{}{}
			}
		}

		filtered := []uint64{}
		for i := range datapoints {
			ref := datapoints[i]
			if _, ok := assertionPoints[ref]; ok {
				continue
			}
			filtered = append(filtered, ref)
		}
		datapoints = filtered
	}

	labelrefs := append(block.Entrypoints, datapoints...)

	var err error
	for _, entrypoint := range labelrefs {
		checksum, ok := code.Checksums[entrypoint]
		if !ok {
			return errors.New("failed to create labels, cannot find checksum for this entrypoint " + strconv.FormatUint(uint64(entrypoint), 10))
		}

		if _, ok := labels[checksum]; ok {
			continue
		}

		labels[checksum], err = createLabel(res, entrypoint, schema)

		if err != nil {
			return err
		}
	}

	// any more checksums that might have been set need to be removed, since we don't need them
	// TODO: there must be a way to do this without having to create the label first
	if code.Assertions != nil {
		for _, assertion := range code.Assertions {
			if !assertion.DecodeBlock {
				continue
			}
			for i := 0; i < len(assertion.Checksums); i++ {
				delete(labels, assertion.Checksums[i])
			}
		}
	}

	return nil
}
