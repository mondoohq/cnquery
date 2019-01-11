package packages

import (
	"context"
	"net/http"

	"go.mondoo.io/mondoo/lumi/resources/parser"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/vadvisor/api"
	"go.mondoo.io/mondoo/vadvisor/cvss"
)

const (
	ADVISORY_SERVICE = "http://localhost:8989"
)

func GetAdvisory(id string) (*api.Advisory, error) {
	sa, err := api.NewSecuriyAdvisorClient(ADVISORY_SERVICE, &http.Client{})
	if err != nil {
		return nil, err
	}
	advisory, err := sa.GetAdvisory(context.TODO(), &api.AdvisoryIdentifier{Id: id})
	if err != nil {
		return nil, err
	}
	return advisory, nil
}

// searches all advisories for given packages
func Analyze(platform platform.Info, pkgs []parser.Package) ([]api.Advisory, error) {
	request := api.Packages{}
	request.Platform = &api.Platform{
		Name:    platform.Name,
		Release: platform.Release,
		Arch:    platform.Arch,
	}
	for _, d := range pkgs {
		request.Packages = append(request.Packages, &api.Package{
			Name:    d.Name,
			Version: d.Version,
			Arch:    d.Arch,
		})
	}

	sa, err := api.NewSecuriyAdvisorClient(ADVISORY_SERVICE, &http.Client{})
	if err != nil {
		return nil, err
	}
	report, err := sa.Analyze(context.TODO(), &request)
	if err != nil {
		return nil, err
	}
	return convertAdvisoryList(report.Advisories)
}

// iterate over list and ask the vadvisor to download all
func convertAdvisoryList(advisoryIds []*api.AdvisoryIdentifier) ([]api.Advisory, error) {
	var advisories []api.Advisory
	for i := range advisoryIds {
		advisoryID := advisoryIds[i]

		advisory, err := GetAdvisory(advisoryID.Id)
		advisory.Affected = advisoryID.Affected
		if err != nil {
			return nil, err
		}
		advisories = append(advisories, *advisory)
	}
	return advisories, nil
}

func MaxCvss(advisories []api.Advisory) (api.CVSS, error) {
	list := []*cvss.Cvss{}
	for i := range advisories {
		advisory := advisories[i]
		maxScore := advisory.MaxScore

		if maxScore != nil {
			res, err := cvss.New(maxScore.Vector)
			if err != nil {
				return api.CVSS{}, err
			}
			list = append(list, res)
		}
	}

	max, err := cvss.MaxScore(list)
	if err != nil {
		return api.CVSS{}, err
	}

	return api.CVSS{
		Vector: max.Vector,
	}, nil
}
