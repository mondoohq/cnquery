package main

import (
	"os"

	"go.mondoo.com/cnquery/providers/core/provider"
	"go.mondoo.com/cnquery/providers/plugin"
)

func main() {
	plugin.Start(os.Args, provider.Init())
}
