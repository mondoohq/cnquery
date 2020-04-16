package services_test

import (
	"os"
	"testing"

	"github.com/tj/assert"
	"go.mondoo.io/mondoo/lumi/resources/services"
)

func TestWindowsServiceParser(t *testing.T) {
	data, err := os.Open("./testdata/windows2019.json")
	if err != nil {
		t.Fatal(err)
	}

	srvs, err := services.ParseWindowsService(data)
	assert.Nil(t, err)
	assert.Equal(t, 7, len(srvs))

	expected := &services.Service{
		Name:        "PolicyAgent",
		Description: "IPsec Policy Agent",
		State:       "ServiceStopped",
		Running:     false,
		Installed:   true,
		Enabled:     true,
		Type:        "windows",
	}
	found := findService(srvs, "PolicyAgent")
	assert.EqualValues(t, expected, found)

	expected = &services.Service{
		Name:        "PlugPlay",
		Description: "Plug and Play",
		State:       "ServiceRunning",
		Running:     true,
		Installed:   true,
		Enabled:     true,
		Type:        "windows",
	}
	found = findService(srvs, "PlugPlay")
	assert.EqualValues(t, expected, found)

	expected = &services.Service{
		Name:        "PhoneSvc",
		Description: "Phone Service",
		State:       "ServiceStopped",
		Running:     false,
		Installed:   true,
		Enabled:     false,
		Type:        "windows",
	}
	found = findService(srvs, "PhoneSvc")
	assert.EqualValues(t, expected, found)
}

func findService(srvs []*services.Service, name string) *services.Service {
	for i := range srvs {
		if srvs[i].Name == name {
			return srvs[i]
		}
	}
	return nil
}
