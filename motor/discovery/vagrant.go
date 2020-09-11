package discovery

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/apps/mondoo/cmd/options"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/local"
)

type VagrantVmSSHConfig struct {
	Host     string
	HostName string
	User     string
	Port     int
	// eg /dev/null
	UserKnownHostsFile string
	// enabled StrictHostKeyChecking - "yes" || "no"
	StrictHostKeyChecking string
	// enabled password authentication - "yes" || "no"
	PasswordAuthentication string
	// eg. .vagrant/machines/default/virtualbox/private_key
	IdentityFile string
	//  "yes" || "no"
	IdentitiesOnly string
	LogLevel       string
}

func ParseVagrantSshConfig(r io.Reader) (map[string]*VagrantVmSSHConfig, error) {
	res := make(map[string]*VagrantVmSSHConfig)
	scanner := bufio.NewScanner(r)

	var config *VagrantVmSSHConfig
	for scanner.Scan() {
		line := scanner.Text()
		log.Debug().Msg(line)

		fields := strings.Fields(strings.TrimSpace(line))

		if len(fields) == 2 {
			switch fields[0] {
			case "Host":
				if config != nil {
					res[config.Host] = config
				}
				config = &VagrantVmSSHConfig{}
				config.Host = fields[1]
			case "HostName":
				config.HostName = fields[1]
			case "IdentitiesOnly":
				config.IdentitiesOnly = fields[1]
			case "IdentityFile":
				config.IdentityFile = fields[1]
			case "LogLevel":
				config.LogLevel = fields[1]
			case "PasswordAuthentication":
				config.PasswordAuthentication = fields[1]
			case "Port":
				config.Port, _ = strconv.Atoi(fields[1])
			case "StrictHostKeyChecking":
				config.StrictHostKeyChecking = fields[1]
			case "User":
				config.User = fields[1]
			case "UserKnownHostsFile":
				config.UserKnownHostsFile = fields[1]
			}
		}
	}

	// add the last element
	if config != nil {
		res[config.Host] = config
	}

	return res, nil
}

var ParseVagrantStatusRegex = regexp.MustCompile(`^(.*?)\s+(not created|running)\s(?:.*)$`)

func ParseVagrantStatus(r io.Reader) (map[string]bool, error) {
	res := make(map[string]bool)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		m := ParseVagrantStatusRegex.FindStringSubmatch(line)
		if len(m) == 3 {
			running := false

			if m[2] == "running" {
				running = true
			}

			res[m[1]] = running
		}

	}
	return res, nil
}

type vagrantResolver struct{}

func (k *vagrantResolver) Name() string {
	return "Vagrant Resolver"
}

type vagrantContext struct {
	Host string
}

func (v *vagrantResolver) Resolve(in *options.VulnOptsAsset, opts *options.VulnOpts) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// parse context from url
	context := v.ParseContext(in.Connection)

	t, err := local.New()
	if err != nil {
		return nil, err
	}
	m, err := motor.New(t)
	if err != nil {
		return nil, err
	}

	// we run status first, since vagrant ssh-config does not return a proper state
	// if in a multi-vm setup not all vms are running
	cmd, err := m.Transport.RunCommand("vagrant status")
	if err != nil {
		return nil, err
	}

	vmStatus, err := ParseVagrantStatus(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	// filter vms if a context was provided
	if len(context.Host) > 0 {
		k := context.Host
		vm, ok := vmStatus[k]
		if !ok {
			return nil, errors.New("could not find vagrant host: " + k)
		}

		if !vm {
			return nil, errors.New("vm is not ready: " + k)
		}

		cmd, err := m.Transport.RunCommand("vagrant ssh-config " + k)
		if err != nil {
			return nil, err
		}

		vmSshConfig, err := ParseVagrantSshConfig(cmd.Stdout)
		if err != nil {
			return nil, err
		}

		resolved = append(resolved, vagrantToAsset(vmSshConfig[k], opts))

	} else {
		vagrantVms := map[string]*VagrantVmSSHConfig{}
		for k := range vmStatus {
			if vmStatus[k] == false {
				log.Debug().Str("vm", k).Msg("skip vm since it is not running")
				continue
			}

			log.Debug().Str("vm", k).Msg("gather ssh config")
			cmd, err := m.Transport.RunCommand("vagrant ssh-config " + k)
			if err != nil {
				return nil, err
			}

			vmSshConfig, err := ParseVagrantSshConfig(cmd.Stdout)
			if err != nil {
				return nil, err
			}

			for k := range vmSshConfig {
				vagrantVms[k] = vmSshConfig[k]
			}
		}

		for i := range vagrantVms {
			resolved = append(resolved, vagrantToAsset(vagrantVms[i], opts))
		}
	}

	return resolved, nil
}

func vagrantToAsset(sshConfig *VagrantVmSSHConfig, opts *options.VulnOpts) *asset.Asset {
	if sshConfig == nil {
		return nil
	}

	return &asset.Asset{
		Name: sshConfig.Host,
		Connections: []*transports.TransportConfig{{
			// TODO: do we need to support winrm?
			Backend:       transports.TransportBackend_CONNECTION_SSH,
			Host:          sshConfig.HostName,
			IdentityFiles: []string{sshConfig.IdentityFile},
			Insecure:      strings.ToLower(sshConfig.StrictHostKeyChecking) == "no",
			User:          sshConfig.User,
			Port:          strconv.Itoa(sshConfig.Port),
			Sudo: &transports.Sudo{
				Active: opts.Sudo.Active,
			},
		}},
		Platform: &platform.Platform{
			Kind: transports.Kind_KIND_VIRTUAL_MACHINE,
		},
	}
}

func (v *vagrantResolver) ParseContext(connection string) vagrantContext {
	var config vagrantContext

	connection = strings.TrimPrefix(connection, "vagrant://")
	config.Host = connection
	return config
}
