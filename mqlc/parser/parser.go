// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package parser

import (
	"bytes"
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/participle/lexer"
)

var mqlLexer lexer.Definition

var (
	Ident    rune
	Float    rune
	Int      rune
	String   rune
	Comment  rune
	Regex    rune
	Op       rune
	CallType rune
)

var tokenNames map[rune]string

func init() {
	mqlLexer = lexer.Must(lexer.Regexp(`(\s+)` +
		`|(?P<Ident>[a-zA-Z$_][a-zA-Z0-9_]*)` +
		`|(?P<Float>[-+]?\d*\.\d+([eE][-+]?\d+)?)` +
		`|(?P<Int>[-+]?\d+([eE][-+]?\d+)?)` +
		`|(?P<String>'[^']*'|"[^"]*")` +
		`|(?P<Comment>(//|#)[^\n]*(\n|\z))` +
		`|(?P<Regex>/([^\\/]+|\\.)+/[msi]*)` +
		`|(?P<Op>[-+*/%,:.=<>!|&~;])` +
		`|(?P<Call>[(){}\[\]])`,
	))

	syms := mqlLexer.Symbols()

	Ident = syms["Ident"]
	Float = syms["Float"]
	Int = syms["Int"]
	String = syms["String"]
	Comment = syms["Comment"]
	Regex = syms["Regex"]
	Op = syms["Op"]
	CallType = syms["Call"]

	tokenNames = map[rune]string{
		Ident:   "identifier",
		Float:   "float",
		Int:     "number",
		String:  "string",
		Comment: "comment",
		Regex:   "regex",
		Op:      "operator",
	}
}

// ErrIncomplete points to an incomplete query.
type ErrIncomplete struct {
	missing string
	pos     lexer.Position
	// Indent is a hint for how far we are indented given
	// strict formatting and using only tabs
	Indent int
}

func (e *ErrIncomplete) Error() string {
	return "incomplete query, missing " + e.missing + " at " + e.pos.String()
}

// ErrIncorrect indicates an incorrect symbol was found in a query.
// For example: when users close an opening '(' with a ']'
type ErrIncorrect struct {
	expected string
	got      string
	pos      lexer.Position
}

func (e *ErrIncorrect) Error() string {
	return "expected " + e.expected + ", got '" + e.got + "' at " + e.pos.String()
}

var blockCall string = "{}"

// Expression at the root of mqlc
type Expression struct {
	Operand    *Operand     `json:",omitempty"`
	Operations []*Operation `json:",omitempty"`
}

// IsEmpty expression returns true if we don't contain any action (e.g. comment-only expressions)
func (x *Expression) IsEmpty() bool {
	return len(x.Operations) == 0 && (x.Operand == nil || (x.Operand.Value == nil && len(x.Operand.Calls) == 0 && len(x.Operand.Block) == 0))
}

// Operation has an operator and an operand
type Operation struct {
	Operator Operator
	Operand  *Operand `json:",omitempty"`
}

// Operand is anything that produces a value
type Operand struct {
	Comments string        `json:",omitempty"`
	Value    *Value        `json:",omitempty"`
	Calls    []*Call       `json:",omitempty"`
	Block    []*Expression `json:",omitempty"`
}

// Value representation
type Value struct {
	Bool   *bool                  `json:",omitempty"`
	String *string                `json:",omitempty"`
	Int    *int64                 `json:",omitempty"`
	Float  *float64               `json:",omitempty"`
	Regex  *string                `json:",omitempty"`
	Array  []*Expression          `json:",omitempty"`
	Map    map[string]*Expression `json:",omitempty"`
	Ident  *string                `json:",omitempty"`
}

// Call to a value
type Call struct {
	Comments string      `json:",omitempty"`
	Ident    *string     `json:",omitempty"`
	Function []*Arg      `json:",omitempty"`
	Accessor *Expression `json:",omitempty"`
}

// Arg is a call argument
type Arg struct {
	Name  string
	Value *Expression
}

// AST holds the parsed syntax tree
type AST struct {
	Expressions []*Expression
}

var (
	trueBool  bool = true
	falseBool bool = false
	neverRef       = "Never"

	trueValue     = Value{Bool: &trueBool}
	falseValue    = Value{Bool: &falseBool}
	nilValue      = Value{}
	nanRef        = math.NaN()
	nanValue      = Value{Float: &nanRef}
	infinityRef   = math.Inf(1)
	infinityValue = Value{Float: &infinityRef}
	neverValue    = Value{Ident: &neverRef}
)

