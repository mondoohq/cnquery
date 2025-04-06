// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package resources

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"strings"

	"go.mondoo.com/cnquery/v11/checksums"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/os/resources/parsers"
	"go.mondoo.com/cnquery/v11/providers/os/resources/plist"
	"sigs.k8s.io/yaml"
)

func fileFromPathOrContent(runtime *plugin.Runtime, args map[string]*llx.RawData) error {
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return errors.New("Wrong type for 'path' it must be a string")
		}

		f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path": llx.StringData(path),
		})
		if err != nil {
			return err
		}
		args["file"] = llx.ResourceData(f, "file")
		delete(args, "path")
	} else {
		if x, ok := args["content"]; ok {
			content := x.Value.(string)
			virtualPath := "in-memory://" + checksums.New.Add(content).String()
			f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
				"path":    llx.StringData(virtualPath),
				"content": llx.StringData(content),
				"exists":  llx.BoolTrue,
			})
			if err != nil {
				return err
			}
			args["file"] = llx.ResourceData(f, "file")
		}
	}
	return nil
}

func initParseIni(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if err := fileFromPathOrContent(runtime, args); err != nil {
		return nil, nil, err
	}

	if _, ok := args["delimiter"]; !ok {
		args["delimiter"] = llx.StringData("=")
	}

	return args, nil, nil
}

func (s *mqlParseIni) id() (string, error) {
	if s.File.Data == nil {
		return "", errors.New("no file provided for parse.ini")
	}

	file := s.File.Data
	del := s.Delimiter.Data
	return file.Path.Data + del, nil
}

func (s *mqlParseIni) content(file *mqlFile) (string, error) {
	c := file.GetContent()
	return c.Data, c.Error
}

func (s *mqlParseIni) sections(content string, delimiter string) (map[string]interface{}, error) {
	ini := parsers.ParseIni(content, delimiter)

	res := make(map[string]interface{}, len(ini.Fields))
	for k, v := range ini.Fields {
		res[k] = v
	}

	return res, nil
}

func (s *mqlParseIni) params(sections map[string]interface{}) (map[string]interface{}, error) {
	res := sections[""]
	if res == nil {
		return map[string]interface{}{}, nil
	}
	return res.(map[string]interface{}), nil
}

func initParseJson(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if err := fileFromPathOrContent(runtime, args); err != nil {
		return nil, nil, err
	}

	return args, nil, nil
}

func (s *mqlParseJson) id() (string, error) {
	if s.File.Data == nil {
		return "", errors.New("no file provided for parse.json")
	}

	file := s.File.Data
	return file.Path.Data, nil
}

func (s *mqlParseJson) content(file *mqlFile) (string, error) {
	c := file.GetContent()
	return c.Data, c.Error
}

func (s *mqlParseJson) params(content string) (interface{}, error) {
	if content == "" {
		return nil, nil
	}

	var res interface{}
	if err := json.Unmarshal([]byte(content), &res); err != nil {
		return nil, err
	}
	return res, nil
}

func initParseXml(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if err := fileFromPathOrContent(runtime, args); err != nil {
		return nil, nil, err
	}

	return args, nil, nil
}

func (s *mqlParseXml) id() (string, error) {
	if s.File.Data == nil {
		return "", errors.New("no file provided for parse.json")
	}

	file := s.File.Data
	return file.Path.Data, nil
}

