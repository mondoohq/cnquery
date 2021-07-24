package ansibleinventory_test

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/inventory/ansibleinventory"
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
	input, err := ioutil.ReadFile("./testdata/local.yml")
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
