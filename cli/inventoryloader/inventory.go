package inventoryloader

import (
	"bytes"
	"io"
	"os"
	"runtime"

	"errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/inventory/ansibleinventory"
	"go.mondoo.com/cnquery/motor/inventory/domainlist"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
)

func loadDataPipe() ([]byte, bool) {
	isTerminal := true
	isNamedPipe := false
	switch runtime.GOOS {
	case "darwin", "dragonfly", "netbsd", "solaris", "linux":
		// when we run the following command, the detection differs between macos and linux
		// cat options.json | mondoo scan
		// for macos, we get isNamedPipe=false, isTerminal=false, size > 0
		// but this only applies to direct terminal execution, for the same command in a bash file, we get
		// for macos bash script, we get isNamedPipe=true, isTerminal=false, size > 0
		// for linux, we get isNamedPipe=true, isTerminal=false, size=0
		// Therefore we always want to check for file size if we detected its not a terminal
		// If we are not checking for fi.Size() > 0 even a run inside of a bash script turn out
		// to be pipes, therefore we need to verify that there is some data available at the pipe
		// also read https://flaviocopes.com/go-shell-pipes/
		fi, _ := os.Stdin.Stat()
		isTerminal = (fi.Mode() & os.ModeCharDevice) == os.ModeCharDevice
		isNamedPipe = (fi.Mode() & os.ModeNamedPipe) == os.ModeNamedPipe
		log.Debug().Bool("isTerminal", isTerminal).Bool("isNamedPipe", isNamedPipe).Int64("size", fi.Size()).Msg("check if we got the scan config from pipe")
		if isNamedPipe || (!isTerminal && fi.Size() > 0) {
			// Pipe input
			log.Debug().Msg("read scan config from stdin pipe")

			// read stdin into buffer
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				log.Error().Err(err).Msg("could not read from pipe")
				return nil, false
			}
			return data, true
		}
	}
	return nil, false
}

// Parse uses the viper flags for `--inventory-file` to load the inventory
func Parse() (*v1.Inventory, error) {
	inventoryFilePath := viper.GetString("inventory-file")

	// check in an inventory file was provided
	if inventoryFilePath == "" {
		return v1.New(), nil
	}

	var data []byte
	var err error

	// for we just read the data from the input file
	if inventoryFilePath != "-" {
		log.Info().Str("inventory-file", inventoryFilePath).Msg("load inventory")
		data, err = os.ReadFile(inventoryFilePath)
		if err != nil {
			return nil, err
		}
	} else {
		log.Info().Msg("load inventory from piped input")
		var ok bool
		data, ok = loadDataPipe()
		if !ok {
			return nil, errors.New("could not read inventory from piped input")
		}
	}

	// force detection
	if viper.GetBool("inventory-ansible") {
		log.Debug().Msg("parse ansible inventory")
		inventory, err := parseAnsibleInventory(data)
		if err != nil {
			return nil, err
		}
		return inventory, nil
	}

	if viper.GetBool("inventory-domainlist") {
		log.Debug().Msg("parse domainlist inventory")
		inventory, err := parseDomainListInventory(data)
		if err != nil {
			return nil, err
		}
		return inventory, nil
	}

	// load mondoo inventory
	log.Debug().Msg("parse inventory")
	inventory, err := v1.InventoryFromYAML(data)
	if err != nil {
		return nil, err
	}
	// we preprocess the content here, to ensure relative paths are
	if inventory.Metadata.Labels == nil {
		inventory.Metadata.Labels = map[string]string{}
	}
	inventory.Metadata.Labels[v1.InventoryFilePath] = inventoryFilePath
	err = inventory.PreProcess()
	if err != nil {
		return nil, err
	}

	return inventory, nil
}

func parseAnsibleInventory(data []byte) (*v1.Inventory, error) {
	inventory, err := ansibleinventory.Parse(data)
	if err != nil {
		return nil, err
	}
	return inventory.ToV1Inventory(), nil
}

func parseDomainListInventory(data []byte) (*v1.Inventory, error) {
	inventory, err := domainlist.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return inventory.ToV1Inventory(), nil
}

// ParseOrUse tries to load the inventory and if nothing exists it
// will instead use the provided asset.
func ParseOrUse(cliAsset *asset.Asset, insecure bool) (*v1.Inventory, error) {
	var v1inventory *v1.Inventory
	var err error

	// parses optional inventory file if inventory was not piped already
	v1inventory, err = Parse()
	if err != nil {
		return nil, errors.Join(err, errors.New("could not parse inventory"))
	}

	// add asset from cli to inventory
	if (len(v1inventory.Spec.GetAssets()) == 0) && cliAsset != nil {
		v1inventory.AddAssets(cliAsset)
	}

	// if the --insecure flag is set, we overwrite the individual setting for the asset
	if insecure == true {
		v1inventory.MarkConnectionsInsecure()
	}

	return v1inventory, nil
}
