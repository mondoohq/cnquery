package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.io/mondoo/lumi/lr"
)

var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "parse an LR file",
	Long:  `parse an LR file and print the AST`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		raw, err := ioutil.ReadFile(args[0])
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}

		res, err := lr.Parse(string(raw))
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}

		s, _ := json.Marshal(res)
		fmt.Println(string(s))
	},
}

func init() {
	rootCmd.AddCommand(parseCmd)
}