type parser struct {
	token      lexer.Token
	nextTokens []lexer.Token
	lex        lexer.Lexer
	comments   bytes.Buffer
	// indent indicates optimal indentation given strict formatting
	// and using only tabs
	indent int
}

// expected generates an error string based on the expected type/field
// and the actual value
func (p *parser) expected(typ string, in string) error {
	name := tokenNames[p.token.Type]
	if name == "" {
		name = "token"
	}
	return p.error("expected "+typ+", got "+name+" \""+p.token.Value+"\"", in)
}

func (p *parser) error(msg string, in string) error {
	return errors.New(msg + " at " + p.token.Pos.String() + " in function " + in)
}

func (p *parser) errorMsg(msg string) error {
	return errors.New(msg + " at " + p.token.Pos.String())
}

// nextToken loads the next token into p.token
func (p *parser) nextToken() error {
	if p.nextTokens == nil {
		var err error

		for {
			p.token, err = p.lex.Next()
			if err != nil {
				return err
			}
			if p.token.Type != Comment {
				break
			}

			p.parseComment()
		}

		return nil
	}

	p.token = p.nextTokens[0]
	if len(p.nextTokens) == 1 {
		p.nextTokens = nil
	} else {
		p.nextTokens = p.nextTokens[1:]
	}

	return nil
}

func (p *parser) parseComment() {
	// we only need the comment's body
	if p.token.Value[0] == '#' {
		if len(p.token.Value) != 1 && p.token.Value[1] == ' ' {
			p.comments.WriteString(strings.TrimRight(p.token.Value[2:], " \t"))
		} else {
			p.comments.WriteString(strings.TrimRight(p.token.Value[1:], " \t"))
		}
	} else {
		if len(p.token.Value) != 2 && p.token.Value[2] == ' ' {
			p.comments.WriteString(strings.TrimRight(p.token.Value[3:], " \t"))
		} else {
			p.comments.WriteString(strings.TrimRight(p.token.Value[2:], " \t"))
		}
	}
}

func (p *parser) flushComments() string {
	if p.comments.Len() == 0 {
		return ""
	}

	res := p.comments.String()
	p.comments.Reset()
	return res
}

// rewind pushes the current token back on the stack and replaces it iwth the given token
func (p *parser) rewind(token lexer.Token) {
	p.nextTokens = append(p.nextTokens, p.token)
	p.token = token
}

var (
	reUnescape  = regexp.MustCompile("\\\\.")
	unescapeMap = map[string]string{
		"\\n": "\n",
		"\\t": "\t",
		"\\v": "\v",
		"\\b": "\b",
		"\\f": "\f",
		"\\0": "\x00",
	}
)

func (p *parser) token2string() string {
	v := p.token.Value
	vv := v[1 : len(v)-1]

	if v[0] == '\'' {
		return vv
	}

	vv = reUnescape.ReplaceAllStringFunc(vv, func(match string) string {
		if found := unescapeMap[match]; found != "" {
			return found
		}
		return string(match[1])
	})
	return vv
}

func (p *parser) parseValue() (*Value, error) {
	switch p.token.Type {
	case Ident:
		switch p.token.Value {
		case "true":
			return &trueValue, nil
		case "false":
			return &falseValue, nil
		case "null":
			return &nilValue, nil
		case "NaN":
			return &nanValue, nil
		case "Infinity":
			return &infinityValue, nil
		case "Never":
			return &neverValue, nil
		default:
			v := p.token.Value
			return &Value{Ident: &v}, nil
		}

	case Float:
		v, err := strconv.ParseFloat(p.token.Value, 64)
		if err != nil {
			return nil, p.errorMsg("failed to parse float: " + err.Error())
		}
		return &Value{Float: &v}, nil

	case Int:
		var v int64
		var err error
		if p.token.Value[0] == '0' {
			v, err = strconv.ParseInt(p.token.Value, 8, 64)
		} else {
			v, err = strconv.ParseInt(p.token.Value, 10, 64)
		}

		if err != nil {
			return nil, p.errorMsg("failed to parse integer: " + err.Error())
		}
		return &Value{Int: &v}, nil

	case String:
		vv := p.token2string()
		return &Value{String: &vv}, nil

	case Regex:
		v := p.token.Value

		reEnd := len(v) - 1
		for ; reEnd > 1; reEnd-- {
			if v[reEnd] == '/' {
				break
			}
		}

		// TODO: handling of escape sequences
		vv := v[1:reEnd]
		mods := v[reEnd+1:]

		if mods != "" {
			vv = "(?" + mods + ")" + vv
		}

		return &Value{Regex: &vv}, nil

	}
	return nil, nil
}

