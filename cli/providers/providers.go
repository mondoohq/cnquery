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
	"go.mondoo.com/cnquery/cli/components"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/types"
)

type Command struct {
	Command *cobra.Command
	Run     func(*cobra.Command, *providers.Runtime, *plugin.ParseCLIRes)
	Action  string
}

// AttachCLIs will attempt to parse the current commandline and look for providers.
// This step is done before cobra ever takes effect
func AttachCLIs(rootCmd *cobra.Command, commands ...*Command) error {
	existing, err := providers.ListActive()
	if err != nil {
		return err
	}

	connectorName, autoUpdate := detectConnectorName(os.Args, commands)
	if connectorName != "" {
		if _, err := providers.EnsureProvider(existing, connectorName, "", autoUpdate); err != nil {
			return err
		}
	}

	// Now that we know we have all providers, it's time to load them to build
	// the remaining CLI. Probably an opportunity to optimize in the future,
	// but fine for now to do it all.

	attachProviders(existing, commands)
	return nil
}

func detectConnectorName(args []string, commands []*Command) (string, bool) {
	autoUpdate := true

	config.InitViperConfig()
	if viper.IsSet("auto_update") {
		autoUpdate = viper.GetBool("auto_update")
	}

	flags := pflag.NewFlagSet("set", pflag.ContinueOnError)
	flags.Bool("auto-update", autoUpdate, "")
	flags.BoolP("help", "h", false, "")

	builtins := genBuiltinFlags()
	for i := range builtins {
		addFlagToSet(flags, builtins[i])
	}

	for i := range commands {
		cmd := commands[i]
		cmd.Command.Flags().VisitAll(func(flag *pflag.Flag) {
			if found := flags.Lookup(flag.Name); found == nil {
				flags.AddFlag(flag)
			}
		})
	}

	err := flags.Parse(args)
	if err != nil {
		log.Warn().Err(err).Msg("CLI pre-processing encountered an issue")
	}

	autoUpdate, _ = flags.GetBool("auto-update")

	remaining := flags.Args()
	if len(remaining) <= 1 {
		return "", autoUpdate
	}

	commandFound := false
	for j := range commands {
		if commands[j].Command.Use == remaining[1] {
			commandFound = true
			break
		}
	}
	if !commandFound {
		return "", autoUpdate
	}

	// since we have a known command, we can now expect the connector to be
	// local by default if nothing else is set
	if len(remaining) == 2 {
		return "local", autoUpdate
	}

	connector := remaining[2]
	// we may want to double-check if the connector exists

	return connector, autoUpdate
}

func attachProviders(existing providers.Providers, commands []*Command) {
	for i := range commands {
		attachProvidersToCmd(existing, commands[i])
	}
}

