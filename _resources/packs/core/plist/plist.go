package plist

import (
	"bytes"
	"encoding/json"
	"io"

	"howett.net/plist"
)

func ToXml(r io.ReadSeeker) ([]byte, error) {
	// convert file format to xml
	var val interface{}
	dec := plist.NewDecoder(r)
	err := dec.Decode(&val)
	if err != nil {
		return nil, err
	}

	out := &bytes.Buffer{}
	enc := plist.NewEncoderForFormat(out, plist.XMLFormat)
	err = enc.Encode(val)
	return out.Bytes(), err
}

func Decode(r io.ReadSeeker) (map[string]interface{}, error) {
	var data map[string]interface{}
	decoder := plist.NewDecoder(r)
	err := decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	// NOTE: we need to do the extra conversion here to make sure we use supported
	// values by our dict structure: string, float64, int64
	// plist also uses uint64 heavily which we do not support
	// TODO: we really do not want to use the poor-man's json conversion version
	jsondata, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var dataJson map[string]interface{}
	err = json.Unmarshal(jsondata, &dataJson)
	if err != nil {
		return nil, err
	}

	return dataJson, nil
}
