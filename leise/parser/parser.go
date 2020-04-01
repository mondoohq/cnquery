package parser

import (
	"errors"
	"strconv"
	"strings"

	"github.com/alecthomas/participle/lexer"
)

var (
	leiseLexer = lexer.Must(lexer.Regexp(`(\s+)` +
		`|(?P<Ident>[a-zA-Z_][a-zA-Z0-9_]*)` +
		`|(?P<Float>[-+]?\d*\.\d+([eE][-+]?\d+)?)` +
		`|(?P<Int>[-+]?\d+([eE][-+]?\d+)?)` +
		`|(?P<String>'[^']*'|"[^"]*")` +
		`|(?P<Regex>/([^/]+|\\.)*/)` +
		`|(?P<Op>[-+*/%,:.=<>!|~])` +
		`|(?P<Call>[(){}\[\]])`,
	))
)

const (
	Ident  rune = -3
	Float  rune = -4
	Int    rune = -6
	String rune = -8
	Regex  rune = -9
	Op     rune = -11
)

var tokenNames = map[rune]string{
	Ident:  "identifier",
	Float:  "float",
	Int:    "number",
	String: "string",
	Regex:  "regex",
	Op:     "operator",
}

// Expression at the root of leise
type Expression struct {
	Operand    *Operand     `json:",omitempty"`
	Operations []*Operation `json:",omitempty"`
}

// Operation has an operator and an operand
type Operation struct {
	Operator Operator
	Operand  *Operand `json:",omitempty"`
}

// Operand is anything that produces a value
type Operand struct {
	Value *Value        `json:",omitempty"`
	Calls []*Call       `json:",omitempty"`
	Block []*Expression `json:",omitempty"`
}

// Value representation
type Value struct {
	Bool   *bool         `json:",omitempty"`
	String *string       `json:",omitempty"`
	Int    *int64        `json:",omitempty"`
	Float  *float64      `json:",omitempty"`
	Regex  *string       `json:",omitempty"`
	Array  []*Expression `json:",omitempty"`
	Ident  *string       `json:",omitempty"`
}

