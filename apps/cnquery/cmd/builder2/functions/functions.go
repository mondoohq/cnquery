package functions

import (
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder2/common"
	"go.mondoo.com/cnquery/motor/providers"
)

type Subcommand struct {
	name         string
	providerType providers.ProviderType
	assetType    common.AssetType
}

var subcommandList = []Subcommand{
	{
		name:         "azure",
		providerType: providers.ProviderType_AZURE,
		assetType:    common.DefaultAssetType,
	},
	{
		name:         "local",
		providerType: providers.ProviderType_LOCAL_OS,
		assetType:    common.DefaultAssetType,
	},
	{
		name:         "inventory-file",
		providerType: providers.ProviderType_UNKNOWN,
		assetType:    common.DefaultAssetType,
	},
}

func NewFunctionMap(runFn func(*cobra.Command, []string, providers.ProviderType, common.AssetType)) common.SubcommandFnMap {
	subcmdFnMap := common.SubcommandFnMap{}

	for i := range subcommandList {
		subcmdFnMap[subcommandList[i].name] = createCallbackFn(subcommandList[i], runFn)
	}

	return subcmdFnMap
}

func createCallbackFn(subcmd Subcommand, runFn func(*cobra.Command, []string, providers.ProviderType, common.AssetType)) common.RunFn {
	fn := func(cmd *cobra.Command, args []string) {
		runFn(cmd, args, subcmd.providerType, subcmd.assetType)
	}

	return fn
}
