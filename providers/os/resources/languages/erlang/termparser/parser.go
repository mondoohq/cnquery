// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package termparser provides a lightweight recursive descent parser for
// Erlang text term syntax, as used in rebar.lock files. It handles lists,
// tuples, strings, binary strings (<<"...">>), atoms, numbers, and comments.
package termparser

import (
	"fmt"
	"strings"
	"unicode"
)

// Node represents a parsed Erlang term.
type Node struct {
	// Type of this node.
	Type NodeType
	// Value for leaf nodes (string, atom, number).
	Value string
	// Children for compound nodes (list, tuple).
	Children []*Node
}

// NodeType identifies the kind of Erlang term.
type NodeType int

const (
	NodeList   NodeType = iota // [...]
	NodeTuple                  // {...}
	NodeString                 // "..." or <<"...">>
	NodeAtom                   // bare atom or 'quoted atom'
	NodeNumber                 // integer or float
)

// Get returns the i-th child, or nil if out of range.
func (n *Node) Get(i int) *Node {
	if n == nil || i < 0 || i >= len(n.Children) {
		return nil
	}
	return n.Children[i]
}

// Str returns the string value of this node. For compound nodes, returns "".
func (n *Node) Str() string {
	if n == nil {
		return ""
	}
	return n.Value
}

// Len returns the number of children.
func (n *Node) Len() int {
	if n == nil {
		return 0
	}
	return len(n.Children)
}

// parser holds the parsing state.
type parser struct {
	input []rune
	pos   int
}

// Parse parses an Erlang term from the given string and returns the root node.
func Parse(input string) (*Node, error) {
	p := &parser{input: []rune(input)}
	p.skipWhitespaceAndComments()
	node, err := p.parseNode()
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (p *parser) parseNode() (*Node, error) {
	p.skipWhitespaceAndComments()
	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("unexpected end of input")
	}

	ch := p.input[p.pos]

	switch {
	case ch == '[':
		return p.parseList()
	case ch == '{':
		return p.parseTuple()
	case ch == '"':
		return p.parseQuotedString('"')
	case ch == '\'':
		return p.parseQuotedString('\'')
	case ch == '<' && p.peek(1) == '<':
		return p.parseBinaryString()
	case ch == '-' || (ch >= '0' && ch <= '9'):
		return p.parseNumber()
	default:
		return p.parseAtom()
	}
}

func (p *parser) parseList() (*Node, error) {
	p.pos++ // skip '['
	node := &Node{Type: NodeList}

	for {
		p.skipWhitespaceAndComments()
		if p.pos >= len(p.input) {
			return nil, fmt.Errorf("unterminated list")
		}
		if p.input[p.pos] == ']' {
			p.pos++
			break
		}

		child, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, child)

		p.skipWhitespaceAndComments()
		if p.pos < len(p.input) && p.input[p.pos] == ',' {
			p.pos++
		}
	}

	return node, nil
}

func (p *parser) parseTuple() (*Node, error) {
	p.pos++ // skip '{'
	node := &Node{Type: NodeTuple}

	for {
		p.skipWhitespaceAndComments()
		if p.pos >= len(p.input) {
			return nil, fmt.Errorf("unterminated tuple")
		}
		if p.input[p.pos] == '}' {
			p.pos++
			break
		}

		child, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, child)

		p.skipWhitespaceAndComments()
		if p.pos < len(p.input) && p.input[p.pos] == ',' {
			p.pos++
		}
	}

	return node, nil
}

func (p *parser) parseQuotedString(quote rune) (*Node, error) {
	p.pos++ // skip opening quote
	var sb strings.Builder

	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if ch == '\\' && p.pos+1 < len(p.input) {
			// Escape sequence
			p.pos++
			sb.WriteRune(p.input[p.pos])
			p.pos++
			continue
		}
		if ch == quote {
			p.pos++
			typ := NodeString
			if quote == '\'' {
				typ = NodeAtom
			}
			return &Node{Type: typ, Value: sb.String()}, nil
		}
		sb.WriteRune(ch)
		p.pos++
	}

	return nil, fmt.Errorf("unterminated string")
}

// parseBinaryString handles <<"...">> syntax.
func (p *parser) parseBinaryString() (*Node, error) {
	p.pos += 2 // skip '<<'
	p.skipWhitespaceAndComments()

	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("unterminated binary string")
	}

	// Expect a quoted string inside
	if p.input[p.pos] != '"' {
		// Non-string binary — skip to >>
		return p.skipToCloseBinary()
	}

	node, err := p.parseQuotedString('"')
	if err != nil {
		return nil, err
	}

	p.skipWhitespaceAndComments()

	// Expect '>>'
	if p.pos+1 < len(p.input) && p.input[p.pos] == '>' && p.input[p.pos+1] == '>' {
		p.pos += 2
	}

	return node, nil
}

func (p *parser) skipToCloseBinary() (*Node, error) {
	var sb strings.Builder
	for p.pos < len(p.input) {
		if p.input[p.pos] == '>' && p.pos+1 < len(p.input) && p.input[p.pos+1] == '>' {
			p.pos += 2
			return &Node{Type: NodeString, Value: sb.String()}, nil
		}
		sb.WriteRune(p.input[p.pos])
		p.pos++
	}
	return nil, fmt.Errorf("unterminated binary")
}

func (p *parser) parseNumber() (*Node, error) {
	start := p.pos
	if p.input[p.pos] == '-' {
		p.pos++
	}
	for p.pos < len(p.input) && (p.input[p.pos] >= '0' && p.input[p.pos] <= '9' || p.input[p.pos] == '.') {
		p.pos++
	}
	return &Node{Type: NodeNumber, Value: string(p.input[start:p.pos])}, nil
}

func (p *parser) parseAtom() (*Node, error) {
	start := p.pos
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '@' || ch == '.' {
			p.pos++
		} else {
			break
		}
	}
	if p.pos == start {
		return nil, fmt.Errorf("unexpected character: %c at position %d", p.input[p.pos], p.pos)
	}
	return &Node{Type: NodeAtom, Value: string(p.input[start:p.pos])}, nil
}

func (p *parser) skipWhitespaceAndComments() {
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if ch == '%' {
			// Skip to end of line
			for p.pos < len(p.input) && p.input[p.pos] != '\n' {
				p.pos++
			}
			continue
		}
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			p.pos++
			continue
		}
		// Skip trailing dot (end of term)
		if ch == '.' {
			p.pos++
			continue
		}
		break
	}
}

func (p *parser) peek(offset int) rune {
	idx := p.pos + offset
	if idx >= len(p.input) {
		return 0
	}
	return p.input[idx]
}
