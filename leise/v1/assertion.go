package v1

import (
	"errors"
	"strconv"
	"strings"

	"go.mondoo.io/mondoo/leise/parser"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/types"
)

func extractComments(c *parser.Expression) string {
	// TODO: we need to clarify how many of the comments we really want to extract.
	// For now we only grab the operand and ignore the rest
	if c == nil || c.Operand == nil {
		return ""
	}
	return c.Operand.Comments
}

func extractMsgTag(comment string) string {
	lines := strings.Split(comment, "\n")
	var msgLines strings.Builder

	var i int
	for i < len(lines) {
		if strings.HasPrefix(lines[i], "@msg ") {
			break
		}
		i++
	}
	if i == len(lines) {
		return ""
	}

	msgLines.WriteString(lines[i][5:])
	msgLines.WriteByte('\n')
	i++

	for i < len(lines) {
		line := lines[i]
		if line != "" && line[0] == '@' {
			break
		}
		msgLines.WriteString(line)
		msgLines.WriteByte('\n')
		i++
	}

	return msgLines.String()
}

func extractMql(s string) (string, error) {
	var openBrackets []byte
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"', '\'':
			// TODO: for all of these string things we need to support proper string interpolation...
			d := s[i]
			for ; i < len(s) && s[i] != d; i++ {
			}
		case '{', '(', '[':
			openBrackets = append(openBrackets, s[i])
		case '}':
			if len(openBrackets) == 0 {
				return s[0:i], nil
			}
			last := openBrackets[len(openBrackets)-1]
			if last != '{' {
				return "", errors.New("unexpected closing bracket '" + string(s[i]) + "'")
			}
			openBrackets = openBrackets[0 : len(openBrackets)-1]
		case ')', ']':
			if len(openBrackets) == 0 {
				return "", errors.New("unexpected closing bracket '" + string(s[i]) + "'")
			}
			last := openBrackets[len(openBrackets)-1]
			if (s[i] == ')' && last != '(') || (s[i] == ']' && last != '[') {
				return "", errors.New("unexpected closing bracket '" + string(s[i]) + "'")
			}
			openBrackets = openBrackets[0 : len(openBrackets)-1]
		}
	}

	return s, nil
}

func compileAssertionMsg(msg string, c *compiler) (*llx.AssertionMessage, error) {
	template := strings.Builder{}
	var codes []string
	var i int
	var max = len(msg)
	textStart := i
	for ; i < max; i++ {
		if msg[i] != '$' {
			continue
		}
		if i+1 == max || msg[i+1] != '{' {
			continue
		}

		template.WriteString(msg[textStart:i])
		template.WriteByte('$')
		template.WriteString(strconv.Itoa(len(codes)))

		// extract the code
		code, err := extractMql(msg[i+2:])
		if err != nil {
			return nil, err
		}

		i += 2 + len(code)
		if i >= max {
			return nil, errors.New("cannot extract code in @msg (message ended before '}')")
		}
		if msg[i] != '}' {
			return nil, errors.New("cannot extract code in @msg (expected '}' but got '" + string(msg[i]) + "')")
		}
		textStart = i + 1 // one past the closing '}'

		codes = append(codes, code)
	}

	template.WriteString(msg[textStart:])

	res := llx.AssertionMessage{
		Template: strings.Trim(template.String(), "\n\t "),
	}

	for i := range codes {
		code := codes[i]

		// Small helper for assertion messages:
		// At the moment, the parser can't deliniate if a given `{}` call
		// is meant to be a map creation or a block call.
		//
		// When it is at the beginning of an operand it is always treated
		// as a map creation, e.g.:
		//     {a: 123, ...}             vs
		//     something { block... }
		//
		// However, in the assertion message case we know it's generally
		// not about map-creation. So we are using a workaround to more
		// easily extract values via blocks.
		//
		// This approach is extremely limited. It works with the most
		// straightforward use-case and prohibits map any type of map
		// creation in assertion messages.
		//
		// TODO: Find a more appropriate solution for this problem.
		// Identify use-cases we don't cover well with this approach
		// before changing it.

		code = strings.Trim(code, " \t\n")
		if code[0] == '{' {
			code = "_" + code
		}

		ast, err := parser.Parse(code)
		if err != nil {
			return nil, errors.New("cannot parse code block in comment: " + code)
		}

		if len(ast.Expressions) == 0 {
			return nil, errors.New("can't have empty calls to `${}` in comments")
		}
		if len(ast.Expressions) > 1 {
			return nil, errors.New("can't have more than one value in `${}`")
		}
		expression := ast.Expressions[0]

		ref, err := c.compileAndAddExpression(expression)
		if err != nil {
			return nil, errors.New("failed to compile comment: " + err.Error())
		}

		res.DeprecatedV5Datapoint = append(res.DeprecatedV5Datapoint, ref)
		resCode := c.Result.DeprecatedV5Code
		resCode.Datapoints = append(resCode.Datapoints, ref)
	}

	return &res, nil
}

