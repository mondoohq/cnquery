package leise

import (
	"errors"
	"strconv"

	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
)

func createLabel(code *llx.Code, ref int32, schema *lumi.Schema) (string, *llx.Labels, error) {
	chunk := code.Code[ref-1]

	if chunk.Call == llx.Chunk_PRIMITIVE {
		return "", nil, nil
	}

	id := chunk.Id
	if chunk.Function == nil {
		return id, nil, nil
	}

	if chunk.Function.Binding == 0 {
		return id, nil, nil
	}

	parentLabel, blockLabels, err := createLabel(code, chunk.Function.Binding, schema)
	if err != nil {
		return "", nil, err
	}
	if blockLabels != nil {
		return "", nil, errors.New("Don't know how to handle parent block labels")
	}

	var res string
	switch id {
	case "[]":
		if len(chunk.Function.Args) != 1 {
			panic("Don't know how to extract label data from array access without args")
		}

		arg := chunk.Function.Args[0].RawData()
		idx := arg.Value

		switch arg.Type {
		case types.Int:
			res = "[" + strconv.FormatInt(idx.(int64), 10) + "]"
		case types.String:
			res = "[" + idx.(string) + "]"
		default:
			panic("Cannot label array index of type " + arg.Type.Label())
		}
		if parentLabel != "" {
			res = parentLabel + res
		}
	case "{}":
		res = parentLabel
		if len(chunk.Function.Args) != 1 {
			panic("Don't know how to extract label data from more than one arg!")
		}

		fref := chunk.Function.Args[0]
		if !types.Type(fref.Type).IsFunction() {
			panic("Don't know how to extract label data when argument is not a function: " + types.Type(fref.Type).Label())
		}

		ref, ok := fref.Ref()
		if !ok {
			panic("Cannot find function reference for data extraction")
		}

		function := code.Functions[ref-1]
		blockLabels, err = CreateLabels(function, schema)
		if err != nil {
			return "", nil, err
		}

	default:
		if parentLabel == "" {
			res = id
		} else {
			res = parentLabel + "." + id
		}
	}

	return res, blockLabels, nil
}

// CreateLabels for the given code under the schema
func CreateLabels(code *llx.Code, schema *lumi.Schema) (*llx.Labels, error) {
	if code == nil {
		return nil, errors.New("Cannot create labels without code")
	}

	labels := &llx.Labels{}

	if len(code.Entrypoints) > 0 {
		labels.Labels = make(map[string]string)
	}

	var err error
	var blockLabels *llx.Labels
	for _, entrypoint := range code.Entrypoints {
		checksum, ok := code.Checksums[entrypoint]
		if !ok {
			return nil, errors.New("failed to create labels, cannot find checksum for this entrypoint " + strconv.FormatUint(uint64(entrypoint), 10))
		}

		labels.Labels[checksum], blockLabels, err = createLabel(code, entrypoint, schema)
		if err != nil {
			return nil, err
		}

		if blockLabels != nil {
			for k, v := range blockLabels.Labels {
				labels.Labels[k] = v
			}
		}
	}

	return labels, nil
}
