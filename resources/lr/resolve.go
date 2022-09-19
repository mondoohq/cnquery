package lr

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

func Resolve(filePath string, readFile func(path string) ([]byte, error)) (*LR, error) {
	raw, err := readFile(filePath)
	if err != nil {
		return nil, err
	}

	anchorPath := path.Dir(filePath)

	res, err := Parse(string(raw))
	if err != nil {
		return nil, err
	}

	res.imports = make(map[string]map[string]struct{})
	res.packPaths = map[string]string{}
	importMap := map[string]map[string]*Resource{}
	for i := range res.Imports {
		// note: we do not recurse into these imports; we only need to know
		// about the things that the import exposes, not about its dependencies
		importPath := res.Imports[i]
		packName := strings.TrimSuffix(path.Base(importPath), ".lr")
		relPath := path.Join(anchorPath, importPath)

		raw, err := readFile(relPath)
		if err != nil {
			return nil, err
		}

		childLR, err := Parse(string(raw))
		if err != nil {
			return nil, err
		}

		resources := map[string]struct{}{}
		importMap[packName] = map[string]*Resource{}
		for i := range childLR.Resources {
			resource := childLR.Resources[i]
			resources[resource.ID] = struct{}{}
			importMap[packName][resource.ID] = resource
		}

		res.imports[packName] = resources

		goPkg := childLR.Options["go_package"]
		if goPkg == "" {
			return nil, errors.New("cannot find name of the go package in " + importPath + " - make sure you set the go_package name")
		}
		res.packPaths[packName] = goPkg
	}

	res.aliases = map[string]*Resource{}
	for _, a := range res.Aliases {
		pack, resourceName, ok := strings.Cut(a.Type.Type, ".")
		if !ok {
			return nil, fmt.Errorf("%v is not a valid alias", a)
		}
		found := false
		p, ok := importMap[pack]
		if ok {
			r, ok := p[resourceName]
			if ok {
				found = true
			}
			res.aliases[a.Definition.Type] = r
		}
		if !found {
			return nil, fmt.Errorf("%s was aliased but not imported", a.Type.Type)
		}
	}

	res.Imports = nil

	return res, nil
}