// Call to a value
type Call struct {
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

var trueBool = true
var falseBool = false
var trueValue = Value{Bool: &trueBool}
var falseValue = Value{Bool: &falseBool}

type parser struct {
	token      lexer.Token
	nextTokens []lexer.Token
	lex        lexer.Lexer
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

// nextToken loads the next token into p.token
func (p *parser) nextToken() error {
	if p.nextTokens == nil {
		var err error
		p.token, err = p.lex.Next()
		return err
	}

	p.token = p.nextTokens[0]
	if len(p.nextTokens) == 1 {
		p.nextTokens = nil
	} else {
		p.nextTokens = p.nextTokens[1:]
	}

	return nil
}

// rewind pushes the current token back on the stack and replaces it iwth the given token
func (p *parser) rewind(token lexer.Token) {
	p.nextTokens = append(p.nextTokens, p.token)
	p.token = token
}

func (p *parser) parseValue() *Value {
	switch p.token.Type {
	case Ident:
		if p.token.Value == "true" {
			return &trueValue
		}

		if p.token.Value == "false" {
			return &falseValue
		}

		v := p.token.Value
		return &Value{Ident: &v}

	case Float:
		v, err := strconv.ParseFloat(p.token.Value, 64)
		if err != nil {
			panic("Failed to parse float: " + err.Error())
		}
		return &Value{Float: &v}

	case Int:
		v, err := strconv.ParseInt(p.token.Value, 10, 64)
		if err != nil {
			panic("Failed to parse integer: " + err.Error())
		}
		return &Value{Int: &v}

	case String:
		v := p.token.Value
		// TODO: handling of escape sequences
		vv := v[1 : len(v)-1]
		return &Value{String: &vv}

	case Regex:
		v := p.token.Value
		// TODO: handling of escape sequences
		vv := v[1 : len(v)-1]
		return &Value{Regex: &vv}

	}
	return nil
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
	}

	return &res, nil
}

func (p *parser) parseOperand() (*Operand, error) {
	// operand:      value [ call | accessor | '.' ident ]+ [ block ]
	value := p.parseValue()
	var err error

	if value == nil {
		// arrays
		if p.token.Value == "[" {
			value, err = p.parseArray()
			if err != nil {
				return nil, err
			}
		}
	}

	if value == nil {
		return nil, nil
	}

	res := Operand{
		Value: value,
	}
	p.nextToken()

	for {
		switch p.token.Value {
		case ".":
			p.nextToken()
			if p.token.Type != Ident {
				return nil, p.expected("identifier", "parseOperand-call")
			}
			v := p.token.Value
			res.Calls = append(res.Calls, &Call{Ident: &v})
			p.nextToken()

		case "(":
			p.nextToken()
			args := []*Arg{}

			for {
				arg, err := p.parseArg()
				if err != nil {
					return nil, err
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
				return nil, p.expected("')'", "parseOperand-function")
			}

			res.Calls = append(res.Calls, &Call{Function: args})
			p.nextToken()

		case "[":
			p.nextToken()

			exp, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if exp == nil {
				return nil, p.error("missing value in accessor", "parseOperand-accessor")
			}

			if p.token.Value != "]" {
				return nil, p.expected("']'", "parseOperand-accessor")
			}
			res.Calls = append(res.Calls, &Call{
				Accessor: exp,
			})
			p.nextToken()

		case "{":
			p.nextToken()
			block := []*Expression{}

			for {
				exp, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				if exp == nil {
					break
				}
				block = append(block, exp)
			}

			if p.token.Value != "}" {
				return nil, p.expected("'}'", "parseOperand-block")
			}

			res.Block = block
			p.nextToken()

		default:
			return &res, nil
		}
	}
}

func (p *parser) parseOperation() (*Operation, error) {
	if p.token.Type != Op {
		return nil, nil
	}

	res := Operation{}
	switch p.token.Value {
	// TODO:	"=":  OpAssignment,
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
			res.Operator = OpAnd
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
			return nil, p.expected("== or =~", "parseOperation")
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
		return nil, nil
	}

	op, err := p.parseOperand()
	if err != nil {
		return nil, err
	}
	if op == nil {
		return nil, p.expected("operand", "parseOperation")
	}

	res.Operand = op
	return &res, nil
}

func (p *parser) parseExpression() (*Expression, error) {
	if p.token.EOF() {
		return nil, nil
	}

	res := Expression{}
	var err error

	// expression:   operand [ op operand ]+
	res.Operand, err = p.parseOperand()
	if res.Operand == nil {
		return nil, err
	}

	var operation *Operation
	for {
		operation, err = p.parseOperation()
		if operation == nil {
			break
		}
		res.Operations = append(res.Operations, operation)
	}
	return &res, err
}

// Parse an input string into an AST
func Parse(input string) (*AST, error) {
	lex, err := leiseLexer.Lex(strings.NewReader(input))
	if err != nil {
		return nil, err
	}
	res := AST{}

	token, err := lex.Next()
	if err != nil {
		return nil, err
	}

	thisParser := parser{
		lex:   lex,
		token: token,
	}

	var exp *Expression
	for {
		exp, err = thisParser.parseExpression()
		if err != nil {
			return nil, err
		}
		if exp == nil {
			break
		}
		res.Expressions = append(res.Expressions, exp)
	}
	return &res, nil
}

// Lex the input leise string to a list of tokens
func Lex(input string) ([]lexer.Token, error) {
	res := []lexer.Token{}
	lex, err := leiseLexer.Lex(strings.NewReader(input))
	if err != nil {
		return res, err
	}

	token, err := lex.Next()
	if err != nil {
		return res, err
	}

	for !token.EOF() {
		res = append(res, token)
		token, err = lex.Next()
		if err != nil {
			return res, err
		}
	}
	return res, nil
}
