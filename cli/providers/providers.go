// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v10/cli/components"
	"go.mondoo.com/cnquery/v10/cli/config"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/types"
)

type Command struct {
	Command             *cobra.Command
	Run                 func(*cobra.Command, *providers.Runtime, *plugin.ParseCLIRes)
	Action              string
	SupportedConnectors []string
}

// AttachCLIs will attempt to parse the current commandline and look for providers.
// This step is done before cobra ever takes effect
func AttachCLIs(rootCmd *cobra.Command, commands ...*Command) error {
	existing, err := providers.ListActive()
	if err != nil {
		return err
	}

	connectorName, autoUpdate := detectConnectorName(os.Args, rootCmd, commands, existing)
	if connectorName != "" {
		if _, err := providers.EnsureProvider(providers.ProviderLookup{ConnName: connectorName}, autoUpdate, existing); err != nil {
			return err
		}
	}

	// Now that we know we have all providers, it's time to load them to build
	// the remaining CLI. Probably an opportunity to optimize in the future,
	// but fine for now to do it all.

	attachProviders(existing, commands)
	return nil
}

func detectConnectorName(args []string, rootCmd *cobra.Command, commands []*Command, providers providers.Providers) (string, bool) {
	autoUpdate := true

	config.InitViperConfig()
	if viper.IsSet("auto_update") {
		autoUpdate = viper.GetBool("auto_update")
	}

	flags := pflag.NewFlagSet("set", pflag.ContinueOnError)
	flags.ParseErrorsWhitelist.UnknownFlags = true
	flags.Bool("auto-update", autoUpdate, "")
	flags.BoolP("help", "h", false, "")

	builtins := genBuiltinFlags()
	for i := range builtins {
		attachFlag(flags, builtins[i])
	}

	// To avoid warnings about flags, we need to mock all flags on the root command.
	// The command after root (eg: run, scan, shell, ...) are normal actions that
	// we want to detect. We only need to add flags from the root command and the
	// attaching subcommands (since those are the ones which end up giving us
	// the connector)
	attachPFlags(flags, rootCmd.Flags())
	attachPFlags(flags, rootCmd.PersistentFlags())

	for i := range commands {
		cmd := commands[i]
		attachPFlags(flags, cmd.Command.Flags())
	}

	for i := range providers {
		provider := providers[i]
		for j := range provider.Connectors {
			conn := provider.Connectors[j]
			for k := range conn.Flags {
				flag := conn.Flags[k]
				if found := flags.Lookup(flag.Long); found == nil {
					attachFlag(flags, flag)
				}
			}
		}
	}

	err := flags.Parse(args)
	if err != nil {
		log.Warn().Err(err).Msg("CLI pre-processing encountered an issue")
	}

	autoUpdate, _ = flags.GetBool("auto-update")

	parsedArgs := flags.Args()
	if len(parsedArgs) <= 1 {
		return "", autoUpdate
	}

	commandFound := false
	for j := range commands {
		if commands[j].Command.Use == parsedArgs[1] {
			commandFound = true
			break
		}
	}
	if !commandFound {
		return "", autoUpdate
	}

	// since we have a known command, we can now expect the connector to be
	// local by default if nothing else is set
	if len(parsedArgs) == 2 {
		return "local", autoUpdate
	}

	connector := parsedArgs[2]
	// we may want to double-check if the connector exists

	return connector, autoUpdate
}

func attachProviders(existing providers.Providers, commands []*Command) {
	for i := range commands {
		attachProvidersToCmd(existing, commands[i])
	}
}

