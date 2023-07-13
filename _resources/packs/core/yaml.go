package core

import (
	"bytes"
	"io/ioutil"
	"strings"

	"k8s.io/client-go/util/jsonpath"
	"sigs.k8s.io/yaml"
)

func (l *mqlYamlPath) id() (string, error) {
	filepath, err := l.Filepath()
	if err != nil {
		return "", err
	}

	jsonpath, err := l.Jsonpath()
	if err != nil {
		return "", err
	}

	var id strings.Builder
	id.WriteString(filepath)
	if len(jsonpath) > 0 {
		id.WriteString(" -jsonpath ")
		id.WriteString(jsonpath)
	}
	return id.String(), nil
}

func (l *mqlYamlPath) GetResult() (string, error) {
	// get file content
	filepath, err := l.Filepath()
	if err != nil {
		return "", err
	}

	// get jsonpath value
	jp, err := l.Jsonpath()
	if err != nil {
		return "", err
	}

	osProvider, err := osProvider(l.MotorRuntime.Motor)
	if err != nil {
		return "", err
	}

	// TODO: I could not get this running with MQL file resource, the content was never returned
	f, err := osProvider.FS().Open(filepath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}

	// unmarshal file content
	var jsonInterface interface{}
	err = yaml.Unmarshal([]byte(data), &jsonInterface)
	if err != nil {
		return "", err
	}

	// parse json path expression
	j := jsonpath.New("mqlyamlpath")
	j.AllowMissingKeys(false)
	err = j.Parse(jp)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	err = j.Execute(buf, jsonInterface)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
