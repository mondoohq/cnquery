package stringx

import (
	"bufio"
	"strings"
)

func Indent(indent int, message string) string {
	indentTxt := ""
	for i := 0; i < indent; i++ {
		indentTxt += " "
	}

	var sb strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(message))
	for scanner.Scan() {
		sb.WriteString(indentTxt)
		sb.WriteString(scanner.Text())
		sb.WriteString("\n")
	}
	return sb.String()
}
