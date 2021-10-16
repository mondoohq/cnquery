package terraform

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
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
