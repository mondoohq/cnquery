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

	connectorName := detectConnector(os.Args, rootCmd, commands)
	if connectorName == "" {
		return nil
	}

	if _, err := providers.EnsureProvider(existing, connectorName); err != nil {
		return err
	}

	// Now that we know we have all providers, it's time to load them to build
	// the remaining CLI. Probably an opportunity to optimize in the future,
	// but fine for now to do it all.

	attacheProviders(existing, commands)
	return nil
}

func flagHasArgs(flag *pflag.Flag) bool {
	if flag == nil {
		return false
	}
	return flag.NoOptDefVal == ""
}

func detectConnector(args []string, rootCmd *cobra.Command, commands []*Command) string {
	// We cannot fully parse the cli, yet. So we have to deal with what we have.
	// We can safely ignore all options up to a point. We are looking for one of
	// the supported commands, which can be followed by a provider.

	// because the default first arg is the calling program, we are ignoring it
	// and instead starting at arg position 2 (idx=1)
	cmd, arg2, err := rootCmd.Find(args[1:])
	if cmd == nil || err != nil {
		return ""
	}

	found := false
	for j := range commands {
		if cmd == commands[j].Command {
			found = true
			continue
		}
	}
	if !found {
		return ""
	}

	argIsValue := false
	for _, arg := range arg2 {
		switch {
		case strings.Contains(arg, "="):
			continue // assigned flags don't get additional arg
		case strings.HasPrefix(arg, "--"):
			argIsValue = flagHasArgs(cmd.Flags().Lookup(arg[2:]))
		case strings.HasPrefix(arg, "-"):
			argIsValue = flagHasArgs(cmd.Flags().ShorthandLookup(arg[1:]))
		case argIsValue:
			argIsValue = false
		default:
			return arg
		}
	}

	// If we arrive here, we can safely assume that the command was called
	// with no provider at all. This means that we default to local.
	return "local"
}

func attacheProviders(existing providers.Providers, commands []*Command) {
	for i := range commands {
		attachProvidersToCmd(existing, commands[i])
	}
}

func attachProvidersToCmd(existing providers.Providers, cmd *Command) {
	for _, provider := range existing {
		for j := range provider.Connectors {
			conn := provider.Connectors[j]
			attachConnectorCmd(provider.Provider, &conn, cmd)
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
		Use:   connector.Use,
		Short: cmd.Action + connector.Short,
		Long:  connector.Long,
	}

	cmd.Command.Flags().VisitAll(func(flag *pflag.Flag) {
		res.Flags().AddFlag(flag)
	})

	cmd.Command.AddCommand(res)
	setConnector(provider, connector, cmd.Run, res)
}

func setConnector(provider *plugin.Provider, connector *plugin.Connector, run func(*cobra.Command, *providers.Runtime, *plugin.ParseCLIRes), cmd *cobra.Command) {
	oldRun := cmd.Run
	oldPreRun := cmd.PreRun

	supportedDiscoveries := append([]string{"all", "auto"}, connector.Discovery...)

	builtinFlags := []plugin.Flag{
		{
			Long: "discover",
			Type: plugin.FlagType_List,
			Desc: "Enable the discovery of nested assets. Supports: " + strings.Join(supportedDiscoveries, ","),
		},
		{
			Long: "record",
			Type: plugin.FlagType_String,
			Desc: "Record all resouce calls and use resouces in the recording",
		},
		{
			Long: "use-recording",
			Type: plugin.FlagType_String,
			Desc: "Use a recording to inject resouces data (read-only)",
		},
		{
			Long:   "pretty",
			Type:   plugin.FlagType_Bool,
			Desc:   "Pretty-print JSON",
			Option: plugin.FlagOption_Hidden,
		},
	}

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

		// the following flags are not processed by the provider; we handle them
		// here instead
		skipFlags := map[string]struct{}{
			"ask-pass":      {},
			"record":        {},
			"use-recording": {},
		}

		flagVals := map[string]*llx.Primitive{}
		for i := range allFlags {
			flag := allFlags[i]

			// we skip these because they are coded above
			if _, skip := skipFlags[flag.Long]; skip {
				continue
			}

			switch flag.Type {
			case plugin.FlagType_Bool:
				if v, err := cmd.Flags().GetBool(flag.Long); err == nil {
					flagVals[flag.Long] = llx.BoolPrimitive(v)
				}
			case plugin.FlagType_Int:
				if v, err := cmd.Flags().GetInt(flag.Long); err == nil {
					flagVals[flag.Long] = llx.IntPrimitive(int64(v))
				}
			case plugin.FlagType_String:
				if v, err := cmd.Flags().GetString(flag.Long); err == nil {
					flagVals[flag.Long] = llx.StringPrimitive(v)
				}
			case plugin.FlagType_List:
				if v, err := cmd.Flags().GetStringSlice(flag.Long); err == nil {
					flagVals[flag.Long] = llx.ArrayPrimitiveT(v, llx.StringPrimitive, types.String)
				}
			case plugin.FlagType_KeyValue:
				if v, err := cmd.Flags().GetStringToString(flag.Long); err == nil {
					flagVals[flag.Long] = llx.MapPrimitiveT(v, llx.StringPrimitive, types.String)
				}
			}
		}

		runtime := providers.Coordinator.NewRuntime()
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

		runtime.Recording, err = providers.NewRecording(recordingPath, providers.RecordingOptions{
			DoRecord:        record != "",
			PrettyPrintJSON: pretty,
		})
		if err != nil {
			log.Fatal().Msg(err.Error())
		}

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
		}

		run(cc, runtime, cliRes)
		runtime.Close()
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
