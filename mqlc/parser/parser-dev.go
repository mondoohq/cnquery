//go:build !production
// +build !production

package parser

import (
	"encoding/json"
	"fmt"
)

func (p *parser) inspect(pos string) {
	fmt.Printf("%s: [%#v] - %#v\n", pos, p.nextTokens, p.token)
}

// Inspect the AST of the parsed tree
func Inspect(ast *AST) {
	res, err := json.MarshalIndent(ast, "", "  ")
	if err != nil {
		fmt.Println("\033[31;1mFailed to marshal AST:\033[0m " + err.Error())
	}
	fmt.Println(string(res))
}
