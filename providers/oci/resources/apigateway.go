// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/oracle/oci-go-sdk/v65/apigateway"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciApigateway) id() (string, error) {
	return "oci.apigateway", nil
}

// Gateways

func (o *mqlOciApigateway) gateways() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getGateways(conn, list.Data), 5)
	poolOfJobs.Run()
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}
	return res, nil
}

func (o *mqlOciApigateway) getGateways(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci api gateways with region %s", regionResource.Id.Data)

			svc, err := conn.ApiGatewayGatewayClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var items []apigateway.GatewaySummary
			var page *string
			for {
				response, err := svc.ListGateways(ctx, apigateway.ListGatewaysRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}
				items = append(items, response.Items...)
				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			res := make([]any, 0, len(items))
			for i := range items {
				g := items[i]

				var created, updated *time.Time
				if g.TimeCreated != nil {
					created = &g.TimeCreated.Time
				}
				if g.TimeUpdated != nil {
					updated = &g.TimeUpdated.Time
				}

				freeformTags := make(map[string]interface{}, len(g.FreeformTags))
				for k, v := range g.FreeformTags {
					freeformTags[k] = v
				}
				definedTags := make(map[string]interface{}, len(g.DefinedTags))
				for k, v := range g.DefinedTags {
					definedTags[k] = v
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.apigateway.gateway", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(g.Id),
					"name":          llx.StringDataPtr(g.DisplayName),
					"compartmentID": llx.StringDataPtr(g.CompartmentId),
					"endpointType":  llx.StringData(string(g.EndpointType)),
					"ipMode":        llx.StringData(string(g.IpMode)),
					"hostname":      llx.StringDataPtr(g.Hostname),
					"state":         llx.StringData(string(g.LifecycleState)),
					"created":       llx.TimeDataPtr(created),
					"timeUpdated":   llx.TimeDataPtr(updated),
					"freeformTags":  llx.MapData(freeformTags, types.String),
					"definedTags":   llx.MapData(definedTags, types.Any),
					"systemTags":    llx.MapData(definedTagsToAny(g.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlGw := mqlInstance.(*mqlOciApigatewayGateway)
				mqlGw.region = regionResource.Id.Data
				mqlGw.cacheSubnetId = stringValue(g.SubnetId)
				mqlGw.cacheCertificateId = stringValue(g.CertificateId)
				mqlGw.cacheNsgIds = append([]string(nil), g.NetworkSecurityGroupIds...)
				res = append(res, mqlGw)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciApigatewayGatewayInternal struct {
	region             string
	cacheSubnetId      string
	cacheCertificateId string
	cacheNsgIds        []string

	// details fetched lazily from GetGateway (ip addresses, CA bundles,
	// response cache) — not in the list summary.
	detailsLock sync.Mutex
	detailsDone bool
}

func (o *mqlOciApigatewayGateway) id() (string, error) {
	return "oci.apigateway.gateway/" + o.Id.Data, nil
}

func (o *mqlOciApigatewayGateway) subnet() (*mqlOciNetworkSubnet, error) {
	if o.cacheSubnetId == "" {
		o.Subnet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.subnet", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheSubnetId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkSubnet), nil
}

func (o *mqlOciApigatewayGateway) networkSecurityGroups() ([]any, error) {
	res := make([]any, 0, len(o.cacheNsgIds))
	for _, nsgId := range o.cacheNsgIds {
		if nsgId == "" {
			continue
		}
		nsg, err := NewResource(o.MqlRuntime, "oci.network.networkSecurityGroup", map[string]*llx.RawData{
			"id": llx.StringData(nsgId),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, nsg)
	}
	return res, nil
}

func (o *mqlOciApigatewayGateway) certificate() (*mqlOciApigatewayCertificate, error) {
	if o.cacheCertificateId == "" {
		o.Certificate.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.apigateway.certificate", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheCertificateId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciApigatewayCertificate), nil
}

// fetchDetails calls GetGateway to populate fields that the list summary
// does not return: ipAddresses, caBundles, responseCacheDetails. Safe for
// concurrent callers.
func (o *mqlOciApigatewayGateway) fetchDetails() error {
	if o.detailsDone {
		return nil
	}
	o.detailsLock.Lock()
	defer o.detailsLock.Unlock()
	if o.detailsDone {
		return nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	svc, err := conn.ApiGatewayGatewayClient(o.region)
	if err != nil {
		return err
	}
	resp, err := svc.GetGateway(context.Background(), apigateway.GetGatewayRequest{
		GatewayId: common.String(o.Id.Data),
	})
	if err != nil {
		return err
	}

	ips := make([]any, 0, len(resp.IpAddresses))
	for i := range resp.IpAddresses {
		ips = append(ips, stringValue(resp.IpAddresses[i].IpAddress))
	}
	o.IpAddresses = plugin.TValue[[]any]{Data: ips, State: plugin.StateIsSet}

	caBundles := make([]any, 0, len(resp.CaBundles))
	for i := range resp.CaBundles {
		switch cb := resp.CaBundles[i].(type) {
		case apigateway.CertificatesCaBundle:
			caBundles = append(caBundles, stringValue(cb.CaBundleId))
		case apigateway.CertificatesCertificateAuthority:
			caBundles = append(caBundles, stringValue(cb.CertificateAuthorityId))
		}
	}
	o.CaBundleIds = plugin.TValue[[]any]{Data: caBundles, State: plugin.StateIsSet}

	// response cache presence only — the concrete config may hold a
	// Redis/external endpoint we don't want to expose carte-blanche.
	hasCache := false
	if resp.ResponseCacheDetails != nil {
		// NoCache is modeled as a concrete type; anything else indicates
		// a cache is configured.
		if _, isNone := resp.ResponseCacheDetails.(apigateway.NoCache); !isNone {
			hasCache = true
		}
	}
	o.HasResponseCache = plugin.TValue[bool]{Data: hasCache, State: plugin.StateIsSet}

	o.detailsDone = true
	return nil
}

func (o *mqlOciApigatewayGateway) ipAddresses() ([]any, error) {
	if err := o.fetchDetails(); err != nil {
		return nil, err
	}
	return o.IpAddresses.Data, nil
}

func (o *mqlOciApigatewayGateway) caBundleIds() ([]any, error) {
	if err := o.fetchDetails(); err != nil {
		return nil, err
	}
	return o.CaBundleIds.Data, nil
}

func (o *mqlOciApigatewayGateway) hasResponseCache() (bool, error) {
	if err := o.fetchDetails(); err != nil {
		return false, err
	}
	return o.HasResponseCache.Data, nil
}

// Deployments

func (o *mqlOciApigateway) deployments() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getDeployments(conn, list.Data), 5)
	poolOfJobs.Run()
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}
	return res, nil
}

func (o *mqlOciApigateway) getDeployments(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci api gateway deployments with region %s", regionResource.Id.Data)

			svc, err := conn.ApiGatewayDeploymentClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var items []apigateway.DeploymentSummary
			var page *string
			for {
				response, err := svc.ListDeployments(ctx, apigateway.ListDeploymentsRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}
				items = append(items, response.Items...)
				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			res := make([]any, 0, len(items))
			for i := range items {
				d := items[i]

				var created, updated *time.Time
				if d.TimeCreated != nil {
					created = &d.TimeCreated.Time
				}
				if d.TimeUpdated != nil {
					updated = &d.TimeUpdated.Time
				}

				freeformTags := make(map[string]interface{}, len(d.FreeformTags))
				for k, v := range d.FreeformTags {
					freeformTags[k] = v
				}
				definedTags := make(map[string]interface{}, len(d.DefinedTags))
				for k, v := range d.DefinedTags {
					definedTags[k] = v
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.apigateway.deployment", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(d.Id),
					"name":          llx.StringDataPtr(d.DisplayName),
					"compartmentID": llx.StringDataPtr(d.CompartmentId),
					"pathPrefix":    llx.StringDataPtr(d.PathPrefix),
					"endpoint":      llx.StringDataPtr(d.Endpoint),
					"state":         llx.StringData(string(d.LifecycleState)),
					"created":       llx.TimeDataPtr(created),
					"timeUpdated":   llx.TimeDataPtr(updated),
					"freeformTags":  llx.MapData(freeformTags, types.String),
					"definedTags":   llx.MapData(definedTags, types.Any),
					"systemTags":    llx.MapData(definedTagsToAny(d.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlDep := mqlInstance.(*mqlOciApigatewayDeployment)
				mqlDep.region = regionResource.Id.Data
				mqlDep.cacheGatewayId = stringValue(d.GatewayId)
				res = append(res, mqlDep)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciApigatewayDeploymentInternal struct {
	region         string
	cacheGatewayId string

	// specFetched + spec populate the request-policy-derived fields.
	specLock    sync.Mutex
	specFetched bool
	spec        *apigateway.ApiSpecification
}

func (o *mqlOciApigatewayDeployment) id() (string, error) {
	return "oci.apigateway.deployment/" + o.Id.Data, nil
}

// initOciApigatewayDeployment resolves a single deployment from the scan
// asset's PlatformId when policies reference `oci.apigateway.deployment` on a
// discovered oci-apigateway-deployment asset. Explicit id takes precedence.
func initOciApigatewayDeployment(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	idVal := ociArgString(args, "id")
	if idVal == "" {
		conn := runtime.Connection.(*connection.OciConnection)
		if conn.Conf == nil || conn.Conf.PlatformId == "" {
			return args, nil, nil
		}
		parsed, ok := parseOciObjectPlatformID(conn.Conf.PlatformId)
		if !ok || parsed.service != "apigateway" || parsed.objectType != "deployment" {
			return args, nil, nil
		}
		idVal = parsed.id
	}

	obj, err := CreateResource(runtime, "oci.apigateway", nil)
	if err != nil {
		return nil, nil, err
	}
	apigw := obj.(*mqlOciApigateway)

	deps := apigw.GetDeployments()
	if deps.Error != nil {
		return nil, nil, deps.Error
	}

	for _, raw := range deps.Data {
		d := raw.(*mqlOciApigatewayDeployment)
		if d.Id.Data == idVal {
			return args, d, nil
		}
	}

	return nil, nil, errors.New("oci.apigateway.deployment not found: " + idVal)
}

func (o *mqlOciApigatewayDeployment) gateway() (*mqlOciApigatewayGateway, error) {
	if o.cacheGatewayId == "" {
		o.Gateway.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.apigateway.gateway", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheGatewayId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciApigatewayGateway), nil
}

// fetchSpec calls GetDeployment once to retrieve the full ApiSpecification
// (routes, request policies, mTLS, CORS, rate limits). Subsequent callers
// use the cached spec. The request is intentionally serialized behind a
// mutex to avoid N concurrent GetDeployment calls on first access.
func (o *mqlOciApigatewayDeployment) fetchSpec() error {
	if o.specFetched {
		return nil
	}
	o.specLock.Lock()
	defer o.specLock.Unlock()
	if o.specFetched {
		return nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	svc, err := conn.ApiGatewayDeploymentClient(o.region)
	if err != nil {
		return err
	}
	resp, err := svc.GetDeployment(context.Background(), apigateway.GetDeploymentRequest{
		DeploymentId: common.String(o.Id.Data),
	})
	if err != nil {
		return err
	}
	o.spec = resp.Specification
	o.specFetched = true
	return nil
}

func (o *mqlOciApigatewayDeployment) requestPolicies() (*apigateway.ApiSpecificationRequestPolicies, error) {
	if err := o.fetchSpec(); err != nil {
		return nil, err
	}
	if o.spec == nil {
		return nil, nil
	}
	return o.spec.RequestPolicies, nil
}

// flattenAuthenticationType maps the SDK's authentication-policy union type
// to the string enum exposed on the MQL deployment resource. Exported as a
// package-level helper so it can be unit tested without a runtime.
func flattenAuthenticationType(rp *apigateway.ApiSpecificationRequestPolicies) string {
	if rp == nil || rp.Authentication == nil {
		return "NONE"
	}
	switch rp.Authentication.(type) {
	case apigateway.JwtAuthenticationPolicy:
		return "JWT_AUTHENTICATION"
	case apigateway.TokenAuthenticationPolicy:
		return "TOKEN_AUTHENTICATION"
	case apigateway.CustomAuthenticationPolicy:
		return "CUSTOM_AUTHENTICATION"
	}
	return "UNKNOWN"
}

// flattenIsAnonymousAccessAllowed returns the authentication policy's
// anonymous-access flag, defaulting to false when absent.
func flattenIsAnonymousAccessAllowed(rp *apigateway.ApiSpecificationRequestPolicies) bool {
	if rp == nil || rp.Authentication == nil {
		return false
	}
	if v := rp.Authentication.GetIsAnonymousAccessAllowed(); v != nil {
		return *v
	}
	return false
}

// flattenJwtAudiences returns the JWT audiences for a deployment whose
// authentication policy is JWT, or an empty slice otherwise.
func flattenJwtAudiences(rp *apigateway.ApiSpecificationRequestPolicies) []any {
	if rp == nil || rp.Authentication == nil {
		return []any{}
	}
	jwt, ok := rp.Authentication.(apigateway.JwtAuthenticationPolicy)
	if !ok {
		return []any{}
	}
	return convert.SliceAnyToInterface(jwt.Audiences)
}

// flattenJwtIssuers returns the JWT issuers for a deployment whose
// authentication policy is JWT, or an empty slice otherwise.
func flattenJwtIssuers(rp *apigateway.ApiSpecificationRequestPolicies) []any {
	if rp == nil || rp.Authentication == nil {
		return []any{}
	}
	jwt, ok := rp.Authentication.(apigateway.JwtAuthenticationPolicy)
	if !ok {
		return []any{}
	}
	return convert.SliceAnyToInterface(jwt.Issuers)
}

// flattenMutualTls returns the mTLS isVerifiedCertificateRequired flag and
// the allowed-SAN list for the request policies. nil-safe.
func flattenMutualTls(rp *apigateway.ApiSpecificationRequestPolicies) (bool, []any) {
	if rp == nil || rp.MutualTls == nil {
		return false, []any{}
	}
	isRequired := false
	if rp.MutualTls.IsVerifiedCertificateRequired != nil {
		isRequired = *rp.MutualTls.IsVerifiedCertificateRequired
	}
	return isRequired, convert.SliceAnyToInterface(rp.MutualTls.AllowedSans)
}

// flattenCors returns the CORS allow-credentials flag and allowed-origin
// list for the request policies. nil-safe.
func flattenCors(rp *apigateway.ApiSpecificationRequestPolicies) (bool, []any) {
	if rp == nil || rp.Cors == nil {
		return false, []any{}
	}
	allowCreds := false
	if rp.Cors.IsAllowCredentialsEnabled != nil {
		allowCreds = *rp.Cors.IsAllowCredentialsEnabled
	}
	return allowCreds, convert.SliceAnyToInterface(rp.Cors.AllowedOrigins)
}

// flattenRateLimiting returns the configured requests-per-second and rate
// key. nil-safe; missing config yields (0, "").
func flattenRateLimiting(rp *apigateway.ApiSpecificationRequestPolicies) (int64, string) {
	if rp == nil || rp.RateLimiting == nil {
		return 0, ""
	}
	var rate int64
	if rp.RateLimiting.RateInRequestsPerSecond != nil {
		rate = int64(*rp.RateLimiting.RateInRequestsPerSecond)
	}
	return rate, string(rp.RateLimiting.RateKey)
}

// hasDynamicAuthentication reports whether request-level dynamic
// authentication is configured on the deployment.
func hasDynamicAuthentication(rp *apigateway.ApiSpecificationRequestPolicies) bool {
	if rp == nil {
		return false
	}
	return rp.DynamicAuthentication != nil
}

func (o *mqlOciApigatewayDeployment) authenticationType() (string, error) {
	rp, err := o.requestPolicies()
	if err != nil {
		return "", err
	}
	return flattenAuthenticationType(rp), nil
}

func (o *mqlOciApigatewayDeployment) isAnonymousAccessAllowed() (bool, error) {
	rp, err := o.requestPolicies()
	if err != nil {
		return false, err
	}
	return flattenIsAnonymousAccessAllowed(rp), nil
}

func (o *mqlOciApigatewayDeployment) jwtAudiences() ([]any, error) {
	rp, err := o.requestPolicies()
	if err != nil {
		return nil, err
	}
	return flattenJwtAudiences(rp), nil
}

func (o *mqlOciApigatewayDeployment) jwtIssuers() ([]any, error) {
	rp, err := o.requestPolicies()
	if err != nil {
		return nil, err
	}
	return flattenJwtIssuers(rp), nil
}

func (o *mqlOciApigatewayDeployment) mtlsIsVerifiedCertificateRequired() (bool, error) {
	rp, err := o.requestPolicies()
	if err != nil {
		return false, err
	}
	required, _ := flattenMutualTls(rp)
	return required, nil
}

func (o *mqlOciApigatewayDeployment) mtlsAllowedSans() ([]any, error) {
	rp, err := o.requestPolicies()
	if err != nil {
		return nil, err
	}
	_, sans := flattenMutualTls(rp)
	return sans, nil
}

func (o *mqlOciApigatewayDeployment) corsAllowedOrigins() ([]any, error) {
	rp, err := o.requestPolicies()
	if err != nil {
		return nil, err
	}
	_, origins := flattenCors(rp)
	return origins, nil
}

func (o *mqlOciApigatewayDeployment) corsAllowCredentials() (bool, error) {
	rp, err := o.requestPolicies()
	if err != nil {
		return false, err
	}
	allow, _ := flattenCors(rp)
	return allow, nil
}

func (o *mqlOciApigatewayDeployment) rateLimitPerSecond() (int64, error) {
	rp, err := o.requestPolicies()
	if err != nil {
		return 0, err
	}
	rate, _ := flattenRateLimiting(rp)
	return rate, nil
}

func (o *mqlOciApigatewayDeployment) rateLimitKey() (string, error) {
	rp, err := o.requestPolicies()
	if err != nil {
		return "", err
	}
	_, key := flattenRateLimiting(rp)
	return key, nil
}

func (o *mqlOciApigatewayDeployment) hasDynamicAuthentication() (bool, error) {
	rp, err := o.requestPolicies()
	if err != nil {
		return false, err
	}
	return hasDynamicAuthentication(rp), nil
}

// API gateway certificates (apigateway service, distinct from certificatesmanagement)

func (o *mqlOciApigateway) certificates() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getCertificates(conn, list.Data), 5)
	poolOfJobs.Run()
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}
	return res, nil
}

func (o *mqlOciApigateway) getCertificates(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci api gateway certificates with region %s", regionResource.Id.Data)

			svc, err := conn.ApiGatewayClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var items []apigateway.CertificateSummary
			var page *string
			for {
				response, err := svc.ListCertificates(ctx, apigateway.ListCertificatesRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}
				items = append(items, response.Items...)
				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			res := make([]any, 0, len(items))
			for i := range items {
				c := items[i]

				var created, updated, notValidAfter *time.Time
				if c.TimeCreated != nil {
					created = &c.TimeCreated.Time
				}
				if c.TimeUpdated != nil {
					updated = &c.TimeUpdated.Time
				}
				if c.TimeNotValidAfter != nil {
					notValidAfter = &c.TimeNotValidAfter.Time
				}

				subjectNames := convert.SliceAnyToInterface(c.SubjectNames)

				freeformTags := make(map[string]interface{}, len(c.FreeformTags))
				for k, v := range c.FreeformTags {
					freeformTags[k] = v
				}
				definedTags := make(map[string]interface{}, len(c.DefinedTags))
				for k, v := range c.DefinedTags {
					definedTags[k] = v
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.apigateway.certificate", map[string]*llx.RawData{
					"id":                llx.StringDataPtr(c.Id),
					"name":              llx.StringDataPtr(c.DisplayName),
					"compartmentID":     llx.StringDataPtr(c.CompartmentId),
					"subjectNames":      llx.ArrayData(subjectNames, types.String),
					"timeNotValidAfter": llx.TimeDataPtr(notValidAfter),
					"state":             llx.StringData(string(c.LifecycleState)),
					"created":           llx.TimeDataPtr(created),
					"timeUpdated":       llx.TimeDataPtr(updated),
					"freeformTags":      llx.MapData(freeformTags, types.String),
					"definedTags":       llx.MapData(definedTags, types.Any),
					"systemTags":        llx.MapData(definedTagsToAny(c.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlInstance)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func initOciApigatewayCertificate(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch oci.apigateway.certificate")
	}
	idVal := args["id"].Value.(string)

	obj, err := CreateResource(runtime, "oci.apigateway", nil)
	if err != nil {
		return nil, nil, err
	}
	apigw := obj.(*mqlOciApigateway)
	rawCerts := apigw.GetCertificates()
	if rawCerts.Error != nil {
		return nil, nil, rawCerts.Error
	}
	for _, raw := range rawCerts.Data {
		c := raw.(*mqlOciApigatewayCertificate)
		if c.Id.Data == idVal {
			return args, c, nil
		}
	}
	return nil, nil, errors.New("oci.apigateway.certificate not found: " + idVal)
}

func (o *mqlOciApigatewayCertificate) id() (string, error) {
	return "oci.apigateway.certificate/" + o.Id.Data, nil
}
