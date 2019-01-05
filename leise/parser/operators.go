package parser

import "errors"

// Operator list
type Operator int

const (
	// strictly following the javascript operator precedence
	OpAssignment Operator = iota + 30
	OpAnd        Operator = iota + 60
	OpOr         Operator = iota + 70
	OpEqual      Operator = iota + 100
	OpCmp
	OpNotEqual
	OpNotCmp
	OpSmaller Operator = iota + 110
	OpSmallerEqual
	OpGreater
	OpGreaterEqual
	OpAdd Operator = iota + 130
	OpSubtract
	OpMultiply Operator = iota + 140
	OpDivide
	OpRemainder
)

var operatorsMap = map[string]Operator{
	"=":  OpAssignment,
	"&&": OpAnd,
	"||": OpOr,
	"==": OpEqual,
	"=~": OpCmp,
	"!=": OpNotEqual,
	"!~": OpNotCmp,
	"<":  OpSmaller,
	"<=": OpSmallerEqual,
	">":  OpGreater,
	">=": OpGreaterEqual,
	"+":  OpAdd,
	"-":  OpSubtract,
	"*":  OpMultiply,
	"/":  OpDivide,
	"%":  OpRemainder,
}

var operatorsStrings map[Operator]string

func init() {
	operatorsStrings = make(map[Operator]string)
	for k, v := range operatorsMap {
		operatorsStrings[v] = k
	}
}

// Capture an operator in participle
func (o *Operator) Capture(s []string) error {
	// capture both tokens
	sop := s[0]
	if len(s) == 2 {
		sop += s[1]
	}

	*o = operatorsMap[sop]
	return nil
}

func (o *Operator) String() string {
	r, ok := operatorsStrings[*o]
	if !ok {
		return "unknown operator"
	}
	return r
}

// processOperators handles simple ops like ==, >=, *, ...
// and turns them into functions; only on the level of this expression and its
// Operations children, not deeper funtion calls
func (e *Expression) processOperators() error {
	if len(e.Operations) == 0 {
		return nil
	}

	if e.Operand == nil {
		return errors.New("expression doesn't have any operand, i.e. it has been parsed without a body")
	}

	// group operators into precedence (matchOp as the cut-off line)
	// check if every value has an operator (since they are all linked)
	// and cycle through all calls and process their parameters too
	maxOp := 0
	for idx := range e.Operations {
		v := e.Operations[idx]
		if maxOp < int(v.Operator) {
			maxOp = int(v.Operator)
		}
	}
	matchOp := maxOp - (maxOp % 10)

	first := []*Operation{&Operation{Operand: e.Operand}}
	allOps := append(first, e.Operations...)
	nuOps := first
	for idx := 1; idx < len(allOps); idx++ {
		v := allOps[idx]
		if int(v.Operator) < matchOp {
			nuOps = append(nuOps, allOps[idx-1])
			continue
		}

		prevIdx := len(nuOps) - 1
		prev := nuOps[prevIdx]
		op := v.Operator.String()
		cur := &Operation{
			Operator: prev.Operator,
			Operand: &Operand{
				Value: &Value{Ident: &op},
				Calls: []*Call{&Call{Function: []*Arg{
					&Arg{Value: &Expression{Operand: prev.Operand}},
					&Arg{Value: &Expression{Operand: v.Operand}},
				}}},
			},
		}
		nuOps[prevIdx] = cur
	}

	e.Operand = nuOps[0].Operand
	e.Operations = nuOps[1:]
	return e.processOperators()
}

// processChildOperators of all block, accessor, and function child calls
func (e *Expression) processChildOperators() error {
	ops := append([]*Operation{&Operation{Operand: e.Operand}}, e.Operations...)

	// tackle all command calls recursively
	for i := range ops {
		v := ops[i].Operand
		for fi := range v.Block {
			v.Block[fi].ProcessOperators()
		}

		for fi := range v.Calls {
			v.Calls[fi].Accessor.ProcessOperators()
			f := v.Calls[fi].Function
			for ffi := range f {
				f[ffi].Value.ProcessOperators()
			}
		}
	}
	return nil
}

// ProcessOperators of this expression and all its children recursively
// and make everything be a flat expression with funtion calls only
func (e *Expression) ProcessOperators() error {
	if e == nil {
		return nil
	}
	if err := e.processChildOperators(); err != nil {
		return err
	}
	return e.processOperators()
}
