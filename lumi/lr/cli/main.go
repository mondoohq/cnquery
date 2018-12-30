// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package main

import (
	"go.mondoo.io/mondoo/logger"
	"go.mondoo.io/mondoo/lumi/lr/cli/cmd"
)

func init() {
	logger.Set(true, true)
}

func main() {
	cmd.Execute()
}
