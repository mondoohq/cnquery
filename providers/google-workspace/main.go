package main

import (
	"os"

	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/google-workspace/provider"
)

func main() {
	plugin.Start(os.Args, provider.Init())
}
