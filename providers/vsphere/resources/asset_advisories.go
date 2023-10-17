package resources

import (
	"context"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/logger"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v9/providers/vsphere/connection"
	"go.mondoo.com/ranger-rpc"
	"net/http"
)

func newAdvisoryScannerHttpClient(mondooapi string, plugins []ranger.ClientPlugin, httpClient *http.Client) (*mvd.AdvisoryScannerClient, error) {
	sa, err := mvd.NewAdvisoryScannerClient(mondooapi, httpClient)
	if err != nil {
		return nil, err
	}

	for i := range plugins {
		sa.AddPlugin(plugins[i])
	}
	return sa, nil
}

func fetchVulnReport(runtime *plugin.Runtime) (interface{}, error) {
	mcc := runtime.Upstream
	if mcc == nil || mcc.ApiEndpoint == "" {
		return nil, resources.MissingUpstreamError{}
	}

	// get new advisory report
	// start scanner client
	scannerClient, err := newAdvisoryScannerHttpClient(mcc.ApiEndpoint, mcc.Plugins, mcc.HttpClient)
	if err != nil {
		return nil, err
	}

	conn := runtime.Connection.(*connection.VsphereConnection)
	apiPackages := []*mvd.Package{}
	kernelVersion := ""

	scanjob := &mvd.AnalyseAssetRequest{
		Platform:      convertPlatform2VulnPlatform(conn.Asset().Platform),
		Packages:      apiPackages,
		KernelVersion: kernelVersion,
	}
	logger.DebugDumpYAML("vuln-scan-job", scanjob)

	log.Debug().Bool("incognito", mcc.Incognito).Msg("run advisory scan")
	report, err := scannerClient.AnalyseAsset(context.Background(), scanjob)
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(report)
}

// fetches the vulnerability report and returns the full report
func (p *mqlAsset) vulnerabilityReport() (interface{}, error) {
	return fetchVulnReport(p.MqlRuntime)
}
