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
	"go.mondoo.com/cnquery/providers/plugin"
	"go.mondoo.com/cnquery/providers/proto"
	"go.mondoo.com/cnquery/types"
)

type Command struct {
	Command *cobra.Command
	Run     func(*cobra.Command, *providers.Runtime, *proto.ParseCLIRes)
	Action  string
}

// ProcessCLI will attempt to parse the current commandline and look for providers.
// This step is done before cobra ever takes effect
func ProcessCLI(rootCmd *cobra.Command, commands ...*Command) error {
	existing, err := providers.List()
	if err != nil {
		return err
	}

	connectorName := detectConnector(os.Args, rootCmd, commands)
	if connectorName == "" {
		return nil
	}

	if _, err := ensureProvider(existing, connectorName); err != nil {
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

func ensureProvider(existing providers.Providers, connectorName string) (*providers.Provider, error) {
	provider := existing.ForConnection(connectorName)
	if provider != nil {
		return provider, nil
	}

	upstream := providers.DefaultProviders.ForConnection(connectorName)
	if upstream == nil {
		// we can't find any provider for this connector in our default set
		return nil, nil
	}

	nu, err := providers.Install(upstream.Name)
	existing.Add(nu)
	return nu, err
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
	if p, ok := existing["os"]; ok {
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
	cmd.Command.AddCommand(res)
	setConnector(provider, connector, cmd.Run, res)
}

func setConnector(provider *plugin.Provider, connector *plugin.Connector, run func(*cobra.Command, *providers.Runtime, *proto.ParseCLIRes), cmd *cobra.Command) {
	oldRun := cmd.Run
	oldPreRun := cmd.PreRun

	supportedDiscoveries := append([]string{"all", "auto"}, connector.Discovery...)

	builtinFlags := []plugin.Flag{
		{
			Long: "discover",
			Type: plugin.FlagType_List,
			Desc: "Enable the discovery of nested assets. Supports: " + strings.Join(supportedDiscoveries, ","),
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

		flagVals := map[string]*llx.Primitive{}
		for i := range allFlags {
			flag := allFlags[i]

			// we skip this because it's coded above
			if flag.Long == "ask-pass" {
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
		if err := runtime.UseProvider(provider.Name); err != nil {
			providers.Coordinator.Shutdown()
			log.Fatal().Err(err).Msg("failed to start provider " + provider.Name)
		}

		cliRes, err := runtime.Provider.Plugin.ParseCLI(&proto.ParseCLIReq{
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
