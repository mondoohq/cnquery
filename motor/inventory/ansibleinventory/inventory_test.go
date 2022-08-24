package ansibleinventory_test

import (
	"io/ioutil"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/inventory/ansibleinventory"
	"go.mondoo.com/cnquery/motor/vault"
)

func TestValidInventory(t *testing.T) {
	assert.False(t, ansibleinventory.IsInventory([]byte{}))

	iniFile := `
[win]
172.16.2.5 
172.16.2.6 
	`
	assert.False(t, ansibleinventory.IsInventory([]byte(iniFile)))

	jsonFile := `
{
	"_meta": {
					"hostvars": {}
	},
	"ungrouped": {}
}
`
	assert.False(t, ansibleinventory.IsInventory([]byte(jsonFile)))

	jsonFile = `
{
	"all": {
		"children": [
				"local", 
				"ungrouped"
		]
	}
}
`
	assert.True(t, ansibleinventory.IsInventory([]byte(jsonFile)))
}

func TestParseInventory(t *testing.T) {
	input, err := ioutil.ReadFile("./testdata/empty.json")
	assert.Nil(t, err)
	assert.True(t, ansibleinventory.IsInventory(input))

	inventory := ansibleinventory.Inventory{}
	err = inventory.Decode(input)
	assert.Nil(t, err)
	assert.Equal(t, inventory.All.Children, []string{"ungrouped"})
}

func TestParseInventoryUngrouped(t *testing.T) {
	input, err := ioutil.ReadFile("./testdata/ungrouped.json")
	assert.Nil(t, err)
	assert.True(t, ansibleinventory.IsInventory(input))

	inventory := ansibleinventory.Inventory{}
	err = inventory.Decode(input)
	assert.Nil(t, err)

	assert.Equal(t, []string{"ungrouped", "workers"}, inventory.All.Children)
	assert.Equal(t, ansibleinventory.Group{Hosts: []string{"34.244.38.44"}}, inventory.Groups["ungrouped"])
}

func TestFullInventory(t *testing.T) {
	input, err := ioutil.ReadFile("./testdata/inventory.json")
	assert.Nil(t, err)
	assert.True(t, ansibleinventory.IsInventory(input))

	inventory := ansibleinventory.Inventory{}
	err = inventory.Decode(input)
	assert.Nil(t, err)

	assert.Equal(t, []string{
		"api",
		"aws_ec2",
		"payment",
		"ungrouped",
		"web",
		"webservers",
	}, inventory.All.Children)

	assert.Equal(t, []string{"192.168.2.1", "192.168.2.2"}, inventory.Groups["api"].Hosts)
	assert.Equal(t, []string{"ec2-34-242-192-191.eu-west-1.compute.amazonaws.com"}, inventory.Groups["aws_ec2"].Hosts)
}

func sortHosts(hosts []*ansibleinventory.Host) {
	sort.SliceStable(hosts, func(i, j int) bool {
		return hosts[i].Alias < hosts[j].Alias
	})
}

func TestHostExtraction(t *testing.T) {
	input, err := ioutil.ReadFile("./testdata/ungrouped.json")
	assert.Nil(t, err)
	assert.True(t, ansibleinventory.IsInventory(input))

	inventory := ansibleinventory.Inventory{}
	err = inventory.Decode(input)
	assert.Nil(t, err)

	hosts := inventory.List()
	assert.Equal(t, 3, len(hosts))

	hosts = inventory.List("ungrouped")
	assert.Equal(t, 1, len(hosts))

	assert.Equal(t, []*ansibleinventory.Host{{
		Alias:      "34.244.38.44",
		Host:       "34.244.38.44",
		Port:       "2222",
		Connection: "ssh",
	}}, hosts)

	hosts = inventory.List("workers")
	assert.Equal(t, 2, len(hosts))

	// ensure order for equality check
	sortHosts(hosts)

	assert.Equal(t, []*ansibleinventory.Host{{
		Alias:      "34.244.38.46",
		Host:       "34.244.38.46",
		User:       "ec2-user",
		Port:       "22",
		Connection: "ssh",
	}, {
		Alias:      "34.255.178.16",
		Host:       "34.255.178.16",
		User:       "ec2-user",
		Identity:   "/Users/chartmann/.ssh/id_rsa",
		Connection: "ssh",
	}}, hosts)
}

// convert the ini via
// ansible-inventory -i integrations/ansibleinventory/testdata/local.ini --list > integrations/ansibleinventory/testdata/local.json
func TestHostConnectionLocal(t *testing.T) {
	input, err := ioutil.ReadFile("./testdata/local.json")
	assert.Nil(t, err)
	assert.True(t, ansibleinventory.IsInventory(input))

	inventory := ansibleinventory.Inventory{}
	err = inventory.Decode(input)
	assert.Nil(t, err)

	hosts := inventory.List()
	assert.Equal(t, 1, len(hosts))

	hosts = inventory.List("local")
	assert.Equal(t, 1, len(hosts))

	assert.Equal(t, []*ansibleinventory.Host{{
		Alias:      "127.0.0.1",
		Host:       "127.0.0.1",
		Connection: "local",
	}}, hosts)
}

// yq -y . integrations/ansibleinventory/testdata/local.json
func TestHostConnectionLocalYaml(t *testing.T) {
	input, err := ioutil.ReadFile("./testdata/local.yaml")
	assert.Nil(t, err)
	assert.True(t, ansibleinventory.IsInventory(input))

	inventory := ansibleinventory.Inventory{}
	err = inventory.Decode(input)
	assert.Nil(t, err)

	hosts := inventory.List()
	assert.Equal(t, 1, len(hosts))

	hosts = inventory.List("local")
	assert.Equal(t, 1, len(hosts))

	assert.Equal(t, []*ansibleinventory.Host{{
		Alias:      "127.0.0.1",
		Host:       "127.0.0.1",
		Connection: "local",
	}}, hosts)
}

