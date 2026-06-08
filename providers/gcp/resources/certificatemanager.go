// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	certificatemanager "cloud.google.com/go/certificatemanager/apiv1"
	"cloud.google.com/go/certificatemanager/apiv1/certificatemanagerpb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProject) certificateManager() (*mqlGcpProjectCertificateManagerService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.certificateManagerService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}

	serviceEnabled, err := g.isServiceEnabled(service_certificatemanager)
	if err != nil {
		return nil, err
	}

	svc := res.(*mqlGcpProjectCertificateManagerService)
	svc.serviceEnabled = serviceEnabled
	if !serviceEnabled {
		log.Debug().Str("service", service_certificatemanager).Msg("gcp service is not enabled, skipping")
	}
	return svc, nil
}

type mqlGcpProjectCertificateManagerServiceInternal struct {
	serviceEnabled bool
}

type mqlGcpProjectCertificateManagerServiceCertificateInternal struct {
	cacheManagedDnsAuthorizationNames []string
	cacheManagedIssuanceConfigName    string
}

type mqlGcpProjectCertificateManagerServiceCertificateMapEntryInternal struct {
	cacheCertificateNames []string
}

func initGcpProjectCertificateManagerService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}
	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	args["projectId"] = llx.StringData(conn.ResourceID())
	return args, nil, nil
}

func (g *mqlGcpProjectCertificateManagerService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf("gcp.project/%s/certificateManagerService", g.ProjectId.Data), nil
}

func (g *mqlGcpProjectCertificateManagerService) certificates() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(certificatemanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client, err := certificatemanager.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListCertificates(ctx, &certificatemanagerpb.ListCertificatesRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", projectId),
	})

	var res []any
	for {
		c, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		certType, managedDomains, managedAuthNames, managedIssuanceName, managedState, managedIssue, managedAttempts, err := flattenCertificate(c)
		if err != nil {
			return nil, err
		}

		mqlCert, err := newCertificateResource(g.MqlRuntime, projectId, c, certType, managedDomains, managedState, managedIssue, managedAttempts, managedAuthNames, managedIssuanceName)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCert)
	}
	return res, nil
}

// newCertificateResource builds the typed cert resource and stashes the raw
// dns-authorization names and issuance-config name in the internal cache so
// that the typed managedDnsAuthorizations() / managedIssuanceConfig()
// accessors can resolve them later via NewResource.
func newCertificateResource(
	runtime *plugin.Runtime,
	projectId string,
	c *certificatemanagerpb.Certificate,
	certType string,
	managedDomains []any,
	managedState string,
	managedIssue map[string]any,
	managedAttempts []any,
	managedAuthNames []string,
	managedIssuanceName string,
) (plugin.Resource, error) {
	mqlCert, err := CreateResource(runtime, "gcp.project.certificateManagerService.certificate", map[string]*llx.RawData{
		"projectId":                       llx.StringData(projectId),
		"resourcePath":                    llx.StringData(c.Name),
		"name":                            llx.StringData(parseResourceName(c.Name)),
		"location":                        llx.StringData(parseLocationFromPath(c.Name)),
		"description":                     llx.StringData(c.Description),
		"labels":                          llx.MapData(convert.MapToInterfaceMap(c.Labels), types.String),
		"createTime":                      llx.TimeDataPtr(timestampAsTimePtr(c.CreateTime)),
		"updateTime":                      llx.TimeDataPtr(timestampAsTimePtr(c.UpdateTime)),
		"expireTime":                      llx.TimeDataPtr(timestampAsTimePtr(c.ExpireTime)),
		"sanDnsnames":                     llx.ArrayData(convert.SliceAnyToInterface(c.SanDnsnames), types.String),
		"pemCertificate":                  llx.StringData(c.PemCertificate),
		"scope":                           llx.StringData(c.Scope.String()),
		"type":                            llx.StringData(certType),
		"managedDomains":                  llx.ArrayData(managedDomains, types.String),
		"managedState":                    llx.StringData(managedState),
		"managedProvisioningIssue":        llx.DictData(managedIssue),
		"managedAuthorizationAttemptInfo": llx.ArrayData(managedAttempts, types.Dict),
	})
	if err != nil {
		return nil, err
	}
	cert := mqlCert.(*mqlGcpProjectCertificateManagerServiceCertificate)
	cert.cacheManagedDnsAuthorizationNames = managedAuthNames
	cert.cacheManagedIssuanceConfigName = managedIssuanceName
	return mqlCert, nil
}

