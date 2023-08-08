package resources

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/os/resources/powershell"
	"go.mondoo.com/cnquery/providers/os/resources/windows"
)

func (k *mqlRegistrykey) id() (string, error) {
	return k.Path.Data, nil
}

func (k *mqlRegistrykey) exists() (bool, error) {
	script := powershell.Encode(windows.GetRegistryKeyItemScript(k.Path.Data))
	o, err := CreateResource(k.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData(script),
	})
	if err != nil {
		log.Error().Err(err).Msg("could not create resource")
		return false, err
	}
	cmd := o.(*mqlCommand)

	exit := cmd.GetExitcode()
	if exit.Error != nil {
		return false, exit.Error
	}
	if exit.Data != 0 {
		stderr := cmd.GetStderr()
		// this would be an expected error and would ensure that we do not throw an error on windows systems
		// TODO: revisit how this is handled for non-english systems
		if err == nil && (strings.Contains(stderr.Data, "not exist") || strings.Contains(stderr.Data, "ObjectNotFound")) {
			return false, nil
		}

		return false, errors.New("could not retrieve registry key")
	}
	return true, nil
}

func (k *mqlRegistrykey) properties() (map[string]interface{}, error) {
	res := map[string]interface{}{}
	script := powershell.Encode(windows.GetRegistryKeyItemScript(k.Path.Data))
	o, err := CreateResource(k.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData(script),
	})
	if err != nil {
		return nil, err
	}
	cmd := o.(*mqlCommand)

	exit := cmd.GetExitcode()
	if exit.Error != nil {
		return nil, exit.Error
	}
	if exit.Data != 0 {
		return nil, errors.New("could not retrieve registry key")
	}

	entries, err := windows.ParseRegistryKeyItems(strings.NewReader(cmd.GetStdout().Data))
	if err != nil {
		return nil, err
	}

	for i := range entries {
		rkey := entries[i]
		res[rkey.Key] = rkey.GetValue()
	}

	return res, nil
}

func (k *mqlRegistrykey) children() ([]interface{}, error) {
	res := []interface{}{}

	script := powershell.Encode(windows.GetRegistryKeyChildItemsScript(k.Path.Data))
	o, err := CreateResource(k.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData(script),
	})
	if err != nil {
		return res, err
	}
	cmd := o.(*mqlCommand)

	exit := cmd.GetExitcode()
	if exit.Error != nil {
		return nil, exit.Error
	}
	if exit.Data != 0 {
		return nil, errors.New("could not retrieve registry key")
	}

	children, err := windows.ParseRegistryKeyChildren(strings.NewReader(cmd.GetStdout().Data))
	if err != nil {
		return nil, err
	}

	for i := range children {
		child := children[i]
		res = append(res, child.Path)
	}

	return res, nil
}

type mqlRegistrykeyPropertyInternal struct {
	key plugin.TValue[*mqlRegistrykey]
}

func (p *mqlRegistrykeyProperty) id() (string, error) {
	return p.Path.Data + " - " + p.Name.Data, nil
}

func (p *mqlRegistrykeyProperty) lookupKey() (*mqlRegistrykey, error) {
	if p.key.State == plugin.StateIsSet {
		return p.key.Data, p.key.Error
	}

	// create resource here, but do not use it yet
	obj, err := CreateResource(p.MqlRuntime, "registrykey", map[string]*llx.RawData{
		"path": llx.StringData(p.Path.Data),
	})
	if err != nil {
		p.key = plugin.TValue[*mqlRegistrykey]{Error: err, State: plugin.StateIsSet}
		return p.key.Data, p.key.Error
	}

	registryKey := obj.(*mqlRegistrykey)
	p.key = plugin.TValue[*mqlRegistrykey]{Data: registryKey, State: plugin.StateIsSet}
	return p.key.Data, p.key.Error
}

func (p *mqlRegistrykeyProperty) exists() (bool, error) {
	key, err := p.lookupKey()
	if err != nil {
		return false, err
	}

	exists := key.GetExists()
	if exists.Error != nil {
		return false, exists.Error
	}

	return exists.Data, nil
}

func (p *mqlRegistrykeyProperty) value(exists bool) (string, error) {
	if !exists {
		return "", nil
	}

	key, err := p.lookupKey()
	if err != nil {
		return "", err
	}

	props := key.GetProperties()
	if props.Error != nil {
		return "", props.Error
	}

	// search for property
	found := ""
	for k := range props.Data {
		if strings.EqualFold(k, p.Name.Data) {
			found = props.Data[k].(string)
			break
		}
	}

	return found, nil
}
