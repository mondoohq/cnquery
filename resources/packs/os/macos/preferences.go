package macos

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"go.mondoo.com/cnquery/motor/providers/os"

	"howett.net/plist"
)

const (
	currentHostDomains           = "defaults -currentHost domains"
	currentHostDomainPreferences = "defaults -currentHost export %s -"
	userDomains                  = "defaults domains"
	userDomainPreferences        = "defaults export %s -"
)

func NewPreferences(p os.OperatingSystemProvider) *Preferences {
	return &Preferences{
		provider: p,
	}
}

type Preferences struct {
	provider os.OperatingSystemProvider
}

func (p *Preferences) UserPreferences() (map[string]map[string]interface{}, error) {
	return p.preferences(userDomains, userDomainPreferences)
}

func (p *Preferences) UserHostPreferences() (map[string]map[string]interface{}, error) {
	return p.preferences(currentHostDomains, currentHostDomainPreferences)
}

func (p *Preferences) preferences(domainCmd string, preferencesCmd string) (map[string]map[string]interface{}, error) {
	c, err := p.provider.RunCommand(domainCmd)
	if err != nil {
		return nil, err
	}

	domains, err := ParseDomains(c.Stdout)
	if err != nil {
		return nil, err
	}

	res := map[string]map[string]interface{}{}

	for i := range domains {
		domain := domains[i]

		c, err := p.provider.RunCommand(fmt.Sprintf(preferencesCmd, domain))
		if err != nil {
			return nil, err
		}

		data, err := ParsePreferences(c.Stdout)
		if err != nil {
			return nil, err
		}

		res[domain] = data
	}

	return res, nil
}

func ParseDomains(r io.Reader) ([]string, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	res := strings.Split(string(data), ",")

	for i := range res {
		res[i] = strings.TrimSpace(res[i])
	}
	return res, nil
}

func ParsePreferences(input io.Reader) (map[string]interface{}, error) {
	var r io.ReadSeeker
	r, ok := input.(io.ReadSeeker)
	if !ok {
		data, err := ioutil.ReadAll(input)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(data)
	}

	var data map[string]interface{}
	decoder := plist.NewDecoder(r)
	err := decoder.Decode(&data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