func attachProvidersToCmd(existing providers.Providers, cmd *Command) {
	for i := range existing {
		provider := existing[i]
		for j := range provider.Connectors {
			conn := provider.Connectors[j]

			attach := true
			if len(cmd.SupportedConnectors) > 0 {
				attach = false // only attach if the connector is in the list
				for k := range cmd.SupportedConnectors {
					if cmd.SupportedConnectors[k] == conn.Name {
						attach = true
						break
					}
				}
			}
			if attach {
				attachConnectorCmd(provider.Provider, &conn, cmd)
			}
		}
	}

	// the default is always os.local if it exists
	if p, ok := existing.GetFirstID(providers.DefaultOsIDs...); ok {
		for i := range p.Connectors {
			c := p.Connectors[i]
			if c.Name == "local" {
				setDefaultConnector(p.Provider, &c, cmd)
				break
			}
		}
	}
}

func setDefaultConnector(provider *plugin.Provider, connector *plugin.Connector, cmd *Command) {
	cmd.Command.Run = func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			log.Error().Msg("provider " + args[0] + " does not exist")
			cmd.Help()
			os.Exit(1)
		}

		log.Info().Msg("no provider specified, defaulting to local. Use --help to see all providers.")
	}

	setConnector(provider, connector, cmd.Run, cmd.Command)
}

func attachConnectorCmd(provider *plugin.Provider, connector *plugin.Connector, cmd *Command) {
	res := &cobra.Command{
		Use:     connector.Use,
		Short:   cmd.Action + connector.Short,
		Long:    connector.Long,
		Aliases: connector.Aliases,
		PreRun:  cmd.Command.PreRun,
		PreRunE: cmd.Command.PreRunE,
	}

	if connector.MinArgs == connector.MaxArgs {
		if connector.MinArgs == 0 {
			res.Args = cobra.NoArgs
		} else {
			res.Args = cobra.ExactArgs(int(connector.MinArgs))
		}
	} else {
		if connector.MaxArgs > 0 && connector.MinArgs == 0 {
			res.Args = cobra.MaximumNArgs(int(connector.MaxArgs))
		} else if connector.MaxArgs == 0 && connector.MinArgs > 0 {
			res.Args = cobra.MinimumNArgs(int(connector.MinArgs))
		} else {
			res.Args = cobra.RangeArgs(int(connector.MinArgs), int(connector.MaxArgs))
		}
	}
	cmd.Command.Flags().VisitAll(func(flag *pflag.Flag) {
		res.Flags().AddFlag(flag)
	})

	cmd.Command.AddCommand(res)
	setConnector(provider, connector, cmd.Run, res)
}

func genBuiltinFlags(discoveries ...string) []plugin.Flag {
	supportedDiscoveries := append([]string{"all", "auto"}, discoveries...)

	return []plugin.Flag{
		// flags for providers:
		{
			Long: "discover",
			Type: plugin.FlagType_List,
			Desc: "Enable the discovery of nested assets. Supports: " + strings.Join(supportedDiscoveries, ","),
		},
		{
			Long:   "pretty",
			Type:   plugin.FlagType_Bool,
			Desc:   "Pretty-print JSON",
			Option: plugin.FlagOption_Hidden,
		},
		// runtime-only flags:
		{
			Long: "record",
			Type: plugin.FlagType_String,
			Desc: "Record all resource calls and use resources in the recording",
		},
		{
			Long: "use-recording",
			Type: plugin.FlagType_String,
			Desc: "Use a recording to inject resource data (read-only)",
		},
	}
}

// the following flags are not processed by providers
var skipFlags = map[string]struct{}{
	"ask-pass":      {},
	"record":        {},
	"use-recording": {},
}

func attachPFlags(base *pflag.FlagSet, nu *pflag.FlagSet) {
	nu.VisitAll(func(flag *pflag.Flag) {
		if found := base.Lookup(flag.Name); found != nil {
			return
		}
		if flag.Shorthand != "" {
			if found := base.ShorthandLookup(flag.Shorthand); found != nil {
				return
			}
		}
		base.AddFlag(flag)
	})
}

