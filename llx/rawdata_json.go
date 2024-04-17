// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"go.mondoo.com/cnquery/v11/types"
	"go.mondoo.com/cnquery/v11/utils/sortx"
)

func intKeys(m map[int]interface{}) []int {
	keys := make([]int, len(m))
	var i int
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
}

// Note: We override the default output here to enable JSON5 like export of infinity.
func int2json(i int64) string {
	if i == math.MaxInt64 {
		return "Inf"
	}
	if i == math.MinInt64 {
		return "-Inf"
	}

	return strconv.FormatInt(i, 10)
}

// Note: We override the default output here to enable JSON5 like export of infinity.
func float2json(f float64) string {
	if math.IsInf(f, 1) {
		return "Inf"
	}
	if math.IsInf(f, -1) {
		return "-Inf"
	}

	return strconv.FormatFloat(f, 'g', -1, 64)
}

// Takes care of escaping the given string
func string2json(s string) (string, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func label(ref string, bundle *CodeBundle, isResource bool) string {
	if bundle == nil {
		return "<unknown>"
	}

	labels := bundle.Labels
	if labels == nil {
		return ref
	}

	label := labels.Labels[ref]
	if label == "" {
		return "<unknown>"
	}

	return label
}

func removeUnderscoreKeys(keys []string) []string {
	results := make([]string, 0, len(keys))
	for i := 0; i < len(keys); i++ {
		if keys[i] != "" && keys[i][0] != '_' {
			results = append(results, keys[i])
		}
	}
	return results
}

func refMapJSON(typ types.Type, data map[string]interface{}, codeID string, bundle *CodeBundle, buf *bytes.Buffer) error {
	buf.WriteByte('{')

	keys := sortx.Keys(data)

	// What is the best explanation for why we do this?
	keys = removeUnderscoreKeys(keys)

	last := len(keys) - 1
	for i, k := range keys {
		v := data[k]
		label := label(k, bundle, true)
		str, err := string2json(label)
		if err != nil {
			return err
		}
		buf.WriteString(str)
		buf.WriteString(":")

		val := v.(*RawData)
		if val.Error != nil {
			buf.WriteString(PrettyPrintString("Error: " + val.Error.Error()))
		} else {
			rawDataJSON(val.Type, val.Value, k, bundle, buf)
		}

		if i != last {
			buf.WriteByte(',')
		}
	}

	buf.WriteByte('}')
	return nil
}

func rawDictJSON(typ types.Type, raw interface{}, buf *bytes.Buffer) error {
	switch data := raw.(type) {
	case bool:
		if data {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
		return nil

	case int64:
		buf.WriteString(int2json(data))
		return nil

	case float64:
		buf.WriteString(float2json(data))
		return nil

	case string:
		str, err := string2json(data)
		if err != nil {
			return err
		}
		buf.WriteString(str)
		return nil

	case time.Time:
		b, err := data.MarshalJSON()
		buf.Write(b)
		return err

	case []interface{}:
		buf.WriteByte('[')

		last := len(data) - 1
		for i := range data {
			err := rawDictJSON(typ, data[i], buf)
			if err != nil {
				return err
			}
			if i != last {
				buf.WriteByte(',')
			}
		}

		buf.WriteByte(']')
		return nil

	case map[string]interface{}:
		buf.WriteByte('{')

		keys := sortx.Keys(data)

		last := len(keys) - 1
		for i, k := range keys {
			v := data[k]
			str, err := string2json(k)
			if err != nil {
				return err
			}
			buf.WriteString(str + ":")

			if v == nil {
				buf.WriteString("null")
			} else {
				err := rawDictJSON(typ, v, buf)
				if err != nil {
					return err
				}
			}

			if i != last {
				buf.WriteByte(',')
			}
		}

		buf.WriteByte('}')
		return nil

	default:
		b, err := json.Marshal(raw)
		buf.Write(b)
		return err
	}
}

func rawArrayJSON(typ types.Type, data []interface{}, codeID string, bundle *CodeBundle, buf *bytes.Buffer) error {
	buf.WriteByte('[')

	last := len(data) - 1
	childType := typ.Child()
	var err error
	for i := range data {
		err = rawDataJSON(childType, data[i], codeID, bundle, buf)
		if err != nil {
			return err
		}

		if i != last {
			buf.WriteByte(',')
		}
	}

	buf.WriteByte(']')

	return nil
}

func rawStringMapJSON(typ types.Type, data map[string]interface{}, codeID string, bundle *CodeBundle, buf *bytes.Buffer) error {
	buf.WriteByte('{')

	last := len(data) - 1
	childType := typ.Child()
	keys := sortx.Keys(data)

	for i, key := range keys {
		str, err := string2json(key)
		if err != nil {
			return err
		}
		buf.WriteString(str + ":")

		err = rawDataJSON(childType, data[key], codeID, bundle, buf)
		if err != nil {
			return err
		}

		if i != last {
			buf.WriteByte(',')
		}
	}

	buf.WriteByte('}')

	return nil
}

func rawIntMapJSON(typ types.Type, data map[int]interface{}, codeID string, bundle *CodeBundle, buf *bytes.Buffer) error {
	buf.WriteByte('{')

	last := len(data) - 1
	childType := typ.Child()

	keys := intKeys(data)
	sort.Ints(keys)

	for i, key := range keys {
		str, err := string2json(strconv.Itoa(key))
		if err != nil {
			return err
		}
		buf.WriteString(str + ":")

		err = rawDataJSON(childType, data[key], codeID, bundle, buf)
		if err != nil {
			return err
		}

		if i != last {
			buf.WriteByte(',')
		}
	}

	buf.WriteByte('}')

	return nil
}

// The heart of the JSON marshaller. We try to avoid the default marshaller whenever
// possible for now, because our type system provides most of the information we need,
// allowing us to avoid more costly reflection calls.
func rawDataJSON(typ types.Type, data interface{}, codeID string, bundle *CodeBundle, buf *bytes.Buffer) error {
	if typ.NotSet() {
		return errors.New("type information is missing")
	}

	if data == nil {
		buf.WriteString("null")
		return nil
	}

	switch typ.Underlying() {
	case types.Any:
		r, err := json.Marshal(data)
		buf.Write(r)
		return err

	case types.Ref:
		r := "\"ref:" + fmt.Sprintf("%d", data.(int32)) + "\""
		buf.WriteString(r)
		return nil

	case types.Nil:
		buf.WriteString("null")
		return nil

	case types.Bool:
		if data.(bool) {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
		return nil

	case types.Int:
		buf.WriteString(int2json(data.(int64)))
		return nil

	case types.Float:
		// Note: We override the default output here to enable JSON5 like export of infinity.
		if math.IsInf(data.(float64), 1) {
			buf.WriteString("Inf")
			return nil
		}
		if math.IsInf(data.(float64), -1) {
			buf.WriteString("-Inf")
			return nil
		}

		buf.WriteString(strconv.FormatFloat(data.(float64), 'g', -1, 64))
		return nil

	case types.String:
		str, err := string2json(data.(string))
		if err != nil {
			return err
		}
		buf.WriteString(str)
		return nil

	case types.Regex:
		raw, err := string2json(data.(string))
		if err != nil {
			return err
		}
		buf.WriteByte(raw[0])
		buf.WriteByte('/')
		buf.WriteString(raw[1 : len(raw)-1])
		buf.WriteByte('/')
		buf.WriteByte(raw[len(raw)-1])
		return nil

	case types.Time:
		time := data.(*time.Time)
		if time == nil {
			buf.WriteString("null")
			return nil
		}

		if *time == NeverPastTime || *time == NeverFutureTime {
			r := "\"" + "Never" + "\""

			buf.WriteString(r)
			return nil
		}

		b, err := time.MarshalJSON()
		buf.Write(b)
		return err

	case types.Dict:
		return rawDictJSON(typ, data, buf)

	case types.Score:
		buf.WriteString(ScoreString(data.([]byte)))
		return nil

	case types.Block:
		return refMapJSON(typ, data.(map[string]interface{}), codeID, bundle, buf)

	case types.ArrayLike:
		return rawArrayJSON(typ, data.([]interface{}), codeID, bundle, buf)

	case types.MapLike:
		if typ.Key() == types.String {
			return rawStringMapJSON(typ, data.(map[string]interface{}), codeID, bundle, buf)
		}
		if typ.Key() == types.Int {
			return rawIntMapJSON(typ, data.(map[int]interface{}), codeID, bundle, buf)
		}
		return errors.New("unable to marshal map, its type is not supported: " + typ.Label() + ", raw: " + fmt.Sprintf("%#v", data))

	case types.ResourceLike:
		r := data.(Resource)
		idline := r.MqlName()
		if id := r.MqlID(); id != "" {
			idline += " id = " + id
		}

		str, err := string2json(idline)
		if err != nil {
			return err
		}
		buf.WriteString(str)
		return nil

	default:
		b, err := json.Marshal(data)
		buf.Write(b)
		return err
	}
}

func JSONerror(err error) []byte {
	str, err := string2json(err.Error())
	if err != nil {
		return []byte("{\"error\":\"" + err.Error() + "\"}")
	}
	return []byte("{\"error\":" + str + "}")
}

func (r *RawData) JSON(codeID string, bundle *CodeBundle) []byte {
	if r.Value == nil && r.Error != nil {
		return JSONerror(r.Error)
	}

	var res bytes.Buffer
	if err := rawDataJSON(r.Type, r.Value, codeID, bundle, &res); err != nil {
		return JSONerror(err)
	}
	return res.Bytes()
}

func (r *RawData) JSONfield(codeID string, bundle *CodeBundle) []byte {
	label := label(codeID, bundle, true)
	value := r.JSON(codeID, bundle)
	str, err := string2json(label)
	if err != nil {
		return JSONerror(err)
	}
	return []byte(str + ":" + string(value))
}
