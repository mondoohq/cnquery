package resources

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/resources/packages"
	"go.mondoo.io/mondoo/lumi/resources/uptime"
)

func (p *lumiOs) id() (string, error) {
	return "os", nil
}

func (p *lumiOs) GetRebootpending() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (p *lumiOs) GetEnv() (map[string]interface{}, error) {
	rawCmd, err := p.Runtime.CreateResource("command", "command", "env")
	if err != nil {
		return nil, err
	}
	cmd := rawCmd.(Command)

	out, err := cmd.Stdout()
	if err != nil {
		return nil, err
	}

	res := map[string]interface{}{}
	lines := strings.Split(out, "\n")
	for i := range lines {
		parts := strings.SplitN(lines[i], "=", 2)
		if len(parts) != 2 {
			continue
		}
		res[parts[0]] = parts[1]
	}

	return res, nil
}

func (p *lumiOs) GetPath() ([]interface{}, error) {
	env, err := p.Env()
	if err != nil {
		return nil, err
	}

	rawPath, ok := env["PATH"]
	if !ok {
		return []interface{}{}, nil
	}

	path := rawPath.(string)
	parts := strings.Split(path, ":")
	res := make([]interface{}, len(parts))
	for i := range parts {
		res[i] = parts[i]
	}

	return res, nil
}

func (p *lumiOs) GetUptime() (int64, error) {
	uptime, err := uptime.New(p.Runtime.Motor)
	if err != nil {
		return 0, err
	}

	t, err := uptime.Duration()
	if err != nil {
		return 0, err
	}
	return int64(t), nil
}

// func (p *lumiOs) GetRebootpending() ([]interface{}, error) {
// 	return nil, errors.New("not implemented")
// }

func (p *lumiOsupdate) id() (string, error) {
	name, _ := p.Name()
	return name, nil
}

func (p *lumiOs) GetUpdates() ([]interface{}, error) {
	// find suitable system updates
	um, err := packages.ResolveSystemUpdateManager(p.Runtime.Motor)
	if um == nil || err != nil {
		return nil, fmt.Errorf("Could not detect suiteable update manager for platform")
	}

	// retrieve all system updates
	updates, err := um.List()
	if err != nil {
		return nil, fmt.Errorf("Could not retrieve updates list for platform")
	}

	// create lumi update resources for each update
	osupdates := make([]interface{}, len(updates))
	log.Debug().Int("updates", len(updates)).Msg("lumi[updates]> found system updates")
	for i, update := range updates {

		lumiOsUpdate, err := p.Runtime.CreateResource("update",
			"name", update.Name,
			"severity", update.Severity,
			"category", update.Category,
			"restart", update.Restart,
			"format", um.Format(),
		)
		if err != nil {
			return nil, err
		}

		osupdates[i] = lumiOsUpdate.(Osupdate)
	}

	// return the packages as new entries
	return osupdates, nil
}
