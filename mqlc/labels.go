package mqlc

import (
	"errors"
	"strconv"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/types"
)

func createLabel(code *llx.CodeV2, ref uint64, labels *llx.Labels, schema *resources.Schema) (string, error) {
	chunk := code.Chunk(ref)

	if chunk.Call == llx.Chunk_PRIMITIVE {
		return "", nil
	}

	id := chunk.Id
	if chunk.Function == nil {
		return id, nil
	}

	// TODO: workaround to get past the builtin global call
	// this needs proper handling for global calls
	if chunk.Function.Binding == 0 && id != "if" {
		return id, nil
	}

	var parentLabel string
	var err error
	if chunk.Function.Binding != 0 {
		parentLabel, err = createLabel(code, chunk.Function.Binding, labels, schema)
		if err != nil {
			return "", err
		}
	}

	var res string
	switch id {
	case "[]":
		if len(chunk.Function.Args) != 1 {
			panic("don't know how to extract label data from array access without args")
		}

		arg := chunk.Function.Args[0].RawData()
		idx := arg.Value

		switch arg.Type {
		case types.Int:
			res = "[" + strconv.FormatInt(idx.(int64), 10) + "]"
		case types.String:
			res = "[" + idx.(string) + "]"
		default:
			panic("cannot label array index of type " + arg.Type.Label())
		}
		if parentLabel != "" {
			res = parentLabel + res
		}
	case "{}", "${}":
		res = parentLabel
	case "if":
		res = "if"
	default:
		if label, ok := llx.ComparableLabel(id); ok {
			arg := chunk.Function.Args[0].LabelV2(code)
			res = parentLabel + " " + label + " " + arg
		} else if parentLabel == "" {
			res = id
		} else {
			res = parentLabel + "." + id
		}
	}

	// TODO: figure out why this string includes control characters in the first place
	return stripCtlAndExtFromUnicode(res), nil
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
func UpdateLabels(code *llx.CodeV2, labels *llx.Labels, schema *resources.Schema) error {
	if code == nil {
		return errors.New("cannot create labels without code")
	}

	for i := range code.Blocks {
		err := updateLabels(code, code.Blocks[i], labels, schema)
		if err != nil {
			return err
		}
	}

	return nil
}

func updateLabels(code *llx.CodeV2, block *llx.Block, labels *llx.Labels, schema *resources.Schema) error {
	datapoints := block.Datapoints

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

		if _, ok := labels.Labels[checksum]; ok {
			continue
		}

		labels.Labels[checksum], err = createLabel(code, entrypoint, labels, schema)
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
				delete(labels.Labels, assertion.Checksums[i])
			}
		}
	}

	return nil
}
