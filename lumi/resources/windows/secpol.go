package windows

import (
	"io"

	"github.com/pkg/errors"
	"gopkg.in/ini.v1"
)

type Secpol struct {
	SystemAccess    map[string]string
	EventAudit      map[string]string
	RegistryValues  map[string]string
	PrivilegeRights map[string]string
}

func ParseSecpol(r io.Reader) (*Secpol, error) {
	res := &Secpol{
		SystemAccess:    map[string]string{},
		EventAudit:      map[string]string{},
		RegistryValues:  map[string]string{},
		PrivilegeRights: map[string]string{},
	}

	cfg, err := ini.Load(r)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse secpol")
	}

	sysAccess, err := cfg.GetSection("System Access")
	if err != nil {
		return nil, err
	}
	keys := sysAccess.Keys()
	for i := range keys {
		entry := keys[i]
		res.SystemAccess[entry.Name()] = entry.Value()
	}

	eventAudit, err := cfg.GetSection("Event Audit")
	if err != nil {
		return nil, err
	}
	keys = eventAudit.Keys()
	for i := range keys {
		entry := keys[i]
		res.EventAudit[entry.Name()] = entry.Value()
	}

	registryValues, err := cfg.GetSection("Registry Values")
	if err != nil {
		return nil, err
	}
	keys = registryValues.Keys()
	for i := range keys {
		entry := keys[i]
		res.RegistryValues[entry.Name()] = entry.Value()
	}

	priviledgeRights, err := cfg.GetSection("Privilege Rights")
	if err != nil {
		return nil, err
	}
	keys = priviledgeRights.Keys()
	for i := range keys {
		entry := keys[i]
		res.PrivilegeRights[entry.Name()] = entry.Value()
	}

	return res, nil
}