func (p *parser) parseArg() (*Arg, error) {
	res := Arg{}

	if p.token.Type == Ident {
		name := p.token
		p.nextToken()

		if p.token.Value == ":" {
			p.nextToken()
			res.Name = name.Value
		} else {
			p.rewind(name)
		}
	}

	exp, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if exp == nil {
		if res.Name != "" {
			return nil, p.expected("argument", "parseArgument")
		}
		return nil, nil
	}
	res.Value = exp
	return &res, nil
}

func (p *parser) parseArray() (*Value, error) {
	res := Value{Array: []*Expression{}}

	p.nextToken()
	if p.token.Value == "]" {
		return &res, nil
	}

	for {
		exp, err := p.parseExpression()
		if exp == nil {
			return nil, p.expected("expression", "parseOperand-array")
		}
		if err != nil {
			return nil, err
		}
		res.Array = append(res.Array, exp)

		if p.token.Value == "]" {
			break
		}
		if p.token.Value != "," {
			return nil, p.expected(", or ]", "parseOperand")
		}

		p.nextToken()

		// catch trailing commas, ie: [a, b, c, ]
		if p.token.Value == "]" {
			break
		}
	}

	return &res, nil
}

func (p *parser) parseMap() (*Value, error) {
	res := Value{
		Map: map[string]*Expression{},
	}

	p.nextToken()
	if p.token.Value == "}" {
		return &res, nil
	}

	for {
		var key string

		switch p.token.Type {
		case String:
			key = p.token2string()
		case Ident:
			key = p.token.Value
		default:
			return nil, p.expected("string", "map key")
		}

		p.nextToken()
		if p.token.Value != ":" || p.token.Type != Op {
			return nil, p.expected(":", "after map key")
		}

		p.nextToken()
		exp, err := p.parseExpression()
		if exp == nil {
			return nil, p.expected("expression", "parseOperand-map")
		}
		if err != nil {
			return nil, err
		}
		res.Map[key] = exp

		if p.token.Value == "}" {
			break
		}
		if p.token.Value != "," {
			return nil, p.expected(", or }", "parseOperand")
		}

		p.nextToken()

		// catch trailing commas, ie: {a: 1,}
		if p.token.Value == "}" {
			break
		}
	}

	return &res, nil
}