// convert winrm.ini via
// ansible-inventory -i integrations/ansibleinventory/testdata/windows.ini --list > integrations/ansibleinventory/testdata/windows.json
func TestHostConnectionWinrm(t *testing.T) {
	input, err := ioutil.ReadFile("./testdata/winrm.json")
	assert.Nil(t, err)
	assert.True(t, ansibleinventory.IsInventory(input))

	inventory := ansibleinventory.Inventory{}
	err = inventory.Decode(input)
	assert.Nil(t, err)

	hosts := inventory.List()
	assert.Equal(t, 2, len(hosts))

	hosts = inventory.List("win")
	assert.Equal(t, 2, len(hosts))

	// ensure order for equality check
	sortHosts(hosts)

	assert.Equal(t, []*ansibleinventory.Host{{
		Alias:      "172.16.2.5",
		Host:       "172.16.2.5",
		User:       "vagrant",
		Password:   "password",
		Connection: "winrm",
	}, {
		Alias:      "172.16.2.6",
		Host:       "172.16.2.6",
		User:       "vagrant",
		Password:   "password",
		Connection: "winrm",
	}}, hosts)
}

func TestHostSSHPrivateKey(t *testing.T) {
	input, err := ioutil.ReadFile("./testdata/ssh_private_key.json")
	assert.Nil(t, err)
	assert.True(t, ansibleinventory.IsInventory(input))

	inventory := ansibleinventory.Inventory{}
	err = inventory.Decode(input)
	assert.Nil(t, err)

	hosts := inventory.List()
	assert.Equal(t, 1, len(hosts))

	assert.Equal(t, []*ansibleinventory.Host{{
		Alias:      "instance1",
		Host:       "192.168.178.11",
		User:       "custom-user",
		Identity:   "/home/custom-user/.ssh/id_rsa",
		Connection: "ssh",
	}}, hosts)
}

func TestInventoryConversion(t *testing.T) {
	input, err := ioutil.ReadFile("./testdata/inventory.json")
	assert.Nil(t, err)
	assert.True(t, ansibleinventory.IsInventory(input))

	ansibleInventory := ansibleinventory.Inventory{}
	err = ansibleInventory.Decode(input)
	assert.Nil(t, err)

	v1Intentory := ansibleInventory.ToV1Inventory()

	assert.Equal(t, 8, len(v1Intentory.Spec.Assets))
}

func TestInventoryWithUsernameConversion(t *testing.T) {
	input, err := ioutil.ReadFile("./testdata/hosts.json")
	assert.Nil(t, err)
	assert.True(t, ansibleinventory.IsInventory(input))

	ansibleInventory := ansibleinventory.Inventory{}
	err = ansibleInventory.Decode(input)
	assert.Nil(t, err)

	v1Intentory := ansibleInventory.ToV1Inventory()
	assert.Equal(t, 2, len(v1Intentory.Spec.Assets))

	a := findAsset(v1Intentory.Spec.Assets, "instance1")
	assert.NotNil(t, a)
	assert.Equal(t, "104.154.55.51", a.Connections[0].Host)
	secretId := a.Connections[0].Credentials[0].SecretId
	cred := v1Intentory.Spec.Credentials[secretId]
	assert.Equal(t, "chris", cred.User)
	assert.Equal(t, vault.CredentialType_ssh_agent, cred.Type)

	a = findAsset(v1Intentory.Spec.Assets, "34.133.130.53")
	assert.NotNil(t, a)
	assert.Equal(t, "34.133.130.53", a.Connections[0].Host)
	secretId = a.Connections[0].Credentials[0].SecretId
	cred = v1Intentory.Spec.Credentials[secretId]
	assert.Equal(t, "chris", cred.User)
	assert.Equal(t, vault.CredentialType_ssh_agent, cred.Type)
}

func TestTagsAndGroups(t *testing.T) {
	input, err := ioutil.ReadFile("./testdata/tags_groups.json")
	assert.Nil(t, err)
	assert.True(t, ansibleinventory.IsInventory(input))

	ansibleInventory := ansibleinventory.Inventory{}
	err = ansibleInventory.Decode(input)
	assert.Nil(t, err)

	hosts := ansibleInventory.List()
	assert.Equal(t, 1, len(hosts))

	assert.Equal(t, []*ansibleinventory.Host{{
		Alias:      "instance1",
		Host:       "192.168.178.11",
		User:       "custom-user",
		Identity:   "/home/custom-user/.ssh/id_rsa",
		Connection: "ssh",
		Labels:     []string{"ansible_host", "mondoo_agent"},
	}}, hosts)

	// convert to mondoo inventory
	v1Intentory := ansibleInventory.ToV1Inventory()
	assert.Equal(t, 1, len(v1Intentory.Spec.Assets))

	a := findAsset(v1Intentory.Spec.Assets, "instance1")
	assert.NotNil(t, a)
	assert.Equal(t, "192.168.178.11", a.Connections[0].Host)
	secretId := a.Connections[0].Credentials[0].SecretId
	cred := v1Intentory.Spec.Credentials[secretId]
	assert.Equal(t, "custom-user", cred.User)
	assert.Equal(t, vault.CredentialType_private_key, cred.Type)
	assert.Equal(t, "/home/custom-user/.ssh/id_rsa", cred.PrivateKeyPath)
}

func findAsset(assetList []*asset.Asset, name string) *asset.Asset {
	for i := range assetList {
		if assetList[i].Name == name {
			return assetList[i]
		}
	}
	return nil
}
