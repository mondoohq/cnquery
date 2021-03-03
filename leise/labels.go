package leise

import (
	"errors"
	"strconv"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
)

func createArgLabel(arg *llx.Primitive, code *llx.Code, labels *llx.Labels, schema *lumi.Schema) error {
	if !types.Type(arg.Type).IsFunction() {
		return nil
	}

	ref, ok := arg.Ref()
	if !ok {
		return errors.New("cannot get function reference")
	}

	function := code.Functions[ref-1]
	return UpdateLabels(function, labels, schema)
}

func createLabel(code *llx.Code, ref int32, labels *llx.Labels, schema *lumi.Schema) (string, error) {
	chunk := code.Code[ref-1]

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
	case "{}":
		res = parentLabel
		if len(chunk.Function.Args) != 1 {
			panic("don't know how to extract label data from more than one arg!")
		}

		fref := chunk.Function.Args[0]
		if !types.Type(fref.Type).IsFunction() {
			panic("don't know how to extract label data when argument is not a function: " + types.Type(fref.Type).Label())
		}

		ref, ok := fref.Ref()
		if !ok {
			panic("cannot find function reference for data extraction")
		}

		function := code.Functions[ref-1]
		err = UpdateLabels(function, labels, schema)
		if err != nil {
			return "", err
		}

	case "if":
		res = "if"

		var i int
		max := len(chunk.Function.Args)
		for i+1 < max {
			arg := chunk.Function.Args[i+1]
			if err := createArgLabel(arg, code, labels, schema); err != nil {
				return "", err
			}
			i += 2
		}

		if i < max {
			arg := chunk.Function.Args[i]
			if err := createArgLabel(arg, code, labels, schema); err != nil {
				return "", err
			}
		}

	default:
		if label, ok := llx.ComparableLabel(id); ok {
			arg := chunk.Function.Args[0].Label(code)
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
func UpdateLabels(code *llx.Code, labels *llx.Labels, schema *lumi.Schema) error {
	if code == nil {
		return errors.New("cannot create labels without code")
	}

	labelrefs := append(code.Entrypoints, code.Datapoints...)

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

	return nil
}
