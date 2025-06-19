// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package xml

import (
	"encoding/xml"
	"io"
	"strings"
)

func Parse(data []byte) (any, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var res xmlElem
	if err := xml.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	return res.params(), nil
}

type xmlElem struct {
	attributes map[string]string
	children   []*xmlElem
	data       string
	isElement  bool
}

func (x *xmlElem) addAttr(a []xml.Attr) {
	if len(a) == 0 {
		return
	}
	if x.attributes == nil {
		x.attributes = map[string]string{}
	}
	for ai := range a {
		attr := a[ai]
		x.attributes[attrName(attr.Name)] = attr.Value
	}
}

func (x *xmlElem) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	x.data = start.Name.Local
	x.isElement = true
	path := []*xmlElem{
		x,
	}
	path[0].addAttr(start.Attr)

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			nu := &xmlElem{
				data:      elem.Name.Local,
				isElement: true,
			}
			nu.addAttr(elem.Attr)
			parent := path[len(path)-1]
			parent.children = append(parent.children, nu)
			path = append(path, nu)
		case xml.EndElement:
			path = path[:len(path)-1]
		case xml.CharData:
			cur := path[len(path)-1]
			v := strings.TrimSpace(string(elem))
			if v != "" {
				cur.children = append(cur.children, &xmlElem{
					data:      v,
					isElement: false,
				})
			}
		}
	}
}

func (x *xmlElem) _params() (string, bool, map[string]any) {
	if !x.isElement {
		return x.data, x.isElement, nil
	}
	res := map[string]any{}
	for k, v := range x.attributes {
		res["@"+k] = v
	}

	for i := range x.children {
		child := x.children[i]
		data, isElem, params := child._params()

		// text data is added flat
		if !isElem {
			field := "__text"
			if cur, ok := res[field]; ok {
				if cur.(string) != "" {
					res[field] = cur.(string) + "\n" + data
				} else {
					res[field] = data
				}
			} else {
				res[field] = data
			}
			continue
		}

		if len(params) == 1 {
			if text, ok := params["__text"]; ok {
				exist, ok := res[data]
				if !ok {
					res[data] = text
					continue
				}

				arr, ok := exist.([]any)
				if ok {
					arr = append(arr, text)
				} else {
					arr = []any{exist, text}
				}
				res[data] = arr

				continue
			}
		}

		// if the key doesn't exist, we just store it as a flat value
		cur, ok := res[data]
		if !ok {
			res[data] = params
			continue
		}

		// if the key does exist, we need to turn it into a list or append
		// to any existing list
		arr, ok := cur.([]any)
		if ok {
			arr = append(arr, params)
		} else {
			arr = []any{cur, params}
		}
		res[data] = arr
	}

	return x.data, true, res
}

func (x *xmlElem) params() map[string]any {
	key, isElem, params := x._params()
	if !isElem {
		return map[string]any{"__text": key}
	}
	if len(params) == 1 {
		if data, ok := params["__text"]; ok {
			return map[string]any{key: data}
		}
	}
	return map[string]any{key: params}
}

func attrName(n xml.Name) string {
	if n.Space == "" {
		return n.Local
	}
	return n.Space + ":" + n.Local
}
