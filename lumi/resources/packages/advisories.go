package packages

import (
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
