package leise

import (
	"errors"
	"strconv"

	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
)

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
		argLen := len(chunk.Function.Args)

		if argLen > 3 {
			panic("don't know how to extract label data for if-call with too many args")
		}

		// if there are no labels yet
		if argLen < 2 {
			return "if", nil
		}

		fref := chunk.Function.Args[1]
		ref, ok := fref.Ref()
		if !ok {
			return "", errors.New("cannot get function reference from first block of if-statement")
		}

		function := code.Functions[ref-1]
		err = UpdateLabels(function, labels, schema)
		if err != nil {
			return "", err
		}

		if argLen == 3 {
			fref := chunk.Function.Args[2]
			ref, ok := fref.Ref()
			if !ok {
				return "", errors.New("cannot get function reference from first block of if-statement")
			}

			function := code.Functions[ref-1]
			err = UpdateLabels(function, labels, schema)
			if err != nil {
				return "", err
			}
		}

	default:
		if parentLabel == "" {
			res = id
		} else {
			res = parentLabel + "." + id
		}
	}

	return res, nil
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
