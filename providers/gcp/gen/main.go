package main

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin/gen"
	"go.mondoo.com/cnquery/providers/gcp/config"
)

func main() {
	gen.CLI(&config.Config)
}