// parseOperand and return the operand, and true if the operand is standalone
func (p *parser) parseOperand() (*Operand, bool, error) {
	// operand:      value [ call | accessor | '.' ident ]+ [ block ]
	value, err := p.parseValue()
	if err != nil {
		return nil, false, err
	}
	if value == nil {
		// arrays
		if p.token.Value == "[" {
			value, err = p.parseArray()
			if err != nil {
				return nil, false, err
			}
		}

		// maps
		if p.token.Value == "{" {
			value, err = p.parseMap()
			if err != nil {
				return nil, false, err
			}
		}

		// glob all fields of a resource
		// ie: resource { * }
		if p.token.Value == "*" {
			p.nextToken()
			star := "*"
			return &Operand{
				Value: &Value{
					Ident: &star,
				},
			}, true, nil
		}
	}

	if value == nil {
		return nil, false, nil
	}

	if value.Ident != nil && *value.Ident == "return" {
		p.nextToken()
		return &Operand{Value: value}, true, nil
	}

	res := Operand{
		Comments: p.flushComments(),
		Value:    value,
	}
	p.nextToken()

	for {
		switch p.token.Value {
		case ".":
			p.nextToken()

			// everything else must be an identifier
			if p.token.Type != Ident {
				v := "."
				res.Calls = append(res.Calls, &Call{Ident: &v})

				if p.token.EOF() {
					p.indent++
					return &res, false, &ErrIncomplete{missing: "identifier after '.'", pos: p.token.Pos, Indent: p.indent}
				}

				return &res, false, p.errorMsg("missing field accessor")
			}

			v := p.token.Value
			res.Calls = append(res.Calls, &Call{
				Ident:    &v,
				Comments: p.flushComments(),
			})
			p.nextToken()

		case "(":
			p.indent++
			p.nextToken()
			args := []*Arg{}

			for {
				arg, err := p.parseArg()
				if err != nil {
					return nil, false, err
				}
				if arg == nil {
					break
				}
				args = append(args, arg)

				if p.token.Value == "," {
					p.nextToken()
				}
			}

			if p.token.Value != ")" {
				if p.token.EOF() {
					return nil, false, &ErrIncomplete{missing: "closing ')'", pos: p.token.Pos, Indent: p.indent}
				}
				return nil, false, &ErrIncorrect{expected: "closing ')'", got: p.token.Value, pos: p.token.Pos}
			}

			p.indent--
			res.Calls = append(res.Calls, &Call{Function: args})
			p.nextToken()

		case "[":
			p.indent++
			p.nextToken()

			exp, err := p.parseExpression()
			if err != nil {
				return nil, false, err
			}

			if p.token.Value != "]" {
				if p.token.EOF() {
					return nil, false, &ErrIncomplete{missing: "closing ']'", pos: p.token.Pos, Indent: p.indent}
				}
				return nil, false, &ErrIncorrect{expected: "closing ']'", got: p.token.Value, pos: p.token.Pos}
			}

			p.indent--
			if exp == nil {
				return nil, false, p.errorMsg("missing value inside of `[]`")
			}
			res.Calls = append(res.Calls, &Call{
				Accessor: exp,
			})
			p.nextToken()

		case "{":
			p.indent++
			if res.Value.Ident != nil && *res.Value.Ident == "switch" {
				p.nextToken()

				for {
					ident := p.token.Value
					if ident == "}" {
						break
					}

					if ident != "case" && ident != "default" {
						return nil, false, errors.New("expected `case` or `default` statements in `switch` call, got `" + ident + "`")
					}
					p.nextToken()

					if ident == "case" {
						exp, err := p.parseExpression()
						if err != nil {
							return nil, false, err
						}
						if exp == nil {
							return nil, false, errors.New("missing expression after `case` statement")
						}
						if err = exp.processOperators(); err != nil {
							return nil, false, err
						}
						if exp == nil || (exp.Operand == nil && exp.Operations == nil) {
							return nil, false, errors.New("missing expression after `case` statement")
						}
						res.Block = append(res.Block, exp)
					} else {
						// we still need to add the empty condition block
						res.Block = append(res.Block, nil)
					}

					if p.token.Value != ":" {
						return nil, false, errors.New("expected `:` in `" + ident + "` statement")
					}
					p.nextToken()

					block := Expression{
						Operand: &Operand{
							Value: &Value{
								Ident: &blockCall,
							},
						},
					}
					for {
						exp, err := p.parseExpression()
						if err != nil {
							return nil, false, err
						}
						if exp == nil || (exp.Operand == nil && exp.Operations == nil) {
							break
						}
						block.Operand.Block = append(block.Operand.Block, exp)
					}

					if len(block.Operand.Block) == 0 {
						return nil, false, errors.New("expected block following `" + ident + "` statement")
					}
					res.Block = append(res.Block, &block)

					for p.token.Value == ";" {
						p.nextToken()
					}
				}

				p.nextToken()
				continue
			}

			p.nextToken()
			block := []*Expression{}

			for {
				exp, err := p.parseExpression()
				if err != nil {
					return nil, false, err
				}
				if exp == nil || (exp.Operand == nil && exp.Operations == nil) {
					break
				}
				block = append(block, exp)
			}

			res.Block = block

			if p.token.Value != "}" {
				if p.token.EOF() {
					return &res, false, &ErrIncomplete{missing: "closing '}'", pos: p.token.Pos, Indent: p.indent}
				}
				return &res, false, &ErrIncorrect{expected: "closing '}'", got: p.token.Value, pos: p.token.Pos}
			}

			p.indent--
			p.nextToken()

		default:
			return &res, false, nil
		}
	}
}

