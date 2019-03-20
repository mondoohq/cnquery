package packages

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/nexus/agents"

	"go.mondoo.io/mondoo/falcon"
	guard_cert_auth "go.mondoo.io/mondoo/falcon/guard/authentication/cert"
	"go.mondoo.io/mondoo/lumi/resources/parser"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/vadvisor/api"
)

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

type Scanner struct {
	MondooApiUrl string
	Agentid      string
	Spaceid      string
	Privatekey   *string
}

func (s Scanner) authentication() []falcon.ClientPlugin {
	plugins := []falcon.ClientPlugin{}
	if s.Privatekey != nil {
		var certplugin falcon.ClientPlugin
		config, err := agents.GetGuardClientConfig(s.Spaceid, s.Agentid, *s.Privatekey)
		if err == nil {
			certplugin, err = guard_cert_auth.NewClientPlugin(config)
		}

		if err == nil {
			plugins = append(plugins, certplugin)
		} else {
			log.Error().Err(err).Msg("Cannot configure certificate authentication")
		}
	}

	return plugins
}

// searches all advisories for given packages
func (s *Scanner) Analyze(scanJob *api.ScanJob) (*api.Report, error) {
	sa, err := api.NewSecurityAdvisorClient(s.MondooApiUrl, &http.Client{})
	if err != nil {
		return nil, err
	}

	auth := s.authentication()
	for i := range auth {
		sa.AddPlugin(auth[i])
	}

	return sa.Analyse(context.Background(), scanJob)
}

func (s *Scanner) GetCve(id string) (*api.CVE, error) {
	sa, err := api.NewSecurityAdvisorClient(s.MondooApiUrl, &http.Client{})
	if err != nil {
		return nil, err
	}
	cve, err := sa.GetCVE(context.TODO(), &api.CveIdentifier{Id: id})
	if err != nil {
		return nil, err
	}
	return cve, nil
}

func (s *Scanner) GetAdvisory(id string) (*api.Advisory, error) {
	sa, err := api.NewSecurityAdvisorClient(s.MondooApiUrl, &http.Client{})
	if err != nil {
		return nil, err
	}
	return sa.GetAdvisory(context.TODO(), &api.AdvisoryIdentifier{Id: id})
}
