package common

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type CommandsDocs struct {
	Entries map[string]CommandDocsEntry
}

type CommandDocsEntry struct {
	Short string
	Long  string
}

func (c CommandsDocs) GetShort(id string) string {
	e, ok := c.Entries[id]
	if ok {
		return e.Short
	}
	return ""
}

func (c CommandsDocs) GetLong(id string) string {
	e, ok := c.Entries[id]
	if ok {
		return e.Long
	}
	return ""
}

type (
	CommonFlagsFn  func(cmd *cobra.Command)
	CommonPreRunFn func(cmd *cobra.Command, args []string)
	RunFn          func(cmd *cobra.Command, args []string)
)

func AzureProviderCmd(commonCmdFlags CommonFlagsFn, preRun CommonPreRunFn, runFn RunFn, docs CommandsDocs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "azure",
		Short: docs.GetShort("azure"),
		Long:  docs.GetLong("azure"),
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			preRun(cmd, args)
			viper.BindPFlag("subscription", cmd.Flags().Lookup("subscription"))
		},
		Run: runFn,
	}
	commonCmdFlags(cmd)
	cmd.Flags().String("tenant-id", "", "Directory (tenant) ID of the service principal")
	cmd.Flags().String("client-id", "", "Application (client) ID of the service principal")
	cmd.Flags().String("client-secret", "", "Secret for application")
	cmd.Flags().String("certificate-path", "", "Path (in PKCS #12/PFX or PEM format) to the authentication certificate")
	cmd.Flags().String("certificate-secret", "", "Passphrase for the authentication certificate file")
	cmd.Flags().String("subscription", "", "ID of the Azure subscription to scan")
	cmd.Flags().String("subscriptions", "", "Comma-separated list of Azure subscriptions to include")
	cmd.Flags().String("subscriptions-exclude", "", "Comma-separated list of Azure subscriptions to exclude")

	return cmd
}
