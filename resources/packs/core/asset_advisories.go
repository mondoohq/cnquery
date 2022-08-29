package core

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/resources/packs/core/vadvisor"
	"go.mondoo.com/ranger-rpc"
)

// fetches the vulnerability report and returns the full report
func (a *mqlAsset) GetVulnerabilityReport() (interface{}, error) {
	r := a.MotorRuntime
	mcc := r.UpstreamConfig
	if mcc == nil {
		return nil, errors.New("mondoo upstream configuration is missing")
	}

	// get asset information
	obj, err := r.CreateResource("asset")
	if err != nil {
		return nil, err
	}

	mqlAsset := obj.(Asset)
	platformObj := convertMqlAsset2ApiPlatform(mqlAsset)

	// check if the data is cached
	// NOTE: we cache it in the asset resource, so that asset.advisories, asset.cves and
	// asset.exploits can all share the results
	cachedReport, ok := mqlAsset.MqlResource().Cache.Load("_report")
	if ok {
		report := cachedReport.Data.(*vadvisor.VulnReport)
		return report, nil
	}

	// get new advisory report
	// start scanner client
	scannerClient, err := newAdvisoryScannerHttpClient(mcc.ApiEndpoint, mcc.Plugins, ranger.DefaultHttpClient())
	if err != nil {
		return nil, err
	}

	apiPackages := []*vadvisor.Package{}
	kernelVersion := ""

	// collect pacakges if the asset supports gathering files
	if r.Motor.Provider.Capabilities().HasCapability(providers.Capability_File) {
		obj, err = r.CreateResource("packages")
		if err != nil {
			return nil, err
		}
		packages := obj.(Packages)

		lumiPkgs, err := packages.List()
		if err != nil {
			return nil, err
		}

		for i := range lumiPkgs {
			lumiPkg := lumiPkgs[i]
			pkg := lumiPkg.(Package)
			name, _ := pkg.Name()
			version, _ := pkg.Version()
			arch, _ := pkg.Arch()
			format, _ := pkg.Format()
			origin, _ := pkg.Origin()

			apiPackages = append(apiPackages, &vadvisor.Package{
				Name:    name,
				Version: version,
				Arch:    arch,
				Format:  format,
				Origin:  origin,
			})
		}

		// determine the kernel version if possible (just needed for linux at this point)
		// therefore we ignore the error because its not important, worst case the user sees to many advisories
		objKernel, err := r.CreateResource("kernel")
		if err == nil {
			kernel := objKernel.(Kernel)
			kernelInfoRaw, err := kernel.Info()
			if err == nil {
				kernelInfo, ok := kernelInfoRaw.(map[string]interface{})
				if ok {
					val, ok := kernelInfo["version"]
					if ok {
						kernelVersion = val.(string)
					}
				}
			}
		}
	}

	scanjob := &vadvisor.AnalyseAssetRequest{
		Platform:      convertAssetPlatform2VulnPlatform(platformObj),
		Packages:      apiPackages,
		KernelVersion: kernelVersion,
	}

	logger.DebugDumpYAML("vuln-scan-job", scanjob)

	log.Debug().Bool("incognito", mcc.Incognito).Msg("run advisory scan")
	report, err := scannerClient.AnalyseAsset(context.Background(), scanjob)
	if err != nil {
		return nil, err
	}

	return JsonToDict(report)
}
