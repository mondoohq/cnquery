package parsers

import "strings"

// Ini contains the parsed contents of an ini-style file
type Ini struct {
	Fields map[string]interface{}
}

// ParseIni parses the raw text contents of an ini-style file
func ParseIni(raw string, delimiter string) *Ini {
	res := Ini{
		Fields: map[string]interface{}{},
	}

	curGroup := ""
	res.Fields[curGroup] = map[string]interface{}{}

	lines := strings.Split(raw, "\n")
	for i := range lines {
		line := lines[i]
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[0:idx]
		}

		if len(line) == 0 {
			continue
		}

		if line[0] == '[' {
			gEnd := strings.Index(line, "]")
			if gEnd > 0 {
				curGroup = line[1:gEnd]
				res.Fields[curGroup] = map[string]interface{}{}
			}
			continue
		}

		// this is a common accurance on space-separated files
		// we pre-process tabs to make things easier on the tester and allow for
		// space-split mechanisms to still work
		if delimiter != "\t" {
			line = strings.ReplaceAll(line, "\t", "    ")
		}

		kv := strings.SplitN(line, delimiter, 2)
		k := strings.Trim(kv[0], " \t\r")
		if k == "" {
			continue
		}

		var v string
		if len(kv) == 2 {
			v = strings.Trim(kv[1], " \t\r")
		}

		res.Fields[curGroup].(map[string]interface{})[k] = v
	}

	return &res
}
