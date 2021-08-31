package leise

import (
	"regexp"
	"strings"
)

var leadingWhitespace = regexp.MustCompile(`^\s*`)

func Dedent(content string) string {
	initial := true
	indent := ""

	lines := strings.Split(content, "\n")

	// find max indent
	for i := range lines {
		line := lines[i]
		if line == "" {
			continue
		}
		whitespace := leadingWhitespace.FindString(line)
		if initial || len(indent) > len(whitespace) {
			indent = whitespace
		}
	}

	// cut the whitespace
	result := []string{}
	for i := range lines {
		line := lines[i]
		if line != "" {
			line = strings.TrimPrefix(line, indent)
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
