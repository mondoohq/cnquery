package windows

import (
	"io"
	"sort"
	"strings"

	"errors"
	"gopkg.in/ini.v1"
)

type Secpol struct {
	SystemAccess    map[string]interface{}
	EventAudit      map[string]interface{}
	RegistryValues  map[string]interface{}
	PrivilegeRights map[string]interface{}
}

func ParseSecpol(r io.Reader) (*Secpol, error) {
	res := &Secpol{
		SystemAccess:    map[string]interface{}{}, // except for NewAdministratorName & NewGuestName, parse everything as int64
		EventAudit:      map[string]interface{}{}, // parse to int
		RegistryValues:  map[string]interface{}{}, // keep strings
		PrivilegeRights: map[string]interface{}{}, // split entries with ,
	}

	cfg, err := ini.Load(r)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not parse secpol"))
	}

	sysAccess, err := cfg.GetSection("System Access")
	if err != nil {
		return nil, err
	}
	keys := sysAccess.Keys()
	for i := range keys {
		entry := keys[i]
		key := entry.Name()
		rawValue := entry.Value()

		if key == "NewAdministratorName" || key == "NewGuestName" {
			res.SystemAccess[key] = rawValue
			continue
		}

		res.SystemAccess[key] = rawValue
	}

	eventAudit, err := cfg.GetSection("Event Audit")
	if err != nil {
		return nil, err
	}
	keys = eventAudit.Keys()
	for i := range keys {
		entry := keys[i]

		rawValue := entry.Value()
		res.EventAudit[entry.Name()] = rawValue
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

	privilegeRights, err := cfg.GetSection("Privilege Rights")
	if err != nil {
		return nil, err
	}
	keys = privilegeRights.Keys()
	for i := range keys {
		entry := keys[i]
		rawValue := entry.Value()

		valuesT := strings.Split(rawValue, ",")
		sort.Sort(sort.StringSlice(valuesT))

		values := make([]interface{}, len(valuesT))
		for i := range valuesT {
			val := valuesT[i]
			val = strings.Replace(val, "*S", "S", 1)
			values[i] = val
		}

		res.PrivilegeRights[entry.Name()] = values
	}

	return res, nil
}

const SecpolScript = `
secedit /export /cfg out.cfg  | Out-Null
$raw = Get-Content out.cfg
Remove-Item .\out.cfg | Out-Null
Write-Output $raw
`
