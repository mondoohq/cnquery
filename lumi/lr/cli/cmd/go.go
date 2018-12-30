package cmd

import (
	"fmt"
	"io/ioutil"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.io/mondoo/lumi/lr"
)

var printStdout = false

var goCmd = &cobra.Command{
	Use:   "go",
	Short: "convert LR file to go",
	Long:  `parse an LR file and convert it to go, saving it in the same location with the suffix .go`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		raw, err := ioutil.ReadFile(args[0])
		if err != nil {
			log.Error().Err(err)
			return
		}

		res, err := lr.Parse(string(raw))
		if err != nil {
			log.Error().Err(err).Msg("failed to parse LR code")
			return
		}

		godata, err := lr.Go(res, lr.NewCollector(args[0]))
		if err != nil {
			log.Error().Err(err).Msg("failed to compile go code")
		}

		if printStdout {
			fmt.Println(godata)
		} else {
			err = ioutil.WriteFile(args[0]+".go", []byte(godata), 0644)
			if err != nil {
				log.Error().Err(err).Msg("failed to write to go file")
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(goCmd)
	goCmd.Flags().BoolVarP(&printStdout, "stdout", "", false, "print generated data to stdout instead of writing to file")
}
