package windows

import (
	"io"
	"strconv"
	"strings"

	"github.com/pkg/errors"
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
		return nil, errors.Wrap(err, "could not parse secpol")
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

		// try to parse the content
		i, err := strconv.ParseInt(rawValue, 10, 64)
		if err == nil {
			res.SystemAccess[key] = i
		} else {
			res.SystemAccess[key] = rawValue
		}
	}

	eventAudit, err := cfg.GetSection("Event Audit")
	if err != nil {
		return nil, err
	}
	keys = eventAudit.Keys()
	for i := range keys {
		entry := keys[i]

		rawValue := entry.Value()
		i, err := strconv.ParseInt(rawValue, 10, 64)
		if err == nil {
			res.EventAudit[entry.Name()] = i
		} else {
			res.EventAudit[entry.Name()] = rawValue
		}
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
		rawValue := entry.Value()
		values := strings.Split(rawValue, ",")

		for i := range values {
			val := values[i]
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
