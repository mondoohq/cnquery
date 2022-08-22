package lr

import (
	"io/ioutil"
	"path"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/types"
)

// Collector provides helpers for go files inside a context
type Collector struct {
	path string
	data types.StringToStrings
}

// NewCollector instantiates a collector to watch files in the context of a LR directory
func NewCollector(lrFile string) *Collector {
	base := path.Dir(lrFile)
	if base == "" {
		panic("Cannot find base folder from LR file in '" + lrFile + "'")
	}
	res := &Collector{
		path: base,
	}
	err := res.collect()
	if err != nil {
		panic("failed to collect: " + err.Error())
	}
	return res
}

var regexMaps = map[string]*regexp.Regexp{
	"init": regexp.MustCompile("func \\(\\S+ \\*(mql\\S+)\\) init\\(\\S+ \\*resources.Args\\) \\(\\*resources.Args, \\S+, error\\) {"),
}

func (c *Collector) collect() error {
	files, err := ioutil.ReadDir(c.path)
	if err != nil {
		return err
	}
	for i := range files {
		file := files[i]
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".go") {
			continue
		}

		f := path.Join(c.path, file.Name())
		res, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}

		for k, v := range regexMaps {
			matches := v.FindAllSubmatch(res, -1)
			for mi := range matches {
				m := matches[mi]
				if len(m) == 0 {
					continue
				}
				log.Debug().Msg("found " + k + " in " + file.Name() + " for " + string(m[1]))
				c.data.Store(k, string(m[1]))
			}
		}
	}

	return nil
}

// HasInit will verify if the given struct has a mondoo init function
func (c *Collector) HasInit(structname string) bool {
	return c.data.Exist("init", structname)
}
