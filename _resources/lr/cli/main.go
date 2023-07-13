// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package main

import (
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/resources/lr/cli/cmd"
)

func init() {
	logger.Set("debug")
}

func main() {
	cmd.Execute()
}