func compileListAssertionMsg(c *compiler, typ types.Type, allRef int32, failedRef int32, assertionRef int32) error {
	// assertions
	msg := extractMsgTag(c.comment)
	if msg == "" {
		return nil
	}

	code := c.Result.DeprecatedV5Code

	blockCompiler := c.newBlockCompiler(&llx.CodeV1{
		Id:         "binding",
		Parameters: 2,
		Checksums: map[int32]string{
			// we must provide the first chunk, which is a reference to the caller
			// and which will always be number 1
			1: code.Checksums[code.ChunkIndex()-1],
			2: code.Checksums[allRef],
		},
		Code: []*llx.Chunk{
			{
				Call:      llx.Chunk_PRIMITIVE,
				Primitive: &llx.Primitive{Type: string(typ)},
			},
			{
				Call:      llx.Chunk_PRIMITIVE,
				Primitive: &llx.Primitive{Type: string(typ)},
			},
		},
	}, &binding{Type: types.Type(typ), Ref: 1})

	blockCompiler.vars["$expected"] = variable{ref: 2, typ: typ}

	assertionMsg, err := compileAssertionMsg(msg, &blockCompiler)
	if err != nil {
		return err
	}
	if assertionMsg != nil {
		if code.Assertions == nil {
			code.Assertions = map[int32]*llx.AssertionMessage{}
		}
		code.Assertions[assertionRef+2] = assertionMsg

		block := blockCompiler.Result.DeprecatedV5Code
		block.UpdateID()
		code.Functions = append(code.Functions, block)
		//return code.FunctionsIndex(), blockCompiler.standalone, nil

		fref := code.FunctionsIndex()
		code.AddChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "${}",
			Function: &llx.Function{
				Type:                string(types.Block),
				DeprecatedV5Binding: failedRef,
				Args: []*llx.Primitive{
					llx.FunctionPrimitiveV1(fref), llx.RefPrimitiveV1(allRef),
				},
			},
		})

		// since it operators on top of a block, we have to add its
		// checksum as the first entry in the list. Once the block is received,
		// all of its child entries are processed for the final result
		blockRef := code.ChunkIndex()
		checksum := code.Checksums[blockRef]
		assertionMsg.Checksums = make([]string, len(assertionMsg.DeprecatedV5Datapoint)+1)
		assertionMsg.Checksums[0] = checksum
		code.Datapoints = append(code.Datapoints, blockRef)

		blocksums := blockCompiler.Result.DeprecatedV5Code.Checksums
		for i := range assertionMsg.DeprecatedV5Datapoint {
			sum, ok := blocksums[assertionMsg.DeprecatedV5Datapoint[i]]
			if !ok {
				return errors.New("cannot find checksum for datapoint in @msg tag")
			}

			assertionMsg.Checksums[i+1] = sum
		}
		assertionMsg.DeprecatedV5Datapoint = nil
		assertionMsg.DecodeBlock = true
	}

	return nil
}

// UpdateAssertions in a bundle and remove all intermediate assertion objects
func UpdateAssertions(bundle *llx.CodeBundle) error {
	bundle.DeprecatedV5Assertions = map[string]*llx.AssertionMessage{}
	return updateCodeAssertions(bundle, bundle.DeprecatedV5Code)
}

func updateCodeAssertions(bundle *llx.CodeBundle, code *llx.CodeV1) error {
	for ref, assert := range code.Assertions {
		sum, ok := code.Checksums[ref]
		if !ok {
			return errors.New("cannot find reference for assertion")
		}

		if !assert.DecodeBlock {
			assert.Checksums = make([]string, len(assert.DeprecatedV5Datapoint))
			for i := range assert.DeprecatedV5Datapoint {
				datapoint := assert.DeprecatedV5Datapoint[i]
				assert.Checksums[i], ok = code.Checksums[datapoint]
				if !ok {
					return errors.New("cannot find reference to datapoint in assertion")
				}
			}
			assert.DeprecatedV5Datapoint = nil
		}

		bundle.DeprecatedV5Assertions[sum] = assert
	}
	code.Assertions = nil

	for i := range code.Functions {
		child := code.Functions[i]
		if err := updateCodeAssertions(bundle, child); err != nil {
			return err
		}
	}

	return nil
}