func attachProvidersToCmd(existing providers.Providers, cmd *Command) {
	for _, provider := range existing {
		for j := range provider.Connectors {
			conn := provider.Connectors[j]
			attachConnectorCmd(provider.Provider, &conn, cmd)
			for k := range conn.Aliases {
				copyConn := conn
				copyConn.Name = conn.Aliases[k]
				attachConnectorCmd(provider.Provider, &copyConn, cmd)
			}
		}
	}

	// the default is always os.local if it exists
	if p, ok := existing[providers.DefaultOsID]; ok {
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
	cmd.Command.Short = cmd.Action + connector.Short

	setConnector(provider, connector, cmd.Run, cmd.Command)
}

func attachConnectorCmd(provider *plugin.Provider, connector *plugin.Connector, cmd *Command) {
	res := &cobra.Command{
		Use:     connector.Use,
		Short:   cmd.Action + connector.Short,
		Long:    connector.Long,
		Aliases: connector.Aliases,
		PreRun:  cmd.Command.PreRun,
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

func addFlagToSet(set *pflag.FlagSet, flag plugin.Flag) {
	switch flag.Type {
	case plugin.FlagType_Bool:
		if flag.Short != "" {
			set.BoolP(flag.Long, flag.Short, false, flag.Desc)
		} else {
			set.Bool(flag.Long, false, flag.Desc)
		}
	case plugin.FlagType_Int:
		if flag.Short != "" {
			set.IntP(flag.Long, flag.Short, 0, flag.Desc)
		} else {
			set.Int(flag.Long, 0, flag.Desc)
		}
	case plugin.FlagType_String:
		if flag.Short != "" {
			set.StringP(flag.Long, flag.Short, "", flag.Desc)
		} else {
			set.String(flag.Long, "", flag.Desc)
		}
	case plugin.FlagType_List:
		if flag.Short != "" {
			set.StringArrayP(flag.Long, flag.Short, []string{}, flag.Desc)
		} else {
			set.StringArray(flag.Long, []string{}, flag.Desc)
		}
	case plugin.FlagType_KeyValue:
		if flag.Short != "" {
			set.StringToStringP(flag.Long, flag.Short, map[string]string{}, flag.Desc)
		} else {
			set.StringToString(flag.Long, map[string]string{}, flag.Desc)
		}
	default:
		log.Warn().Msg("unknown flag type for " + flag.Long)
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

		// TODO: add flag to set timeout and then use RuntimeWithShutdownTimeout
		runtime := providers.Coordinator.NewRuntime()

		// TODO: read from config
		runtime.AutoUpdate = providers.UpdateProvidersConfig{
			Enabled:         true,
			RefreshInterval: 60 * 60,
		}

		if err := runtime.UseProvider(provider.ID); err != nil {
			providers.Coordinator.Shutdown()
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
			providers.Coordinator.Shutdown()
			log.Fatal().Err(err).Msg("failed to parse cli arguments")
		}

		if cliRes == nil {
			runtime.Close()
			providers.Coordinator.Shutdown()
			log.Fatal().Msg("failed to process CLI arguments, nothing was returned")
		}

		run(cc, runtime, cliRes)
		runtime.Close()
		providers.Coordinator.Shutdown()
	}

	for i := range allFlags {
		flag := allFlags[i]
		switch flag.Type {
		case plugin.FlagType_Bool:
			if flag.Short != "" {
				cmd.Flags().BoolP(flag.Long, flag.Short, json2T(flag.Default, false), flag.Desc)
			} else {
				cmd.Flags().Bool(flag.Long, json2T(flag.Default, false), flag.Desc)
			}
		case plugin.FlagType_Int:
			if flag.Short != "" {
				cmd.Flags().IntP(flag.Long, flag.Short, json2T(flag.Default, 0), flag.Desc)
			} else {
				cmd.Flags().Int(flag.Long, json2T(flag.Default, 0), flag.Desc)
			}
		case plugin.FlagType_String:
			if flag.Short != "" {
				cmd.Flags().StringP(flag.Long, flag.Short, flag.Default, flag.Desc)
			} else {
				cmd.Flags().String(flag.Long, flag.Default, flag.Desc)
			}
		case plugin.FlagType_List:
			if flag.Short != "" {
				cmd.Flags().StringSliceP(flag.Long, flag.Short, json2T(flag.Default, []string{}), flag.Desc)
			} else {
				cmd.Flags().StringSlice(flag.Long, json2T(flag.Default, []string{}), flag.Desc)
			}
		case plugin.FlagType_KeyValue:
			if flag.Short != "" {
				cmd.Flags().StringToStringP(flag.Long, flag.Short, json2T(flag.Default, map[string]string{}), flag.Desc)
			} else {
				cmd.Flags().StringToString(flag.Long, json2T(flag.Default, map[string]string{}), flag.Desc)
			}
		}

		if flag.Option&plugin.FlagOption_Hidden != 0 {
			cmd.Flags().MarkHidden(flag.Long)
		}
		if flag.Option&plugin.FlagOption_Deprecated != 0 {
			cmd.Flags().MarkDeprecated(flag.Long, "has been deprecated")
		}
	}
}

func json2T[T any](s string, empty T) T {
	var res T
	if err := json.Unmarshal([]byte(s), &res); err == nil {
		return res
	}
	return empty
}
