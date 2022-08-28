package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/motor/providers/mock"
	"go.mondoo.com/cnquery/resources/packs/os/services"
)

func TestParseBsdInit(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/freebsd12.toml")
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("service -e")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)
	m, err := services.ParseBsdInit(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 25, len(m), "detected the right amount of services")

	assert.Equal(t, "/etc/rc.d/hostid", m[0].Name, "service name detected")
	assert.Equal(t, true, m[0].Running, "service is running")
	assert.Equal(t, true, m[0].Installed, "service is installed")
	assert.Equal(t, "bsd", m[0].Type, "service type is added")
}
