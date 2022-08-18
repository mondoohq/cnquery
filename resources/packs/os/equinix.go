package os

import (
	"time"

	"github.com/cockroachdb/errors"
	"github.com/packethost/packngo"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/resources/packs/core"

	equinix_transport "go.mondoo.io/mondoo/motor/providers/equinix"
)

func equinixtransport(t providers.Transport) (*equinix_transport.Provider, error) {
	at, ok := t.(*equinix_transport.Provider)
	if !ok {
		return nil, errors.New("equinix resource is not supported on this transport")
	}
	return at, nil
}

// "2021-03-03T11:13:46Z"
func parseEquinixTime(timestamp string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05Z", timestamp)
}

func (p *lumiEquinixMetalProject) id() (string, error) {
	return p.Url()
}

func (g *lumiEquinixMetalProject) init(args *lumi.Args) (*lumi.Args, EquinixMetalProject, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	// fetch the default project from the transport
	et, err := equinixtransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	p := et.Project()
	if p == nil {
		return nil, nil, errors.New("could not retrieve project information from transport")
	}

	pm, _ := core.JsonToDict(p.PaymentMethod)

	created, err := parseEquinixTime(p.Created)
	if err != nil {
		return nil, nil, err
	}
	updated, err := parseEquinixTime(p.Updated)
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = p.ID
	(*args)["name"] = p.Name
	(*args)["url"] = p.URL
	(*args)["paymentMethod"] = pm
	(*args)["createdAt"] = &created
	(*args)["updatedAt"] = &updated
	return args, nil, nil
}

func (p *lumiEquinixMetalProject) GetOrganization() (interface{}, error) {
	et, err := equinixtransport(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	c := et.Client()

	// NOTE: if we are going to support multiple projects, we need to change this logic
	project := et.Project()

	// we need to list the organization to circumvent the get issue
	// if we request the project and try to access the org, it only returns the url
	// its similar to https://github.com/packethost/packngo/issues/245
	var org *packngo.Organization
	orgs, _, err := c.Organizations.List(nil)
	if err != nil {
		return nil, err
	}

	for i := range orgs {
		o := orgs[i]
		if o.URL == project.Organization.URL {
			org = &o
			break
		}
	}

	if org == nil {
		return nil, errors.New("could not retrieve the organization: " + project.Organization.URL)
	}

	created, _ := parseEquinixTime(org.Created)
	updated, _ := parseEquinixTime(org.Updated)
	address, _ := core.JsonToDict(org.Address)

	return p.MotorRuntime.CreateResource("equinix.metal.organization",
		"url", org.URL,
		"id", org.ID,
		"name", org.Name,
		"description", org.Description,
		"website", org.Website,
		"twitter", org.Twitter,
		"address", address,
		"taxId", org.TaxID,
		"mainPhone", org.MainPhone,
		"billingPhone", org.BillingPhone,
		"creditAmount", org.CreditAmount,
		"createdAt", &created,
		"updatedAt", &updated,
	)
}

func (p *lumiEquinixMetalProject) GetUsers() ([]interface{}, error) {
	et, err := equinixtransport(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	c := et.Client()

	// NOTE: if we are going to support multiple projects, we need to change this logic
	project := et.Project()

	// NOTE: circumvent the API, since project user only includes url of the user
	userMap := map[string]packngo.User{}
	users, _, err := c.Users.List(nil)
	if err != nil {
		return nil, err
	}

	for i := range users {
		user := users[i]
		userMap[user.URL] = user
	}

	// now iterate over the user urls of the project
	res := []interface{}{}
	for i := range project.Users {
		usr := project.Users[i]
		fetchedUserData, ok := userMap[usr.URL]
		if !ok {
			return nil, errors.New("could not retrieve information for user: " + usr.URL)
		}

		created, _ := parseEquinixTime(fetchedUserData.Created)
		updated, _ := parseEquinixTime(fetchedUserData.Updated)

		lumiEquinixSshKey, err := p.MotorRuntime.CreateResource("equinix.metal.user",
			"url", fetchedUserData.URL,
			"id", fetchedUserData.ID,
			"firstName", fetchedUserData.FirstName,
			"lastName", fetchedUserData.LastName,
			"fullName", fetchedUserData.FullName,
			"email", fetchedUserData.Email,
			"phoneNumber", fetchedUserData.PhoneNumber,
			"twitter", fetchedUserData.Twitter,
			"facebook", fetchedUserData.Facebook,
			"linkedin", fetchedUserData.LinkedIn,
			"timezone", fetchedUserData.TimeZone,
			"vpn", fetchedUserData.VPN,
			"twoFactorAuth", fetchedUserData.TwoFactor,
			"avatarUrl", fetchedUserData.AvatarURL,
			"createdAt", &created,
			"updatedAt", &updated,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiEquinixSshKey)
	}

	return res, nil
}

func (p *lumiEquinixMetalProject) GetSshKeys() ([]interface{}, error) {
	et, err := equinixtransport(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	c := et.Client()

	// NOTE: if we are going to support multiple projects, we need to change this logic
	project := et.Project()

	keys, _, err := c.SSHKeys.ProjectList(project.ID)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range keys {
		key := keys[i]

		created, _ := parseEquinixTime(key.Created)
		updated, _ := parseEquinixTime(key.Updated)

		lumiEquinixSshKey, err := p.MotorRuntime.CreateResource("equinix.metal.sshkey",
			"url", key.URL,
			"id", key.ID,
			"label", key.Label,
			"key", key.Key,
			"fingerPrint", key.FingerPrint,
			"createdAt", &created,
			"updatedAt", &updated,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiEquinixSshKey)
	}

	return res, nil
}

func (p *lumiEquinixMetalProject) GetDevices() ([]interface{}, error) {
	et, err := equinixtransport(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	c := et.Client()

	// NOTE: if we are going to support multiple projects, we need to change this logic
	project := et.Project()

	devices, _, err := c.Devices.List(project.ID, nil)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range devices {
		device := devices[i]

		created, _ := parseEquinixTime(device.Created)
		updated, _ := parseEquinixTime(device.Updated)
		os, _ := core.JsonToDict(device.OS)

		lumiEquinixDevice, err := p.MotorRuntime.CreateResource("equinix.metal.sshkey",
			"url", device.Href,
			"id", device.ID,
			"shortID", device.ShortID,
			"hostname", device.Hostname,
			"description", device.Description,
			"state", device.State,
			"locked", device.Locked,
			"billingCycle", device.BillingCycle,
			"spotInstance", device.SpotInstance,
			"os", os,
			"createdAt", &created,
			"updatedAt", &updated,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiEquinixDevice)
	}

	return res, nil
}

func (o *lumiEquinixMetalOrganization) id() (string, error) {
	return o.Url()
}

func (u *lumiEquinixMetalUser) id() (string, error) {
	return u.Url()
}

func (s *lumiEquinixMetalSshkey) id() (string, error) {
	return s.Url()
}

func (d *lumiEquinixMetalDevice) id() (string, error) {
	return d.Url()
}