func attachFlag(flagset *pflag.FlagSet, flag plugin.Flag) {
	switch flag.Type {
	case plugin.FlagType_Bool:
		if flag.Short != "" {
			flagset.BoolP(flag.Long, flag.Short, json2T(flag.Default, false), flag.Desc)
		} else {
			flagset.Bool(flag.Long, json2T(flag.Default, false), flag.Desc)
		}
	case plugin.FlagType_Int:
		if flag.Short != "" {
			flagset.IntP(flag.Long, flag.Short, json2T(flag.Default, 0), flag.Desc)
		} else {
			flagset.Int(flag.Long, json2T(flag.Default, 0), flag.Desc)
		}
	case plugin.FlagType_String:
		if flag.Short != "" {
			flagset.StringP(flag.Long, flag.Short, flag.Default, flag.Desc)
		} else {
			flagset.String(flag.Long, flag.Default, flag.Desc)
		}
	case plugin.FlagType_List:
		if flag.Short != "" {
			flagset.StringSliceP(flag.Long, flag.Short, json2T(flag.Default, []string{}), flag.Desc)
		} else {
			flagset.StringSlice(flag.Long, json2T(flag.Default, []string{}), flag.Desc)
		}
	case plugin.FlagType_KeyValue:
		if flag.Short != "" {
			flagset.StringToStringP(flag.Long, flag.Short, json2T(flag.Default, map[string]string{}), flag.Desc)
		} else {
			flagset.StringToString(flag.Long, json2T(flag.Default, map[string]string{}), flag.Desc)
		}
	}

	if flag.Option&plugin.FlagOption_Hidden != 0 {
		flagset.MarkHidden(flag.Long)
	}
	if flag.Option&plugin.FlagOption_Deprecated != 0 {
		flagset.MarkDeprecated(flag.Long, "has been deprecated")
	}
}

func attachFlags(flagset *pflag.FlagSet, flags []plugin.Flag) {
	for i := range flags {
		attachFlag(flagset, flags[i])
	}
}

func getFlagValue(flag plugin.Flag, cmd *cobra.Command) *llx.Primitive {
	switch flag.Type {
	case plugin.FlagType_Bool:
		v, err := cmd.Flags().GetBool(flag.Long)
		if err == nil {
			return llx.BoolPrimitive(v)
		}
		log.Warn().Err(err).Msg("failed to get flag " + flag.Long)
	case plugin.FlagType_Int:
		if v, err := cmd.Flags().GetInt(flag.Long); err == nil {
			return llx.IntPrimitive(int64(v))
		}
	case plugin.FlagType_String:
		if v, err := cmd.Flags().GetString(flag.Long); err == nil {
			return llx.StringPrimitive(v)
		}
	case plugin.FlagType_List:
		if v, err := cmd.Flags().GetStringSlice(flag.Long); err == nil {
			return llx.ArrayPrimitiveT(v, llx.StringPrimitive, types.String)
		}
	case plugin.FlagType_KeyValue:
		if v, err := cmd.Flags().GetStringToString(flag.Long); err == nil {
			return llx.MapPrimitiveT(v, llx.StringPrimitive, types.String)
		}
	default:
		log.Warn().Msg("unknown flag type for " + flag.Long)
		return nil
	}
	return nil
}

