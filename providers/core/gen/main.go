package main

import (
	"go.mondoo.com/cnquery/providers/core/config"
	"go.mondoo.com/cnquery/providers/plugin/gen"
)

func main() {
	gen.CLI(&config.Config)
}