func (g *mqlGcpProjectCertificateManagerServiceCertificate) id() (string, error) {
	return g.ResourcePath.Data, g.ResourcePath.Error
}

func (g *mqlGcpProjectCertificateManagerServiceCertificate) expired() (bool, error) {
	return certExpired(g.ExpireTime)
}

func (g *mqlGcpProjectCertificateManagerServiceCertificate) daysUntilExpiry() (int64, error) {
	return certDaysUntilExpiry(g.ExpireTime)
}

func (g *mqlGcpProjectCertificateManagerService) certificateMaps() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(certificatemanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client, err := certificatemanager.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListCertificateMaps(ctx, &certificatemanagerpb.ListCertificateMapsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", projectId),
	})

	var res []any
	for {
		m, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		mqlMap, err := CreateResource(g.MqlRuntime, "gcp.project.certificateManagerService.certificateMap", map[string]*llx.RawData{
			"projectId":    llx.StringData(projectId),
			"resourcePath": llx.StringData(m.Name),
			"name":         llx.StringData(parseResourceName(m.Name)),
			"location":     llx.StringData(parseLocationFromPath(m.Name)),
			"description":  llx.StringData(m.Description),
			"labels":       llx.MapData(convert.MapToInterfaceMap(m.Labels), types.String),
			"createTime":   llx.TimeDataPtr(timestampAsTimePtr(m.CreateTime)),
			"updateTime":   llx.TimeDataPtr(timestampAsTimePtr(m.UpdateTime)),
			"gclbTargets":  llx.ArrayData(flattenGclbTargets(m.GclbTargets), types.Dict),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlMap)
	}
	return res, nil
}

func (g *mqlGcpProjectCertificateManagerServiceCertificateMap) id() (string, error) {
	return g.ResourcePath.Data, g.ResourcePath.Error
}