func (p *parser) parseOperation() (*Operation, error) {
	if p.token.Type != Op {
		return nil, nil
	}

	res := Operation{}
	switch p.token.Value {
	case ";":
		return nil, nil
	case ":":
		return nil, nil
	case "&":
		p.nextToken()
		if p.token.Value == "&" {
			res.Operator = OpAnd
			p.nextToken()
		} else {
			return nil, p.expected("&&", "parseOperation")
		}
	case "|":
		p.nextToken()
		if p.token.Value == "|" {
			res.Operator = OpOr
			p.nextToken()
		} else {
			return nil, p.expected("||", "parseOperation")
		}
	case "=":
		p.nextToken()
		if p.token.Value == "=" {
			res.Operator = OpEqual
			p.nextToken()
		} else if p.token.Value == "~" {
			res.Operator = OpCmp
			p.nextToken()
		} else {
			res.Operator = OpAssignment
		}
	case "!":
		p.nextToken()
		if p.token.Value == "=" {
			res.Operator = OpNotEqual
			p.nextToken()
		} else if p.token.Value == "~" {
			res.Operator = OpNotCmp
			p.nextToken()
		} else {
			return nil, p.expected("!= or !~", "parseOperation")
		}
	case "<":
		p.nextToken()
		if p.token.Value == "=" {
			res.Operator = OpSmallerEqual
			p.nextToken()
		} else {
			res.Operator = OpSmaller
		}
	case ">":
		p.nextToken()
		if p.token.Value == "=" {
			res.Operator = OpGreaterEqual
			p.nextToken()
		} else {
			res.Operator = OpGreater
		}
	case "+":
		res.Operator = OpAdd
		p.nextToken()
	case "-":
		res.Operator = OpSubtract
		p.nextToken()
	case "*":
		res.Operator = OpMultiply
		p.nextToken()
	case "/":
		res.Operator = OpDivide
		p.nextToken()
	case "%":
		res.Operator = OpRemainder
		p.nextToken()
	default:
		return nil, errors.New("found unexpected operation '" + p.token.Value + "'")
	}

	op, _, err := p.parseOperand()
	if err != nil {
		return nil, err
	}
	if op == nil {
		return nil, p.expected("operand", "parseOperation")
	}

	res.Operand = op
	return &res, nil
}

func (p *parser) flushExpression() *Expression {
	if p.comments.Len() == 0 {
		return nil
	}

	return &Expression{
		Operand: &Operand{
			Comments: p.flushComments(),
		},
	}
}

func (p *parser) parseExpression() (*Expression, error) {
	if p.token.EOF() {
		return p.flushExpression(), nil
	}

	res := Expression{}
	var err error
	var standalone bool

	// expression:   operand [ op operand ]+
	res.Operand, standalone, err = p.parseOperand()
	if err != nil {
		return &res, err
	}
	if standalone {
		return &res, err
	}

	var operation *Operation
	for {
		if p.token.Value == "," {
			break
		}

		operation, err = p.parseOperation()
		if operation == nil {
			break
		}
		res.Operations = append(res.Operations, operation)
	}

	if res.Operand == nil && res.Operations == nil {
		return p.flushExpression(), err
	}

	return &res, err
}

// Parse an input string into an AST
func Parse(input string) (*AST, error) {
	lex, err := mqlLexer.Lex(strings.NewReader(input))
	if err != nil {
		return nil, err
	}
	res := AST{}

	thisParser := parser{
		lex: lex,
	}

	err = thisParser.nextToken()
	if err != nil {
		return nil, err
	}
	if thisParser.token.EOF() {
		return &res, nil
	}

	var exp *Expression
	for {
		exp, err = thisParser.parseExpression()
		if exp == nil {
			break
		}

		res.Expressions = append(res.Expressions, exp)
		if err != nil {
			break
		}

		if thisParser.token.Value == ";" {
			err = thisParser.nextToken()
			if err != nil {
				return &res, err
			}
		}

		if thisParser.token.Value != "" && thisParser.token.Type == CallType && thisParser.token.Value != "[" && thisParser.token.Value != "{" {
			return &res, errors.New("mismatched symbol '" + thisParser.token.Value + "' at the end of expression")
		}
	}

	return &res, err
}

// Lex the input mqlc string to a list of tokens
func Lex(input string) ([]lexer.Token, error) {
	res := []lexer.Token{}
	lex, err := mqlLexer.Lex(strings.NewReader(input))
	if err != nil {
		return res, err
	}

	token, err := lex.Next()
	if err != nil {
		return res, err
	}

	for !token.EOF() {
		if token.Type != Comment {
			res = append(res, token)
		}

		token, err = lex.Next()
		if err != nil {
			return res, err
		}
	}
	return res, nil
}