func setConnector(provider *plugin.Provider, connector *plugin.Connector, run func(*cobra.Command, *providers.Runtime, *plugin.ParseCLIRes), cmd *cobra.Command) {
	oldRun := cmd.Run
	oldPreRun := cmd.PreRun

	builtinFlags := genBuiltinFlags(connector.Discovery...)
	allFlags := append(connector.Flags, builtinFlags...)

	cmd.PreRun = func(cc *cobra.Command, args []string) {
		if oldPreRun != nil {
			oldPreRun(cc, args)
		}

		// Config options need to be connected to flags before the Run begins.
		// Flags are provided by the connector.
		for i := range allFlags {
			flag := allFlags[i]
			if flag.ConfigEntry == "-" {
				continue
			}

			flagName := flag.ConfigEntry
			if flagName == "" {
				flagName = flag.Long
			}

			viper.BindPFlag(flagName, cmd.Flags().Lookup(flag.Long))
		}
	}

	cmd.Run = func(cc *cobra.Command, args []string) {
		if oldRun != nil {
			oldRun(cc, args)
		}

		log.Debug().Msg("using provider " + provider.Name + " with connector " + connector.Name)

		// TODO: replace this hard-coded block. This should be dynamic for all
		// fields that are specified to be passwords with the --ask-field
		// associated with it to make it simple.
		// check if the user used --password without a value
		askPass, err := cc.Flags().GetBool("ask-pass")
		if err == nil && askPass {
			pass, err := components.AskPassword("Enter password: ")
			if err != nil {
				log.Fatal().Err(err).Msg("failed to get password")
			}
			cc.Flags().Set("password", pass)
		}
		// ^^

		useRecording, err := cc.Flags().GetString("use-recording")
		if err != nil {
			log.Warn().Msg("failed to get flag --recording")
		}
		record, err := cc.Flags().GetString("record")
		if err != nil {
			log.Warn().Msg("failed to get flag --record")
		}
		pretty, err := cc.Flags().GetBool("pretty")
		if err != nil {
			log.Warn().Msg("failed to get flag --pretty")
		}

		flagVals := map[string]*llx.Primitive{}
		for i := range allFlags {
			flag := allFlags[i]

			// we skip these because they are coded above
			if _, skip := skipFlags[flag.Long]; skip {
				continue
			}

			if v := getFlagValue(flag, cmd); v != nil {
				flagVals[flag.Long] = v
			}
		}

		coordinator := providers.NewCoordinator()
		defer coordinator.Shutdown()

		// TODO: add flag to set timeout and then use RuntimeWithShutdownTimeout
		runtime := coordinator.NewRuntime()
		defer runtime.Close()
		if err = providers.SetDefaultRuntime(runtime); err != nil {
			log.Error().Msg(err.Error())
		}

		autoUpdate := true
		if viper.IsSet("auto_update") {
			autoUpdate = viper.GetBool("auto_update")
		}

		runtime.AutoUpdate = providers.UpdateProvidersConfig{
			Enabled:         autoUpdate,
			RefreshInterval: 60 * 60,
		}

		if err := runtime.UseProvider(provider.ID); err != nil {
			log.Fatal().Err(err).Msg("failed to start provider " + provider.Name)
		}

		if record != "" && useRecording != "" {
			log.Fatal().Msg("please only use --record or --use-recording, but not both at the same time")
		}
		recordingPath := record
		if recordingPath == "" {
			recordingPath = useRecording
		}
		doRecord := record != ""

		recording, err := providers.NewRecording(recordingPath, providers.RecordingOptions{
			DoRecord:        doRecord,
			PrettyPrintJSON: pretty,
		})
		if err != nil {
			log.Fatal().Msg(err.Error())
		}
		runtime.SetRecording(recording)

		cliRes, err := runtime.Provider.Instance.Plugin.ParseCLI(&plugin.ParseCLIReq{
			Connector: connector.Name,
			Args:      args,
			Flags:     flagVals,
		})
		if err != nil {
			runtime.Close()
			log.Fatal().Err(err).Msg("failed to parse cli arguments")
		}

		if cliRes == nil {
			runtime.Close()
			log.Fatal().Msg("failed to process CLI arguments, nothing was returned")
			return // adding this here as a compiler hint to stop warning about nil-dereferences
		}

		if cliRes.Asset == nil {
			log.Warn().Err(err).Msg("failed to discover assets after processing CLI arguments")
		} else {
			assetRuntime, err := coordinator.RuntimeFor(cliRes.Asset, runtime)
			if err != nil {
				log.Warn().Err(err).Msg("failed to get runtime for an asset that was detected after parsing the CLI")
			} else {
				runtime = assetRuntime
			}
		}

		run(cc, runtime, cliRes)
	}

	attachFlags(cmd.Flags(), allFlags)
}

func json2T[T any](s string, empty T) T {
	var res T
	if err := json.Unmarshal([]byte(s), &res); err == nil {
		return res
	}
	return empty
}
