package terraform

import (
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func ParseHclDirectory(path string, fileList []fs.FileInfo) (*hclparse.Parser, error) {
	// parse all files
	hclParser := hclparse.NewParser()
	for i := range fileList {
		fi := fileList[i]

		if fi.IsDir() {
			continue
		}

		var parseFunc func(filename string) (*hcl.File, hcl.Diagnostics)
		switch {
		case strings.HasSuffix(fi.Name(), ".tf"):
			parseFunc = hclParser.ParseHCLFile
		case strings.HasSuffix(fi.Name(), ".tf.json"):
			parseFunc = hclParser.ParseJSONFile
		default:
			continue
		}

		path := filepath.Join(path, fi.Name())
		_, diag := parseFunc(path)
		if diag != nil && diag.HasErrors() {
			return nil, diag
		}
	}

	return hclParser, nil
}

func ParseTfVars(path string, fileList []fs.FileInfo) (map[string]*hcl.Attribute, error) {
	terraformVars := make(map[string]*hcl.Attribute)

	for i := range fileList {
		fi := fileList[i]

		if fi.IsDir() {
			continue
		}

		switch {
		case strings.HasSuffix(fi.Name(), ".tfvars"):
			fallthrough
		case strings.HasSuffix(fi.Name(), ".tfvars.json"):
			filename := filepath.Join(path, fi.Name())
			src, err := ioutil.ReadFile(filename)
			if err != nil {
				return nil, err
			}

			// we ignore the diagnositics information here
			variableFile, _ := hclsyntax.ParseConfig(src, filename, hcl.Pos{Line: 1, Column: 1})

			// NOTE: we ignore the diagnositics info
			attrs, _ := variableFile.Body.JustAttributes()
			for k := range attrs {
				v := attrs[k]
				terraformVars[k] = v
			}
		default:
			continue
		}
	}
	return terraformVars, nil
}
