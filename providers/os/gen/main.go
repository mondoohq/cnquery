package main

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin/gen"
	"go.mondoo.com/cnquery/providers/os/config"
)

func main() {
	gen.CLI(&config.Config)
}
