package parsers

import "strings"

// Ini contains the parsed contents of an ini-style file
type Ini struct {
	Fields map[string]map[string]string
}

// ParseIni parses the raw text contents of an ini-style file
func ParseIni(raw string) *Ini {
	res := Ini{
		Fields: map[string]map[string]string{},
	}

	curGroup := ""
	res.Fields[curGroup] = map[string]string{}

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
				res.Fields[curGroup] = map[string]string{}
			}
			continue
		}

		kv := strings.SplitN(line, "=", 2)
		k := strings.Trim(kv[0], " \t\r")
		if k == "" {
			continue
		}

		var v string
		if len(kv) == 2 {
			v = strings.Trim(kv[1], " \t\r")
		}

		res.Fields[curGroup][k] = v
	}

	return &res
}
