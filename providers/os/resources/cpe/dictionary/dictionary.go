package dictionary

import (
	"encoding/xml"
	"github.com/facebookincubator/nvdtools/cpedict"
	"github.com/facebookincubator/nvdtools/wfn"
	"io"
	"strings"
)

// CPEList represents a CPE dictionary
type CPEList struct {
	cpedict.CPEList
}

// cpes include a couple or errors that we need to fix
var aliasEcosystem = map[string]string{
	"nodejs":  "node.js",
	"node-js": "node.js",
	"npmjs":   "node.js",
	"pypi":    "python",
	"andoird": "android",
	"andriod": "android",
}

// Filter filters CPE dictionary with the following rules:
// 1. Remove all CPEs with deprecated status
// 2. Remove all CPEs that are assigned to hardware
// 2. Remove all CPEs that do not have a vendor
// 4. Remove all CPEs that do not have a cpe 2.3 URI
// 5. Convert all CPEs to cpe 2.3 URI
func (l *CPEList) Map() map[string]map[string]string {
	list := map[string]*wfn.Attributes{}

	// parts: application (a), operating system (o), hardware (h)
	for _, item := range l.Items {
		cpe23 := item.CPE23

		// remove hardware and operating systems
		if cpe23.Name.Part == "h" || cpe23.Name.Part == "o" {
			continue
		}

		// we do not need version and update
		attributes := wfn.Attributes(cpe23.Name)
		vCPE := VersionlessCPE(&attributes)
		list[vCPE.BindToFmtString()] = vCPE
	}

	// convert to slice
	mapEcosystem := map[string]map[string]string{}
	for k := range list {
		entry := list[k]
		ecosystem := wfn.StripSlashes(entry.TargetSW)
		product := wfn.StripSlashes(entry.Product)
		// the library does not parse escaped slashes properly
		product = strings.ReplaceAll(product, "\\/", "/")
		if mapEcosystem[ecosystem] == nil {
			mapEcosystem[ecosystem] = map[string]string{}
		}
		mapEcosystem[ecosystem][product] = entry.BindToFmtString()
	}

	return mapEcosystem
}

// VersionlessCPE returns a CPE without version and update
func VersionlessCPE(cpe *wfn.Attributes) *wfn.Attributes {
	c := *cpe
	c.Version = ""
	c.Update = ""
	return &c
}

// Decode decodes dictionary XML
func Decode(r io.Reader) (*CPEList, error) {
	var list CPEList
	if err := xml.NewDecoder(r).Decode(&list); err != nil {
		return nil, err
	}
	return &list, nil
}
