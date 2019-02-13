package packages

import (
	"context"
	"net/http"
	"os"

	"go.mondoo.io/mondoo/lumi/resources/parser"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/vadvisor/api"
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
	return sa.GetAdvisory(context.TODO(), &api.AdvisoryIdentifier{Id: id})
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
func Analyze(scanJob *api.ScanJob) (*api.Report, error) {
	sa, err := api.NewSecurityAdvisorClient(MONDOO_API, &http.Client{})
	if err != nil {
		return nil, err
	}

	return sa.Analyse(context.Background(), scanJob)
}
