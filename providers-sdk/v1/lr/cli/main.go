// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package main

import (
	"go.mondoo.com/cnquery/v10/logger"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/lr/cli/cmd"
)

func init() {
	logger.Set("debug")
}

func main() {
	cmd.Execute()
}
