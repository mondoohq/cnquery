package googleworkspace

import (
	"time"

	"go.mondoo.com/cnquery/resources"
	directory "google.golang.org/api/admin/directory/v1"
)

func (g *mqlGoogleworkspace) GetDomains() ([]interface{}, error) {
	provider, directoryService, err := directoryService(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	domains, err := directoryService.Domains.List(provider.GetCustomerID()).Do()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range domains.Domains {
		r, err := newMqlGoogleWorkspaceDomain(g.MotorRuntime, domains.Domains[i])
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func newMqlGoogleWorkspaceDomain(runtime *resources.Runtime, entry *directory.Domains) (interface{}, error) {
	unixTimeUTC := time.Unix(entry.CreationTime, 0)
	return runtime.CreateResource("googleworkspace.domain",
		"domainName", entry.DomainName,
		"isPrimary", entry.IsPrimary,
		"verified", entry.Verified,
		"creationTime", &unixTimeUTC,
	)
}

func (g *mqlGoogleworkspaceDomain) id() (string, error) {
	id, err := g.DomainName()
	if err != nil {
		return "", err
	}
	return "googleworkspace.domain/" + id, nil
}
