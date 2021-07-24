package ansibleinventory

import (
	"errors"
	"strconv"

	"github.com/mitchellh/mapstructure"
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

	res := []*Host{}
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

			res = append(res, host)
		}
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
