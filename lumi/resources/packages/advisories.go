package packages

import (
	"context"
	"net/http"
	"os"

	"go.mondoo.io/mondoo/lumi/resources/parser"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/vadvisor/api"
	"go.mondoo.io/mondoo/vadvisor/cvss"
)

var MONDOO_API = "https://api.mondoo.app"

// allow overwrite of the API url by an environment variable
func init() {
	if len(os.Getenv("MONDOO_API")) > 0 {
		MONDOO_API = os.Getenv("MONDOO_API")
	}
}

func GetAdvisory(id string) (*api.Advisory, error) {
	sa, err := api.NewSecurityAdvisorClient(MONDOO_API, &http.Client{})
	if err != nil {
		return nil, err
	}
	advisory, err := sa.GetAdvisory(context.TODO(), &api.AdvisoryIdentifier{Id: id})
	if err != nil {
		return nil, err
	}
	return advisory, nil
}

func ConvertPlatform(platform platform.Info) *api.Platform {
	return &api.Platform{
		Name:    platform.Name,
		Release: platform.Release,
		Arch:    platform.Arch,
	}
}

func ConvertParserPackages(pkgs []parser.Package) []*api.Package {
	apiPkgs := []*api.Package{}

	for _, d := range pkgs {
		apiPkgs = append(apiPkgs, &api.Package{
			Name:    d.Name,
			Version: d.Version,
			Arch:    d.Arch,
			Origin:  d.Origin,
		})
	}

	return apiPkgs
}

// searches all advisories for given packages
func Analyze(platform *api.Platform, pkgs []*api.Package) ([]*api.Advisory, error) {
	request := api.Packages{}
	request.Platform = platform
	request.Packages = pkgs

	sa, err := api.NewSecurityAdvisorClient(MONDOO_API, &http.Client{})
	if err != nil {
		return nil, err
	}
	report, err := sa.AnalysePackages(context.TODO(), &request)
	if err != nil {
		return nil, err
	}
	return report.Advisories, nil
}

func MaxCvss(advisories []*api.Advisory) (*api.CVSS, error) {
	list := []*cvss.Cvss{}
	for i := range advisories {
		advisory := advisories[i]
		maxScore := advisory.MaxScore

		if maxScore != nil && len(maxScore.Vector) > 0 {
			res, err := cvss.New(maxScore.Vector)
			if err != nil {
				return nil, err
			}
			list = append(list, res)
		}
	}

	max, err := cvss.MaxScore(list)
	if err != nil {
		return nil, err
	}

	return &api.CVSS{
		Vector: max.Vector,
	}, nil
}
