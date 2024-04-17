// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
)

func init() {
	vaultListCmd.Flags().Bool("show-options", false, "displays configured options")
	VaultCmd.AddCommand(vaultListCmd)

	vaultConfigureCmd.Flags().String("type", "", "possible values: "+strings.Join(vault.TypeIds(), " | "))
	vaultConfigureCmd.Flags().StringToString("option", nil, "addition vault connection options, multiple options via --option key=value")
	vaultConfigureCmd.Flags().String("inventory-file", "", "Set the path to the inventory file.")
	VaultCmd.AddCommand(vaultConfigureCmd)

	VaultCmd.AddCommand(vaultRemoveCmd)
	VaultCmd.AddCommand(vaultResetCmd)

	vaultAddSecretCmd.Flags().String("inventory-file", "", "Set the path to the inventory file.")
	vaultAddSecretCmd.MarkFlagRequired("inventory-file")
	VaultCmd.AddCommand(vaultAddSecretCmd)

	rootCmd.AddCommand(VaultCmd)
}

// VaultCmd represents the vault command
var VaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Manage vault environments",
	Long:  ``,
}

var vaultListCmd = &cobra.Command{
	Use:    "list",
	Short:  "List vault environments",
	Long:   ``,
	Hidden: true,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("show-options", cmd.Flags().Lookup("show-options"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Fatal().Msg("sub-command is not supported anymore, see https://mondoo.com/docs/platform/infra/opsys/automation/vault/ for how to use vault environments")
	},
}

var vaultConfigureCmd = &cobra.Command{
	Use:     "configure VAULTNAME",
	Aliases: []string{"set"},
	Short:   "Configure a vault environment",
	Long: `

cnquery vault configure mondoo-client-vault --type linux-kernel-keyring

`,
	Args: cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("type", cmd.Flags().Lookup("type"))
		viper.BindPFlag("option", cmd.Flags().Lookup("option"))
		viper.BindPFlag("inventory-file", cmd.Flags().Lookup("inventory-file"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		// overwrite existing / set vault config
		// field name = vault name
		vt, err := vault.NewVaultType(viper.GetString("type"))
		if err != nil {
			log.Fatal().Err(err).Msg("invalid vault configuration type")
		}

		vaultName := args[0]
		cfg := &vault.VaultConfiguration{
			Name:    vaultName,
			Type:    vt,
			Options: viper.GetStringMapString("option"),
		}

		inventoryFile := viper.GetString("inventory-file")
		if inventoryFile != "" {
			inventory, err := inventory.InventoryFromFile(inventoryFile)
			if err != nil {
				log.Fatal().Err(err).Msg("could not load inventory")
			}
			inventory.Spec.Vault = cfg

			// store inventory file
			data, err := inventory.ToYAML()
			if err != nil {
				log.Fatal().Err(err).Msg("could not marshal inventory")
			}
			err = os.WriteFile(viper.GetString("inventory-file"), data, 0o644)
			if err != nil {
				log.Fatal().Err(err).Msg("could not write inventory file")
			}
			log.Info().Msg("stored vault configuration successfully")
		} else {
			log.Info().Msg("add the following vault configuration to your inventory file")

			inventory := &inventory.Inventory{
				Spec: &inventory.InventorySpec{
					Vault: cfg,
				},
			}
			data, err := inventory.ToYAML()
			if err != nil {
				log.Fatal().Err(err).Msg("could not marshal vault configuration")
			}
			fmt.Println(string(data))
		}
	},
}

var vaultRemoveCmd = &cobra.Command{
	Use:    "remove VAULTNAME",
	Short:  "Remove a configured vault environment",
	Long:   ``,
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		log.Fatal().Msg("sub-command is not supported anymore, see https://mondoo.com/docs/platform/infra/opsys/automation/vault/ for how to use vault environments")
	},
}

var vaultResetCmd = &cobra.Command{
	Use:    "reset",
	Short:  "Reset the vault configuration to defaults",
	Long:   ``,
	Args:   cobra.ExactArgs(0),
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		log.Fatal().Msg("sub-command is not supported anymore, see https://mondoo.com/docs/platform/infra/opsys/automation/vault/ for how to use vault environments")
	},
}

var vaultAddSecretCmd = &cobra.Command{
	Use:   "add-secret SECRETID SECRETVALUE",
	Short: "Store a secret in a vault",
	Args:  cobra.ExactArgs(2),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("inventory-file", cmd.Flags().Lookup("inventory-file"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Msg("load vault configuration from inventory")
		inventory, err := inventory.InventoryFromFile(viper.GetString("inventory-file"))
		if err != nil {
			log.Fatal().Err(err).Msg("could not load inventory")
		}

		v, err := inventory.GetVault()
		if err != nil {
			log.Fatal().Err(err).Msg("could not load vault configuration from inventory")
		}

		_, err = v.Set(context.Background(), &vault.Secret{
			Key:  args[0],
			Data: []byte(args[1]),
		})
		if err != nil {
			log.Fatal().Err(err).Msg("could not store secret")
		}
		log.Info().Msg("stored secret successfully")
	},
}
