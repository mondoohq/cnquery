package main

import (
	"os"

	"go.mondoo.com/cnquery/providers/os/provider"
	"go.mondoo.com/cnquery/providers/plugin"
)

func main() {
	plugin.Start(os.Args, &provider.Service{})
}
