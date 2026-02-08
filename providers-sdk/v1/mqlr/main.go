// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package main

import (
	"go.mondoo.com/mql/v13/logger"
	"go.mondoo.com/mql/v13/providers-sdk/v1/mqlr/cmd"
)

func init() {
	logger.Set("debug")
}

func main() {
	cmd.Execute()
}
