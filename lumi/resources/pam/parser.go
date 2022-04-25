package pam

import (
	"fmt"
	"strings"
)

type PamLine struct {
	PamType string
	Control string
	Module  string
	Options []interface{}
}

func ParseLine(line string) (*PamLine, error) {
	line = StripComments(line)

	if line == "" {
		return nil, nil
	}

	fields := strings.Fields(line)

	options := []interface{}{}

	// check if we have @include
	if len(fields) == 2 && fields[0] == "@include" {
		return &PamLine{
			PamType: fields[0],
			Control: fields[1],
			Module:  "",
			Options: options,
		}, nil
	}

	if len(fields) < 3 {
		return &PamLine{}, fmt.Errorf("Invalid pam entry" + line)
	}

	// parse modules

	pamType := fields[0]
	control := fields[1]
	// Control can either be one word or several contained in [] backets
	if control[0] == '[' && control[len(control)-1] != ']' {
		return complicatedParse(fields)
	}
	module := fields[2]

	if len(fields) >= 3 {
		for _, f := range fields[3:] {
			options = append(options, f)
		}
	} else {
		options = nil
	}

	pl := &PamLine{
		PamType: pamType,
		Control: control,
		Module:  module,
		Options: options,
	}
	return pl, nil
}

func complicatedParse(fields []string) (*PamLine, error) {
	pamType := fields[0]
	control := fields[1]
	i := 2
	for ; i < len(fields)-1; i++ {
		str := fields[i]
		control += " " + str
		if str[len(str)-1:] == "]" {
			break
		}
	}
	module := fields[i+1]
	options := []interface{}{}
	if i+2 < len(fields) {
		for _, f := range fields[i+2:] {
			options = append(options, f)
		}
	}
	pl := &PamLine{
		PamType: pamType,
		Control: control,
		Module:  module,
		Options: options,
	}
	return pl, nil
}

func StripComments(line string) string {
	if idx := strings.Index(line, "#"); idx >= 0 {
		line = line[0:idx]
	}
	line = strings.Trim(line, " \t\r")
	return line
}
