package npm

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"regexp"
	"strings"

	"go.mondoo.com/cnquery/vadvisor"

	"github.com/rs/zerolog/log"
	"sigs.k8s.io/yaml"
)

type YarnLockEntry struct {
	Version      string
	Resolved     string
	Dependencies map[string]string
}

func ParseYarnLock(r io.Reader) ([]*vadvisor.Package, error) {
	var b bytes.Buffer

	// iterate and convert the format to yaml on the fly
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		reStr := regexp.MustCompile(`^(\s*.*)\s\"(.*)$`)
		repStr := "${1}: \"$2"
		line = reStr.ReplaceAllString(line, repStr)

		b.Write([]byte(line))
		b.Write([]byte("\n"))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var yarnLock map[string]YarnLockEntry

	err := yaml.Unmarshal(b.Bytes(), &yarnLock)
	if err != nil {
		return nil, err
	}

	entries := []*vadvisor.Package{}

	// add all dependencies
	for k, v := range yarnLock {
		name, _, err := ParseYarnPackageName(k)
		if err != nil {
			log.Error().Str("name", name).Msg("cannot parse yarn package name")
			continue
		}
		entries = append(entries, &vadvisor.Package{
			Name:      name,
			Version:   v.Version,
			Format:    "npm",
			Namespace: "nodejs",
		})
	}

	return entries, nil
}

func ParseYarnPackageName(name string) (string, string, error) {
	// a yarn package line may include may items
	pkgNames := strings.Split(name, ",")

	if len(pkgNames) == 0 {
		// something wrong
		return "", "", errors.New("cannot parse yarn package name")
	}

	parse := regexp.MustCompile(`^(.*)@(.*)$`)
	m := parse.FindStringSubmatch(strings.TrimSpace(pkgNames[0]))
	return m[1], m[2], nil
}
