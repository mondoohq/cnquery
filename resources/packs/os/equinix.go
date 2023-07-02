package os

import (
	"time"

	"errors"
	"github.com/packethost/packngo"
	"go.mondoo.com/cnquery/motor/providers"
	equinix_provider "go.mondoo.com/cnquery/motor/providers/equinix"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func equinixProvider(t providers.Instance) (*equinix_provider.Provider, error) {
	provider, ok := t.(*equinix_provider.Provider)
	if !ok {
		return nil, errors.New("equinix resource is not supported on this provider")
	}
	return provider, nil
}

// "2021-03-03T11:13:46Z"
func parseEquinixTime(timestamp string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05Z", timestamp)
}

func (p *mqlEquinixMetalProject) id() (string, error) {
	return p.Url()
}

func (g *mqlEquinixMetalProject) init(args *resources.Args) (*resources.Args, EquinixMetalProject, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	// fetch the default project from the provider
	et, err := equinixProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	p := et.Project()
	if p == nil {
		return nil, nil, errors.New("could not retrieve project information from provider")
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

func (p *mqlEquinixMetalProject) GetOrganization() (interface{}, error) {
	provider, err := equinixProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	c := provider.Client()

	// NOTE: if we are going to support multiple projects, we need to change this logic
	project := provider.Project()

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

func (p *mqlEquinixMetalProject) GetUsers() ([]interface{}, error) {
	provider, err := equinixProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	c := provider.Client()

	// NOTE: if we are going to support multiple projects, we need to change this logic
	project := provider.Project()

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

		var twitter, facebook, linkedin string
		if fetchedUserData.SocialAccounts != nil {
			twitter = fetchedUserData.SocialAccounts.Twitter
			linkedin = fetchedUserData.SocialAccounts.LinkedIn
			// TODO: let's update the used fields here, I'm not sure which ones are needed (dom)
		}

		mqlEquinixSshKey, err := p.MotorRuntime.CreateResource("equinix.metal.user",
			"url", fetchedUserData.URL,
			"id", fetchedUserData.ID,
			"firstName", fetchedUserData.FirstName,
			"lastName", fetchedUserData.LastName,
			"fullName", fetchedUserData.FullName,
			"email", fetchedUserData.Email,
			"phoneNumber", fetchedUserData.PhoneNumber,
			"twitter", twitter,
			"facebook", facebook,
			"linkedin", linkedin,
			"timezone", fetchedUserData.TimeZone,
			"twoFactorAuth", fetchedUserData.TwoFactorAuth,
			"avatarUrl", fetchedUserData.AvatarURL,
			"createdAt", &created,
			"updatedAt", &updated,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlEquinixSshKey)
	}

	return res, nil
}

func (p *mqlEquinixMetalProject) GetSshKeys() ([]interface{}, error) {
	provider, err := equinixProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	c := provider.Client()

	// NOTE: if we are going to support multiple projects, we need to change this logic
	project := provider.Project()

	keys, _, err := c.SSHKeys.ProjectList(project.ID)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range keys {
		key := keys[i]

		created, _ := parseEquinixTime(key.Created)
		updated, _ := parseEquinixTime(key.Updated)

		mqlEquinixSshKey, err := p.MotorRuntime.CreateResource("equinix.metal.sshkey",
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
		res = append(res, mqlEquinixSshKey)
	}

	return res, nil
}

func (p *mqlEquinixMetalProject) GetDevices() ([]interface{}, error) {
	provider, err := equinixProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	c := provider.Client()

	// NOTE: if we are going to support multiple projects, we need to change this logic
	project := provider.Project()

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

		mqlEquinixDevice, err := p.MotorRuntime.CreateResource("equinix.metal.sshkey",
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
		res = append(res, mqlEquinixDevice)
	}

	return res, nil
}

func (o *mqlEquinixMetalOrganization) id() (string, error) {
	return o.Url()
}

func (u *mqlEquinixMetalUser) id() (string, error) {
	return u.Url()
}

func (s *mqlEquinixMetalSshkey) id() (string, error) {
	return s.Url()
}

func (d *mqlEquinixMetalDevice) id() (string, error) {
	return d.Url()
}