func (s *mqlParseXml) content(file *mqlFile) (string, error) {
	c := file.GetContent()
	return c.Data, c.Error
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
	x.attributes = map[string]string{}
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
			cur.children = append(cur.children, &xmlElem{
				data:      v,
				isElement: false,
			})
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
				res[field] = cur.(string) + data
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

func (s *mqlParseXml) params(content string) (interface{}, error) {
	if content == "" {
		return nil, nil
	}

	var res xmlElem
	if err := xml.Unmarshal([]byte(content), &res); err != nil {
		return nil, err
	}

	return res.params(), nil
}

func initParseYaml(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if err := fileFromPathOrContent(runtime, args); err != nil {
		return nil, nil, err
	}

	return args, nil, nil
}

func (s *mqlParseYaml) id() (string, error) {
	if s.File.Data == nil {
		return "", errors.New("no file provided for parse.yaml")
	}

	file := s.File.Data
	return file.Path.Data, nil
}

func (s *mqlParseYaml) content(file *mqlFile) (string, error) {
	c := file.GetContent()
	return c.Data, c.Error
}

func (s *mqlParseYaml) params(content string) (map[string]interface{}, error) {
	res := make(map[string](interface{}))

	if content == "" {
		return nil, nil
	}

	err := yaml.Unmarshal([]byte(content), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func initParsePlist(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if err := fileFromPathOrContent(runtime, args); err != nil {
		return nil, nil, err
	}
	return args, nil, nil
}

func (s *mqlParsePlist) id() (string, error) {
	if s.File.Data == nil {
		return "", errors.New("no file provided for parse.plist")
	}

	file := s.File.Data
	return file.Path.Data, nil
}

func (s *mqlParsePlist) content(file *mqlFile) (string, error) {
	c := file.GetContent()
	return c.Data, c.Error
}

func (s *mqlParsePlist) params(content string) (map[string]interface{}, error) {
	return plist.Decode(strings.NewReader(content))
}

func initParseCertificates(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// resolve path to file
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in certificates initialization, it must be a string")
		}

		f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path": llx.StringData(path),
		})
		if err != nil {
			return nil, nil, err
		}
		args["file"] = llx.ResourceData(f, "file")

	} else if x, ok := args["content"]; ok {
		content := x.Value.(string)
		virtualPath := "in-memory://" + checksums.New.Add(content).String()
		f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path":    llx.StringData(virtualPath),
			"content": llx.StringData(content),
			"exists":  llx.BoolTrue,
		})
		if err != nil {
			return nil, nil, err
		}
		args["file"] = llx.ResourceData(f, "file")
		args["path"] = llx.StringData(virtualPath)
	} else {
		return nil, nil, errors.New("missing 'path' or 'content' for parse.json initialization")
	}

	return args, nil, nil
}

func certificatesid(path string) string {
	return "certificates:" + path
}

func (a *mqlParseCertificates) id() (string, error) {
	f := a.File.Data
	if f == nil {
		return "", errors.New("missing file in parse certificate")
	}

	return certificatesid(f.Path.Data), nil
}

func (a *mqlParseCertificates) file() (*mqlFile, error) {
	f, err := CreateResource(a.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(a.Path.Data),
	})
	if err != nil {
		return nil, err
	}
	return f.(*mqlFile), nil
}

func (a *mqlParseCertificates) content(file *mqlFile) (string, error) {
	res := file.GetContent()
	return res.Data, res.Error
}

func (p *mqlParseCertificates) list(content string, path string) ([]interface{}, error) {
	certificates, err := p.MqlRuntime.CreateSharedResource("certificates", map[string]*llx.RawData{
		"pem": llx.StringData(content),
	})
	if err != nil {
		return nil, err
	}

	list, err := p.MqlRuntime.GetSharedData("certificates", certificates.MqlID(), "list")
	if err != nil {
		return nil, err
	}

	return list.Value.([]interface{}), nil
}

func initParseOpenpgp(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// resolve path to file
	if x, ok := args["path"]; ok {
		f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path": x,
		})
		if err != nil {
			return nil, nil, err
		}
		args["file"] = llx.ResourceData(f, "file")

	} else if x, ok := args["content"]; ok {
		content := x.Value.(string)
		virtualPath := "in-memory://" + checksums.New.Add(content).String()
		f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path":    llx.StringData(virtualPath),
			"content": llx.StringData(content),
			"exists":  llx.BoolTrue,
		})
		if err != nil {
			return nil, nil, err
		}
		args["file"] = llx.ResourceData(f, "file")
		args["path"] = llx.StringData(virtualPath)

	} else {
		return nil, nil, errors.New("missing 'path' or 'content' for parse.json initialization")
	}

	return args, nil, nil
}

func (a *mqlParseOpenpgp) id() (string, error) {
	if a.File.Error != nil {
		return "", a.File.Error
	}

	return a.File.Data.Path.Data, nil
}

func (a *mqlParseOpenpgp) content(file plugin.Resource) (string, error) {
	res := file.(*mqlFile).GetContent()
	return res.Data, res.Error
}

func (p *mqlParseOpenpgp) list(content string) ([]interface{}, error) {
	certificates, err := p.MqlRuntime.CreateSharedResource("openpgp.entities", map[string]*llx.RawData{
		"content": llx.StringData(content),
	})
	if err != nil {
		return nil, err
	}

	list, err := p.MqlRuntime.GetSharedData("openpgp.entities", certificates.MqlID(), "list")
	if err != nil {
		return nil, err
	}

	return list.Value.([]interface{}), nil
}
