package terraform

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func NewHCLFileLoader() *hclFileLoader {
	hclParser := hclparse.NewParser()

	return &hclFileLoader{
		hclParser: hclParser,
	}
}

type hclFileLoader struct {
	hclParser *hclparse.Parser
}

// ParseHclFile parses a single terraform file
func (h *hclFileLoader) ParseHclFile(filepath string) error {
	var parseFunc func(filename string) (*hcl.File, hcl.Diagnostics)
	switch {
	case strings.HasSuffix(filepath, ".tf"):
		parseFunc = h.hclParser.ParseHCLFile
	case strings.HasSuffix(filepath, ".tf.json"):
		parseFunc = h.hclParser.ParseJSONFile
	default:
		return nil
	}

	_, diag := parseFunc(filepath)
	if diag != nil && diag.HasErrors() {
		return diag
	}
	return nil
}

// ParseHclDirectory parses all files in a directory
func (h *hclFileLoader) ParseHclDirectory(path string, fileList []os.DirEntry) error {
	for i := range fileList {
		fi := fileList[i]

		if fi.IsDir() {
			continue
		}

		path := filepath.Join(path, fi.Name())
		err := h.ParseHclFile(path)
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *hclFileLoader) GetParser() *hclparse.Parser {
	return h.hclParser
}

func ReadTfVarsFromFile(filename string, terraformVars map[string]*hcl.Attribute) error {
	switch {
	case strings.HasSuffix(filename, ".tfvars"):
		fallthrough
	case strings.HasSuffix(filename, ".tfvars.json"):

		src, err := os.ReadFile(filename)
		if err != nil {
			return err
		}

		// we ignore the diagnostics information here
		variableFile, _ := hclsyntax.ParseConfig(src, filename, hcl.Pos{Line: 1, Column: 1})

		// NOTE: we ignore the diagnostics info
		attrs, _ := variableFile.Body.JustAttributes()
		for k := range attrs {
			v := attrs[k]
			terraformVars[k] = v
		}
		return nil
	default:
		return nil
	}
}

func ReadTfVarsFromDir(path string, fileList []os.DirEntry, terraformVars map[string]*hcl.Attribute) error {
	for i := range fileList {
		fi := fileList[i]

		if fi.IsDir() {
			continue
		}

		filename := filepath.Join(path, fi.Name())
		err := ReadTfVarsFromFile(filename, terraformVars)
		if err != nil {
			return err
		}
	}
	return nil
}
