package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the cnquery version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(cnquery.Info())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
