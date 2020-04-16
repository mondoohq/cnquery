package logindefs

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

func Parse(r io.Reader) map[string]string {
	res := map[string]string{}

	// ignore line if it starts with a comment
	logindefEntry := regexp.MustCompile(`^\s*([^#]\S+)\s+(\S+)\s*$`)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		noWhitespace := strings.TrimSpace(line)

		m := logindefEntry.FindStringSubmatch(noWhitespace)
		if len(m) == 3 {
			res[m[1]] = m[2]
		}
	}

	return res
}