func (g *mqlGcpProjectCertificateManagerServiceCertificateMap) entries() ([]any, error) {
	if g.ResourcePath.Error != nil {
		return nil, g.ResourcePath.Error
	}
	mapPath := g.ResourcePath.Data
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	mapName := g.Name.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(certificatemanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client, err := certificatemanager.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListCertificateMapEntries(ctx, &certificatemanagerpb.ListCertificateMapEntriesRequest{
		Parent: mapPath,
	})

	var res []any
	for {
		e, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		hostname, matcher := flattenCertMapEntryMatch(e)

		mqlEntry, err := CreateResource(g.MqlRuntime, "gcp.project.certificateManagerService.certificateMapEntry", map[string]*llx.RawData{
			"projectId":      llx.StringData(projectId),
			"resourcePath":   llx.StringData(e.Name),
			"name":           llx.StringData(parseResourceName(e.Name)),
			"location":       llx.StringData(parseLocationFromPath(e.Name)),
			"certificateMap": llx.StringData(mapName),
			"description":    llx.StringData(e.Description),
			"labels":         llx.MapData(convert.MapToInterfaceMap(e.Labels), types.String),
			"createTime":     llx.TimeDataPtr(timestampAsTimePtr(e.CreateTime)),
			"updateTime":     llx.TimeDataPtr(timestampAsTimePtr(e.UpdateTime)),
			"hostname":       llx.StringData(hostname),
			"matcher":        llx.StringData(matcher),
			"state":          llx.StringData(e.State.String()),
		})
		if err != nil {
			return nil, err
		}
		entry := mqlEntry.(*mqlGcpProjectCertificateManagerServiceCertificateMapEntry)
		entry.cacheCertificateNames = append([]string(nil), e.Certificates...)
		res = append(res, mqlEntry)
	}
	return res, nil
}

func (g *mqlGcpProjectCertificateManagerServiceCertificateMapEntry) id() (string, error) {
	return g.ResourcePath.Data, g.ResourcePath.Error
}

func (g *mqlGcpProjectCertificateManagerService) dnsAuthorizations() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(certificatemanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client, err := certificatemanager.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListDnsAuthorizations(ctx, &certificatemanagerpb.ListDnsAuthorizationsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", projectId),
	})

	var res []any
	for {
		a, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var record map[string]any
		if a.DnsResourceRecord != nil {
			record = map[string]any{
				"name": a.DnsResourceRecord.Name,
				"type": a.DnsResourceRecord.Type,
				"data": a.DnsResourceRecord.Data,
			}
		}

		mqlAuth, err := CreateResource(g.MqlRuntime, "gcp.project.certificateManagerService.dnsAuthorization", map[string]*llx.RawData{
			"projectId":         llx.StringData(projectId),
			"resourcePath":      llx.StringData(a.Name),
			"name":              llx.StringData(parseResourceName(a.Name)),
			"location":          llx.StringData(parseLocationFromPath(a.Name)),
			"description":       llx.StringData(a.Description),
			"labels":            llx.MapData(convert.MapToInterfaceMap(a.Labels), types.String),
			"createTime":        llx.TimeDataPtr(timestampAsTimePtr(a.CreateTime)),
			"updateTime":        llx.TimeDataPtr(timestampAsTimePtr(a.UpdateTime)),
			"domain":            llx.StringData(a.Domain),
			"type":              llx.StringData(a.Type.String()),
			"dnsResourceRecord": llx.DictData(record),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAuth)
	}
	return res, nil
}

func (g *mqlGcpProjectCertificateManagerServiceDnsAuthorization) id() (string, error) {
	return g.ResourcePath.Data, g.ResourcePath.Error
}

func (g *mqlGcpProjectCertificateManagerService) certificateIssuanceConfigs() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(certificatemanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client, err := certificatemanager.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListCertificateIssuanceConfigs(ctx, &certificatemanagerpb.ListCertificateIssuanceConfigsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", projectId),
	})

	var res []any
	for {
		c, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		caConfig, err := protoToDict(c.CertificateAuthorityConfig)
		if err != nil {
			return nil, err
		}
		lifetime := ""
		if c.Lifetime != nil {
			lifetime = fmt.Sprintf("%ds", c.Lifetime.Seconds)
		}

		mqlCfg, err := CreateResource(g.MqlRuntime, "gcp.project.certificateManagerService.certificateIssuanceConfig", map[string]*llx.RawData{
			"projectId":                  llx.StringData(projectId),
			"resourcePath":               llx.StringData(c.Name),
			"name":                       llx.StringData(parseResourceName(c.Name)),
			"location":                   llx.StringData(parseLocationFromPath(c.Name)),
			"description":                llx.StringData(c.Description),
			"labels":                     llx.MapData(convert.MapToInterfaceMap(c.Labels), types.String),
			"createTime":                 llx.TimeDataPtr(timestampAsTimePtr(c.CreateTime)),
			"updateTime":                 llx.TimeDataPtr(timestampAsTimePtr(c.UpdateTime)),
			"certificateAuthorityConfig": llx.DictData(caConfig),
			"lifetime":                   llx.StringData(lifetime),
			"rotationWindowPercentage":   llx.IntData(int64(c.RotationWindowPercentage)),
			"keyAlgorithm":               llx.StringData(c.KeyAlgorithm.String()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCfg)
	}
	return res, nil
}

func (g *mqlGcpProjectCertificateManagerServiceCertificateIssuanceConfig) id() (string, error) {
	return g.ResourcePath.Data, g.ResourcePath.Error
}

func (g *mqlGcpProjectCertificateManagerService) trustConfigs() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(certificatemanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client, err := certificatemanager.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListTrustConfigs(ctx, &certificatemanagerpb.ListTrustConfigsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", projectId),
	})

	var res []any
	for {
		t, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		mqlTc, err := CreateResource(g.MqlRuntime, "gcp.project.certificateManagerService.trustConfig", map[string]*llx.RawData{
			"projectId":    llx.StringData(projectId),
			"resourcePath": llx.StringData(t.Name),
			"name":         llx.StringData(parseResourceName(t.Name)),
			"location":     llx.StringData(parseLocationFromPath(t.Name)),
			"description":  llx.StringData(t.Description),
			"labels":       llx.MapData(convert.MapToInterfaceMap(t.Labels), types.String),
			"createTime":   llx.TimeDataPtr(timestampAsTimePtr(t.CreateTime)),
			"updateTime":   llx.TimeDataPtr(timestampAsTimePtr(t.UpdateTime)),
			"etag":         llx.StringData(t.Etag),
			"trustStores":  llx.ArrayData(flattenTrustStores(t.TrustStores), types.Dict),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlTc)
	}
	return res, nil
}

func (g *mqlGcpProjectCertificateManagerServiceTrustConfig) id() (string, error) {
	return g.ResourcePath.Data, g.ResourcePath.Error
}

// flattenCertificate extracts the type discriminator and per-type fields from
// a Certificate. SelfManaged certs only ever have output-only fields visible
// at the top level (PemCertificate, ExpireTime), so all the populated
// per-type fields belong to the managed branch. The dnsAuthorization names
// and issuance-config name come back as raw strings so they can be cached
// for typed-accessor resolution.
func flattenCertificate(c *certificatemanagerpb.Certificate) (
	certType string,
	managedDomains []any,
	managedAuthNames []string,
	managedIssuanceName, managedState string,
	managedIssue map[string]any,
	managedAttempts []any,
	err error,
) {
	managedDomains = []any{}
	managedAuthNames = []string{}
	managedAttempts = []any{}

	switch t := c.Type.(type) {
	case *certificatemanagerpb.Certificate_SelfManaged:
		certType = "selfManaged"
	case *certificatemanagerpb.Certificate_Managed:
		certType = "managed"
		if t.Managed != nil {
			for _, d := range t.Managed.Domains {
				managedDomains = append(managedDomains, d)
			}
			managedAuthNames = append(managedAuthNames, t.Managed.DnsAuthorizations...)
			managedIssuanceName = t.Managed.IssuanceConfig
			managedState = t.Managed.State.String()
			if t.Managed.ProvisioningIssue != nil {
				managedIssue, err = protoToDict(t.Managed.ProvisioningIssue)
				if err != nil {
					return
				}
			}
			for _, a := range t.Managed.AuthorizationAttemptInfo {
				d, derr := protoToDict(a)
				if derr != nil {
					err = derr
					return
				}
				managedAttempts = append(managedAttempts, d)
			}
		}
	}
	return
}

// flattenCertMapEntryMatch returns the (hostname, matcher) discriminator
// values for a CertificateMapEntry. Exactly one of Hostname or Matcher_ is
// set on the proto; the other field returns the empty string.
func flattenCertMapEntryMatch(e *certificatemanagerpb.CertificateMapEntry) (hostname, matcher string) {
	switch m := e.Match.(type) {
	case *certificatemanagerpb.CertificateMapEntry_Hostname:
		hostname = m.Hostname
	case *certificatemanagerpb.CertificateMapEntry_Matcher_:
		matcher = m.Matcher.String()
	}
	return
}

func flattenGclbTargets(targets []*certificatemanagerpb.CertificateMap_GclbTarget) []any {
	res := make([]any, 0, len(targets))
	for _, t := range targets {
		entry := map[string]any{}
		switch tp := t.TargetProxy.(type) {
		case *certificatemanagerpb.CertificateMap_GclbTarget_TargetHttpsProxy:
			entry["targetHttpsProxy"] = tp.TargetHttpsProxy
		case *certificatemanagerpb.CertificateMap_GclbTarget_TargetSslProxy:
			entry["targetSslProxy"] = tp.TargetSslProxy
		}
		ipConfigs := make([]any, 0, len(t.IpConfigs))
		for _, ip := range t.IpConfigs {
			ports := make([]any, 0, len(ip.Ports))
			for _, p := range ip.Ports {
				ports = append(ports, int64(p))
			}
			ipConfigs = append(ipConfigs, map[string]any{
				"ipAddress": ip.IpAddress,
				"ports":     ports,
			})
		}
		entry["ipConfigs"] = ipConfigs
		res = append(res, entry)
	}
	return res
}

func flattenTrustStores(stores []*certificatemanagerpb.TrustConfig_TrustStore) []any {
	res := make([]any, 0, len(stores))
	for _, s := range stores {
		anchors := make([]any, 0, len(s.TrustAnchors))
		for _, a := range s.TrustAnchors {
			anchors = append(anchors, map[string]any{"pemCertificate": a.GetPemCertificate()})
		}
		intermediates := make([]any, 0, len(s.IntermediateCas))
		for _, i := range s.IntermediateCas {
			intermediates = append(intermediates, map[string]any{"pemCertificate": i.GetPemCertificate()})
		}
		res = append(res, map[string]any{
			"trustAnchors":    anchors,
			"intermediateCas": intermediates,
		})
	}
	return res
}

func (g *mqlGcpProjectCertificateManagerServiceCertificate) managedDnsAuthorizations() ([]any, error) {
	res := make([]any, 0, len(g.cacheManagedDnsAuthorizationNames))
	for _, name := range g.cacheManagedDnsAuthorizationNames {
		if name == "" {
			continue
		}
		ref, err := NewResource(g.MqlRuntime, "gcp.project.certificateManagerService.dnsAuthorization",
			map[string]*llx.RawData{"resourcePath": llx.StringData(name)})
		if err != nil {
			return nil, err
		}
		res = append(res, ref)
	}
	return res, nil
}

func (g *mqlGcpProjectCertificateManagerServiceCertificate) managedIssuanceConfig() (*mqlGcpProjectCertificateManagerServiceCertificateIssuanceConfig, error) {
	if g.cacheManagedIssuanceConfigName == "" {
		g.ManagedIssuanceConfig.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	ref, err := NewResource(g.MqlRuntime, "gcp.project.certificateManagerService.certificateIssuanceConfig",
		map[string]*llx.RawData{"resourcePath": llx.StringData(g.cacheManagedIssuanceConfigName)})
	if err != nil {
		return nil, err
	}
	return ref.(*mqlGcpProjectCertificateManagerServiceCertificateIssuanceConfig), nil
}

func (g *mqlGcpProjectCertificateManagerServiceCertificateMapEntry) certificates() ([]any, error) {
	res := make([]any, 0, len(g.cacheCertificateNames))
	for _, name := range g.cacheCertificateNames {
		if name == "" {
			continue
		}
		ref, err := NewResource(g.MqlRuntime, "gcp.project.certificateManagerService.certificate",
			map[string]*llx.RawData{"resourcePath": llx.StringData(name)})
		if err != nil {
			return nil, err
		}
		res = append(res, ref)
	}
	return res, nil
}

// initGcpProjectCertificateManagerServiceCertificate resolves a Certificate
// from its full resourcePath (projects/*/locations/*/certificates/*). When
// invoked from a typed cross-reference, only `resourcePath` is set on args
// and the cert hasn't been listed yet, so we fetch it directly via
// GetCertificate and populate the cache fields used by the typed accessors.
func initGcpProjectCertificateManagerServiceCertificate(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	resourcePathRaw, ok := args["resourcePath"]
	if !ok || resourcePathRaw == nil || resourcePathRaw.Value == nil || resourcePathRaw.Value.(string) == "" {
		return args, nil, nil
	}
	resourcePath := resourcePathRaw.Value.(string)

	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	creds, err := conn.Credentials(certificatemanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, nil, err
	}
	ctx := context.Background()
	client, err := certificatemanager.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, nil, err
	}
	defer client.Close()

	c, err := client.GetCertificate(ctx, &certificatemanagerpb.GetCertificateRequest{Name: resourcePath})
	if err != nil {
		return nil, nil, err
	}

	certType, managedDomains, managedAuthNames, managedIssuanceName, managedState, managedIssue, managedAttempts, err := flattenCertificate(c)
	if err != nil {
		return nil, nil, err
	}

	res, err := newCertificateResource(runtime, conn.ResourceID(), c, certType, managedDomains, managedState, managedIssue, managedAttempts, managedAuthNames, managedIssuanceName)
	if err != nil {
		return nil, nil, err
	}
	return args, res, nil
}

// initGcpProjectCertificateManagerServiceDnsAuthorization resolves a
// DnsAuthorization from its full resourcePath via GetDnsAuthorization.
func initGcpProjectCertificateManagerServiceDnsAuthorization(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	resourcePathRaw, ok := args["resourcePath"]
	if !ok || resourcePathRaw == nil || resourcePathRaw.Value == nil || resourcePathRaw.Value.(string) == "" {
		return args, nil, nil
	}
	resourcePath := resourcePathRaw.Value.(string)

	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	creds, err := conn.Credentials(certificatemanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, nil, err
	}
	ctx := context.Background()
	client, err := certificatemanager.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, nil, err
	}
	defer client.Close()

	a, err := client.GetDnsAuthorization(ctx, &certificatemanagerpb.GetDnsAuthorizationRequest{Name: resourcePath})
	if err != nil {
		return nil, nil, err
	}

	var record map[string]any
	if a.DnsResourceRecord != nil {
		record = map[string]any{
			"name": a.DnsResourceRecord.Name,
			"type": a.DnsResourceRecord.Type,
			"data": a.DnsResourceRecord.Data,
		}
	}

	res, err := CreateResource(runtime, "gcp.project.certificateManagerService.dnsAuthorization", map[string]*llx.RawData{
		"projectId":         llx.StringData(conn.ResourceID()),
		"resourcePath":      llx.StringData(a.Name),
		"name":              llx.StringData(parseResourceName(a.Name)),
		"location":          llx.StringData(parseLocationFromPath(a.Name)),
		"description":       llx.StringData(a.Description),
		"labels":            llx.MapData(convert.MapToInterfaceMap(a.Labels), types.String),
		"createTime":        llx.TimeDataPtr(timestampAsTimePtr(a.CreateTime)),
		"updateTime":        llx.TimeDataPtr(timestampAsTimePtr(a.UpdateTime)),
		"domain":            llx.StringData(a.Domain),
		"type":              llx.StringData(a.Type.String()),
		"dnsResourceRecord": llx.DictData(record),
	})
	if err != nil {
		return nil, nil, err
	}
	return args, res, nil
}

// initGcpProjectCertificateManagerServiceCertificateIssuanceConfig resolves
// a CertificateIssuanceConfig from its full resourcePath via
// GetCertificateIssuanceConfig.
func initGcpProjectCertificateManagerServiceCertificateIssuanceConfig(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	resourcePathRaw, ok := args["resourcePath"]
	if !ok || resourcePathRaw == nil || resourcePathRaw.Value == nil || resourcePathRaw.Value.(string) == "" {
		return args, nil, nil
	}
	resourcePath := resourcePathRaw.Value.(string)

	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	creds, err := conn.Credentials(certificatemanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, nil, err
	}
	ctx := context.Background()
	client, err := certificatemanager.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, nil, err
	}
	defer client.Close()

	c, err := client.GetCertificateIssuanceConfig(ctx, &certificatemanagerpb.GetCertificateIssuanceConfigRequest{Name: resourcePath})
	if err != nil {
		return nil, nil, err
	}

	caConfig, err := protoToDict(c.CertificateAuthorityConfig)
	if err != nil {
		return nil, nil, err
	}
	lifetime := ""
	if c.Lifetime != nil {
		lifetime = fmt.Sprintf("%ds", c.Lifetime.Seconds)
	}

	res, err := CreateResource(runtime, "gcp.project.certificateManagerService.certificateIssuanceConfig", map[string]*llx.RawData{
		"projectId":                  llx.StringData(conn.ResourceID()),
		"resourcePath":               llx.StringData(c.Name),
		"name":                       llx.StringData(parseResourceName(c.Name)),
		"location":                   llx.StringData(parseLocationFromPath(c.Name)),
		"description":                llx.StringData(c.Description),
		"labels":                     llx.MapData(convert.MapToInterfaceMap(c.Labels), types.String),
		"createTime":                 llx.TimeDataPtr(timestampAsTimePtr(c.CreateTime)),
		"updateTime":                 llx.TimeDataPtr(timestampAsTimePtr(c.UpdateTime)),
		"certificateAuthorityConfig": llx.DictData(caConfig),
		"lifetime":                   llx.StringData(lifetime),
		"rotationWindowPercentage":   llx.IntData(int64(c.RotationWindowPercentage)),
		"keyAlgorithm":               llx.StringData(c.KeyAlgorithm.String()),
	})
	if err != nil {
		return nil, nil, err
	}
	return args, res, nil
}
