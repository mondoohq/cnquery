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
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return &PamLine{}, fmt.Errorf("Invalid pam entry" + line)
	}
	pamType := fields[0]
	control := fields[1]
	//Control can either be one word or several contained in [] backets
	if control[:1] == "[" {
		return complicatedParse(fields)
	}
	module := fields[2]
	options := []interface{}{}
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

func ToGenericArray(arr ...interface{}) []interface{} {
	return arr
}
