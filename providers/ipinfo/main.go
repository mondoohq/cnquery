package main

import (
	"os"

	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/ipinfo/provider"
)

func main() {
	plugin.Start(os.Args, provider.Init())
}
