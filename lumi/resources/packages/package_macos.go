package packages

import (
	"io"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
	plist "howett.net/plist"
)

// parse macos system version property list
func ParseMacOSPackages(input io.Reader) ([]Package, error) {
	var r io.ReadSeeker
	r, ok := input.(io.ReadSeeker)

	// if the read seaker is not implemented lets cache stdout in-memory
	if !ok {
		packageList, err := ioutil.ReadAll(input)
		if err != nil {
			return nil, err
		}
		r = strings.NewReader(string(packageList))
	}

	type sysProfilerItems struct {
		Name    string `plist:"_name"`
		Version string `plist:"version"`
	}

	type sysProfiler struct {
		Items []sysProfilerItems `plist:"_items"`
	}

	var data []sysProfiler
	decoder := plist.NewDecoder(r)
	err := decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	if len(data) != 1 {
		return nil, errors.New("format not supported")
	}

	pkgs := make([]Package, len(data[0].Items))
	for i, entry := range data[0].Items {
		pkgs[i].Name = entry.Name
		pkgs[i].Version = entry.Version
	}

	return pkgs, nil
}
