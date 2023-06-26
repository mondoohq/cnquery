package providers

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.mondoo.com/cnquery/providers"
	"go.mondoo.com/cnquery/providers/plugin"
)

type Command struct {
	Command *cobra.Command
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
			attachConnectorToCmd(provider.Provider, &conn, cmd)
		}
	}
}

func attachConnectorToCmd(provider *plugin.Provider, connector *plugin.Connector, cmd *Command) {
	res := &cobra.Command{
		Use:   connector.Use,
		Short: cmd.Action + connector.Short,
		Long:  connector.Long,
	}

	for i := range connector.Flags {
		flag := connector.Flags[i]
		switch flag.Type {
		case plugin.FlagType_Bool:
			if flag.Short != "" {
				res.Flags().BoolP(flag.Long, flag.Short, json2T(flag.Default, false), flag.Desc)
			} else {
				res.Flags().Bool(flag.Long, json2T(flag.Default, false), flag.Desc)
			}
		case plugin.FlagType_Int:
			if flag.Short != "" {
				res.Flags().IntP(flag.Long, flag.Short, json2T(flag.Default, 0), flag.Desc)
			} else {
				res.Flags().Int(flag.Long, json2T(flag.Default, 0), flag.Desc)
			}
		case plugin.FlagType_String:
			if flag.Short != "" {
				res.Flags().StringP(flag.Long, flag.Short, flag.Default, flag.Desc)
			} else {
				res.Flags().String(flag.Long, flag.Default, flag.Desc)
			}
		case plugin.FlagType_List:
			if flag.Short != "" {
				res.Flags().StringSliceP(flag.Long, flag.Short, json2T(flag.Default, []string{}), flag.Desc)
			} else {
				res.Flags().StringSlice(flag.Long, json2T(flag.Default, []string{}), flag.Desc)
			}
		case plugin.FlagType_KeyValue:
			if flag.Short != "" {
				res.Flags().StringToStringP(flag.Long, flag.Short, json2T(flag.Default, map[string]string{}), flag.Desc)
			} else {
				res.Flags().StringToString(flag.Long, json2T(flag.Default, map[string]string{}), flag.Desc)
			}
		}

		if flag.Option&plugin.FlagOption_Hidden != 0 {
			res.Flags().MarkHidden(flag.Long)
		}
		if flag.Option&plugin.FlagOption_Deprecated != 0 {
			res.Flags().MarkDeprecated(flag.Long, "has been deprecated")
		}
	}

	cmd.Command.AddCommand(res)
}

func json2T[T any](s string, empty T) T {
	var res T
	if err := json.Unmarshal([]byte(s), &res); err == nil {
		return res
	}
	return empty
}
