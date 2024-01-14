// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ansibleinventory

import (
	"errors"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
	"sigs.k8s.io/yaml"
)

type Group struct {
	Hosts []string
}

type Groups map[string]Group

type Meta struct {
	HostVars map[string]map[string]interface{}
}

type All struct {
	Children []string
}

func Parse(data []byte) (*Inventory, error) {
	inventory := Inventory{}
	err := inventory.Decode(data)
	if err != nil {
		return nil, err
	}
	return &inventory, nil
}

type Inventory struct {
	Meta Meta
	All  All
	Groups
}

func IsInventory(data []byte) bool {
	// parse json to map[string]interface{}
	var raw map[string]interface{}
	err := yaml.Unmarshal(data, &raw)
	if err != nil {
		return false
	}

	// if the all key is there, its a ansible yaml
	// NOTE: as this point we only support fully resolved ansible config
	_, ok := raw["all"]
	if ok {
		return true
	}
	return false
}

func (i *Inventory) Decode(data []byte) error {
	if i == nil {
		return errors.New("object cannot be nil")
	}

	// parse json to map[string]interface{}
	var raw map[string]interface{}
	err := yaml.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	var meta Meta
	err = mapstructure.Decode(raw["_meta"], &meta)
	if err != nil {
		return err
	}
	delete(raw, "_meta")
	i.Meta = meta

	var all All
	err = mapstructure.Decode(raw["all"], &all)
	if err != nil {
		return err
	}
	delete(raw, "all")
	i.All = all

	// assign all other entries to groups
	var groups Groups
	err = mapstructure.Decode(raw, &groups)
	if err != nil {
		return err
	}
	i.Groups = groups

	return nil
}

type Host struct {
	Alias      string
	Host       string // ansible_host
	Port       string // ansible_port
	User       string // ansible_user
	Password   string // ansible_password
	Identity   string // ansible_ssh_private_key_file
	Become     bool   // ansible_become
	Connection string // ansible_connection: ssh, local, docker
	Groups     []string
	Labels     []string
}

// https://docs.ansible.com/ansible/latest/user_guide/intro_inventory.html
func (inventory *Inventory) List(groups ...string) []*Host {
	if inventory == nil {
		return nil
	}

	list := inventory.All.Children
	if len(groups) > 0 {
		list = Filter(list, func(x string) bool {
			for i := range groups {
				if groups[i] == x {
					return true
				}
			}
			return false
		})
	}

	hostMap := map[string]*Host{}
	for i := range list {
		groupname := list[i]
		hosts := inventory.Groups[groupname].Hosts
		for j := range hosts {
			alias := hosts[j]

			host := &Host{
				Alias:      alias,
				Host:       alias,
				Connection: "ssh",
			}

			meta := inventory.Meta.HostVars[alias]

			if d, ok := meta["ansible_host"]; ok {
				host.Host = d.(string)
			}

			if f, ok := meta["ansible_port"]; ok {
				s := strconv.FormatFloat(f.(float64), 'f', 0, 64)
				host.Port = s
			}

			if d, ok := meta["ansible_user"]; ok {
				host.User = d.(string)
			}

			if d, ok := meta["ansible_password"]; ok {
				host.Password = d.(string)
			}

			if d, ok := meta["ansible_ssh_private_key_file"]; ok {
				host.Identity = d.(string)
			}

			if d, ok := meta["ansible_connection"]; ok {
				host.Connection = d.(string)
			}

			if d, ok := meta["tags"]; ok {
				labels, ok := d.([]interface{})
				if ok {
					for i := range labels {
						key, kok := labels[i].(string)
						if kok {
							host.Labels = append(host.Labels, key)
						}
					}
				}
			}

			hostMap[alias] = host
		}
	}

	res := []*Host{}

	for k := range hostMap {
		res = append(res, hostMap[k])
	}

	return res
}

func Filter(vs []string, f func(string) bool) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

func (i *Inventory) ToV1Inventory() *inventory.Inventory {
	out := inventory.New()

	// convert assets
	hosts := i.List()
	for i := range hosts {
		host := hosts[i]

		name := host.Host
		if host.Alias != "" {
			name = host.Alias
		}

		asset := &inventory.Asset{
			Name:        name,
			Connections: ansibleConnections(host),
			Labels:      map[string]string{},
		}

		for l := range host.Labels {
			key := host.Labels[l]
			asset.Labels[key] = ""
		}

		out.Spec.Assets = append(out.Spec.Assets, asset)
	}

	// move credentials out into credentials section
	out.PreProcess()

	return out
}

var validConnectionTypes = []string{"ssh", "winrm", "local", "docker"}

func isValidConnectionType(conn string) bool {
	for i := range validConnectionTypes {
		if conn == validConnectionTypes[i] {
			return true
		}
	}
	return false
}

// ansibleBackend maps an ansible connection to mondoo backend
// https://docs.ansible.com/ansible/latest/plugins/connection.html
// quickly get a list of available plugins via `ansible-doc -t connection -l`
func ansibleBackend(connection string) string {
	switch strings.TrimSpace(connection) {
	case "local":
		break
	case "ssh":
		break
	case "winrm":
		break
	case "docker":
		break
	default:
		log.Warn().Str("ansible-connection", connection).Msg("unknown connection, fallback to ssh")
		return "ssh"
	}
	return connection
}

func ansibleConnections(host *Host) []*inventory.Config {
	backend := ansibleBackend(host.Connection)

	// in the case where the port is 0, we will fallback to default ports (eg 22)
	// further down in the execution chain
	port, _ := strconv.Atoi(host.Port)

	res := &inventory.Config{
		Type: backend,
		Host: host.Host,
		Port: int32(port),
		Sudo: &inventory.Sudo{
			Active: host.Become,
		},
	}

	credentials := []*vault.Credential{}

	if host.Password != "" {
		credentials = append(credentials, &vault.Credential{
			Type:     vault.CredentialType_password,
			User:     host.User,
			Password: host.Password,
		})
	}

	if host.Identity != "" {
		credentials = append(credentials, &vault.Credential{
			Type:           vault.CredentialType_private_key,
			User:           host.User,
			PrivateKeyPath: host.Identity,
		})
	}

	// fallback to ssh agent as default in case nothing was provided
	if len(credentials) == 0 && backend == "ssh" {
		credentials = append(credentials, &vault.Credential{
			Type: vault.CredentialType_ssh_agent,
			User: host.User,
		})
	}

	res.Credentials = credentials
	return []*inventory.Config{res}
}
