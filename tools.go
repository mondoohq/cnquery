//go:build tools
// +build tools

package cnquery

import (
	_ "github.com/golang/mock/mockgen"
	_ "golang.org/x/tools/cmd/stringer"
)
