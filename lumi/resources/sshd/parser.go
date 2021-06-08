package sshd

import (
	"errors"
	"strings"
)

type SshdLine struct {
	key  string
	args string
}

// ParseLine parses a single line of the sshd_config file
// A valid line contains a key. If the entire line is a
// comment, key will be empty. If the line could not be
// parsed, an error is returned.
//
// Note: The caller must provide only a single line.
func ParseLine(s []rune) (SshdLine, error) {
	l := SshdLine{}
	if len(s) == 0 {
		return l, nil
	}

	l.key, s = parseKeyword(s)

	var err error
	l.args, s, err = parseArgs(s)
	if err != nil {
		return l, err
	}

	return l, nil
}

func parseKeyword(s []rune) (string, []rune) {
	s = consumeWhitespace(s)

	i := 0
LOOP:
	for {
		if i >= len(s) {
			break
		}

		switch s[i] {
		case '#':
			break LOOP
		case ' ', '\t', '\r', '\n', '=':
			break LOOP
		}
		i++
	}

	return string(s[0:i]), consumeEqualOrWhitespace(s[i:])
}

func parseArgs(s []rune) (string, []rune, error) {
	args := []string{}
LOOP:
	for len(s) > 0 {
		var arg string

		switch s[0] {
		case '#':
			break LOOP
		case '"':
			var err error
			arg, s, err = parseQuotedArg(s)
			if err != nil {
				return "", nil, err
			}
			args = append(args, arg)
		default:
			arg, s = parseArg(s)
			args = append(args, arg)
		}
	}

	return strings.Join(args, " "), s, nil
}

func parseArg(s []rune) (string, []rune) {
	i := 0
LOOP:
	for i < len(s) {
		switch s[i] {
		case ' ', '\t', '\r', '\n':
			break LOOP
		}
		i++
	}
	return string(s[0:i]), consumeWhitespace(s[i:])
}

var errUnexpectedEOF = errors.New("Unexpected EOF")

func parseQuotedArg(s []rune) (string, []rune, error) {
	i := 1
LOOP:
	for {
		if i >= len(s) {
			return "", nil, errUnexpectedEOF
		}

		switch s[i] {
		case '\\':
			i++
			if i < len(s) && s[i] == '"' {
				i++
			}
		case '\r', '\n':
			return "", nil, errUnexpectedEOF
		case '"':
			i++
			break LOOP
		}
		i++
	}
	return string(s[0:i]), consumeWhitespace(s[i:]), nil
}

func consumeWhitespace(s []rune) []rune {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t', '\r', '\n':
		default:
			return s[i:]
		}
	}
	return []rune{}
}

func consumeEqualOrWhitespace(s []rune) []rune {
	s = consumeWhitespace(s)
	if len(s) > 0 && s[0] == '=' {
		s = consumeWhitespace(s[1:])
	}
	return s
}
