package stringx

import (
	"bufio"
	"strconv"
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

func MaxLines(lines int, message string) string {
	res := strings.Split(message, "\n")
	if len(res) <= lines {
		return message
	}

	n := len(res) - lines

	return strings.Join(res[:lines], "\n") + "\n... " + strconv.Itoa(n) + " more lines ..."
}
