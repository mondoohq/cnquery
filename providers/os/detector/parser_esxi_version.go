package detector

import (
	"fmt"
	"regexp"
)

var EsxiReleaseRegex = regexp.MustCompile(`^VMware ESXi\s(.*)\s*$`)

func ParseEsxiRelease(content string) (string, error) {
	m := EsxiReleaseRegex.FindStringSubmatch(content)
	if len(m) < 1 {
		return "", fmt.Errorf("could not parse esxi version: %s", content)
	}
	return m[1], nil
}
