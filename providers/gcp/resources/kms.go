// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/gcp/connection"
	"go.mondoo.com/cnquery/v10/types"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/cloud/location"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
)

func (g *mqlGcpProjectKmsService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	return fmt.Sprintf("%s/gcp.project.kmsService", projectId), nil
}

func initGcpProjectKmsService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GcpConnection)

	projectId := conn.ResourceID()
	args["projectId"] = llx.StringData(projectId)

	return args, nil, nil
}

func (g *mqlGcpProject) kms() (*mqlGcpProjectKmsService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.kmsService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectKmsService), nil
}

func (g *mqlGcpProjectKmsServiceKeyring) id() (string, error) {
	return g.ResourcePath.Data, g.ResourcePath.Error
}

func (g *mqlGcpProjectKmsServiceKeyringCryptokey) id() (string, error) {
	return g.ResourcePath.Data, g.ResourcePath.Error
}

func (g *mqlGcpProjectKmsServiceKeyringCryptokeyVersion) id() (string, error) {
	return g.ResourcePath.Data, g.ResourcePath.Error
}

func (g *mqlGcpProjectKmsServiceKeyringCryptokeyVersionAttestation) id() (string, error) {
	if g.CryptoKeyVersionName.Error != nil {
		return "", g.CryptoKeyVersionName.Error
	}
	name := g.CryptoKeyVersionName.Data
	return fmt.Sprintf("%s/attestation", name), nil
}

func (g *mqlGcpProjectKmsServiceKeyringCryptokeyVersionExternalProtectionLevelOptions) id() (string, error) {
	if g.CryptoKeyVersionName.Error != nil {
		return "", g.CryptoKeyVersionName.Error
	}
	name := g.CryptoKeyVersionName.Data
	return fmt.Sprintf("%s/externalProtectionLevelOptions", name), nil
}

func (g *mqlGcpProjectKmsServiceKeyringCryptokeyVersionAttestationCertificatechains) id() (string, error) {
	if g.CryptoKeyVersionName.Error != nil {
		return "", g.CryptoKeyVersionName.Error
	}
	name := g.CryptoKeyVersionName.Data
	return fmt.Sprintf("%s/attestation/certchains", name), nil
}

func (g *mqlGcpProjectKmsService) locations() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(kms.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	kmsSvc, err := kms.NewKeyManagementClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer kmsSvc.Close()

	var locations []interface{}
	it := kmsSvc.ListLocations(ctx, &location.ListLocationsRequest{Name: fmt.Sprintf("projects/%s", projectId)})
	for {
		l, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		locations = append(locations, l.LocationId)
	}
	return locations, nil
}

func (g *mqlGcpProjectKmsService) keyrings() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	locations := g.GetLocations()
	if locations.Error != nil {
		return nil, locations.Error
	}

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(kms.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	kmsSvc, err := kms.NewKeyManagementClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer kmsSvc.Close()

	var keyrings []interface{}
	var wg sync.WaitGroup
	wg.Add(len(locations.Data))
	mux := &sync.Mutex{}

	for _, location := range locations.Data {
		go func(svc *kms.KeyManagementClient, project string, location string) {
			defer wg.Done()
			it := kmsSvc.ListKeyRings(ctx,
				&kmspb.ListKeyRingsRequest{Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, location)})
			for {
				k, err := it.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					log.Error().Err(err)
					return
				}

				created := k.CreateTime.AsTime()
				mqlKeyring, err := CreateResource(g.MqlRuntime, "gcp.project.kmsService.keyring", map[string]*llx.RawData{
					"projectId":    llx.StringData(projectId),
					"resourcePath": llx.StringData(k.Name),
					"name":         llx.StringData(parseResourceName(k.Name)),
					"created":      llx.TimeData(created),
					"location":     llx.StringData(location),
				})
				if err != nil {
					log.Error().Err(err)
					return
				}
				mux.Lock()
				keyrings = append(keyrings, mqlKeyring)
				mux.Unlock()
			}
		}(kmsSvc, projectId, location.(string))
	}
	wg.Wait()
	return keyrings, nil
}

func (g *mqlGcpProjectKmsServiceKeyring) cryptokeys() ([]interface{}, error) {
	if g.ResourcePath.Error != nil {
		return nil, g.ResourcePath.Error
	}
	keyring := g.ResourcePath.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(kms.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	kmsSvc, err := kms.NewKeyManagementClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer kmsSvc.Close()

	var keys []interface{}

	it := kmsSvc.ListCryptoKeys(ctx, &kmspb.ListCryptoKeysRequest{
		Parent: keyring,
	})

	type mqlVersionTemplate struct {
		ProtectionLevel string `json:"protectionLevel"`
		Algorithm       string `json:"algorithm"`
	}
	for {
		k, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var mqlPrimary plugin.Resource
		if k.Primary != nil {
			mqlPrimary, err = cryptoKeyVersionToMql(g.MqlRuntime, k.Primary)
			if err != nil {
				return nil, err
			}
		}

		var versionTemplate map[string]interface{}
		if k.VersionTemplate != nil {
			versionTemplate, err = convert.JsonToDict(mqlVersionTemplate{
				ProtectionLevel: k.VersionTemplate.ProtectionLevel.String(),
				Algorithm:       k.VersionTemplate.Algorithm.String(),
			})
			if err != nil {
				return nil, err
			}
		}

		var mqlRotationPeriod *time.Time
		rotationPeriod := k.GetRotationPeriod()
		if rotationPeriod != nil {
			v := llx.DurationToTime(rotationPeriod.Seconds)
			mqlRotationPeriod = &v
		}

		var mqlDestroyScheduledDuration *time.Time
		if k.DestroyScheduledDuration != nil {
			v := llx.DurationToTime(k.DestroyScheduledDuration.Seconds)
			mqlDestroyScheduledDuration = &v
		}

		mqlKey, err := CreateResource(g.MqlRuntime, "gcp.project.kmsService.keyring.cryptokey", map[string]*llx.RawData{
			"resourcePath":             llx.StringData(k.Name),
			"name":                     llx.StringData(parseResourceName(k.Name)),
			"primary":                  llx.ResourceData(mqlPrimary, "gcp.project.kmsService.keyring.cryptokey.version"),
			"purpose":                  llx.StringData(k.Purpose.String()),
			"created":                  llx.TimeData(k.CreateTime.AsTime()),
			"nextRotation":             llx.TimeData(k.NextRotationTime.AsTime()),
			"rotationPeriod":           llx.TimeDataPtr(mqlRotationPeriod),
			"versionTemplate":          llx.DictData(versionTemplate),
			"labels":                   llx.MapData(convert.MapToInterfaceMap(k.Labels), types.String),
			"importOnly":               llx.BoolData(k.ImportOnly),
			"destroyScheduledDuration": llx.TimeDataPtr(mqlDestroyScheduledDuration),
			"cryptoKeyBackend":         llx.StringData(k.CryptoKeyBackend),
		})

		keys = append(keys, mqlKey)
	}
	return keys, nil
}

func (g *mqlGcpProjectKmsServiceKeyringCryptokey) versions() ([]interface{}, error) {
	if g.ResourcePath.Error != nil {
		return nil, g.ResourcePath.Error
	}
	cryptokey := g.ResourcePath.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(kms.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	kmsSvc, err := kms.NewKeyManagementClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer kmsSvc.Close()

	var versions []interface{}

	it := kmsSvc.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
		Parent: cryptokey,
	})

	for {
		v, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		mqlVersion, err := cryptoKeyVersionToMql(g.MqlRuntime, v)
		versions = append(versions, mqlVersion)
	}
	return versions, nil
}

func (g *mqlGcpProjectKmsServiceKeyringCryptokey) iamPolicy() ([]interface{}, error) {
	if g.ResourcePath.Error != nil {
		return nil, g.ResourcePath.Error
	}
	cryptokey := g.ResourcePath.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(kms.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	kmsSvc, err := kms.NewKeyManagementClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer kmsSvc.Close()

	policy, err := kmsSvc.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: cryptokey})
	if err != nil {
		return nil, err
	}
	res := make([]interface{}, 0, len(policy.Bindings))
	for i, b := range policy.Bindings {
		mqlBinding, err := CreateResource(g.MqlRuntime, "gcp.resourcemanager.binding", map[string]*llx.RawData{
			"id":      llx.StringData(cryptokey + "-" + strconv.Itoa(i)),
			"role":    llx.StringData(b.Role),
			"members": llx.ArrayData(convert.SliceAnyToInterface(b.Members), types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBinding)
	}
	return res, nil
}

func cryptoKeyVersionToMql(runtime *plugin.Runtime, v *kmspb.CryptoKeyVersion) (plugin.Resource, error) {
	var mqlAttestation plugin.Resource
	if v.Attestation != nil {
		mqlAttestationCertChains, err := CreateResource(runtime, "gcp.project.kmsService.keyring.cryptokey.version.attestation.certificatechains", map[string]*llx.RawData{
			"cryptoKeyVersionName": llx.StringData(v.Name),
			"caviumCerts":          llx.ArrayData(convert.SliceAnyToInterface(v.Attestation.CertChains.CaviumCerts), types.String),
			"googleCardCerts":      llx.ArrayData(convert.SliceAnyToInterface(v.Attestation.CertChains.GoogleCardCerts), types.String),
			"googlePartitionCerts": llx.ArrayData(convert.SliceAnyToInterface(v.Attestation.CertChains.GooglePartitionCerts), types.String),
		})
		if err != nil {
			return nil, err
		}

		mqlAttestation, err = CreateResource(runtime, "gcp.project.kmsService.keyring.cryptokey.version.attestation", map[string]*llx.RawData{
			"cryptoKeyVersionName": llx.StringData(v.Name),
			"format":               llx.StringData(v.Attestation.Format.String()),
			"certificateChains":    llx.ResourceData(mqlAttestationCertChains, "gcp.project.kmsService.keyring.cryptokey.version.attestation.certificatechains"),
		})
		if err != nil {
			return nil, err
		}
	}

	var mqlExtProtOpts plugin.Resource
	var err error
	if v.ExternalProtectionLevelOptions != nil {
		mqlExtProtOpts, err = CreateResource(runtime, "gcp.project.kmsService.keyring.cryptokey.version.externalProtectionLevelOptions", map[string]*llx.RawData{
			"cryptoKeyVersionName": llx.StringData(v.Name),
			"externalKeyUri":       llx.StringData(v.ExternalProtectionLevelOptions.ExternalKeyUri),
			"ekmConnectionKeyPath": llx.StringData(v.ExternalProtectionLevelOptions.EkmConnectionKeyPath),
		})
		if err != nil {
			return nil, err
		}
	}
	return CreateResource(runtime, "gcp.project.kmsService.keyring.cryptokey.version", map[string]*llx.RawData{
		"resourcePath":                   llx.StringData(v.Name),
		"name":                           llx.StringData(parseResourceName(v.Name)),
		"state":                          llx.StringData(v.State.String()),
		"protectionLevel":                llx.StringData(v.ProtectionLevel.String()),
		"algorithm":                      llx.StringData(v.Algorithm.String()),
		"attestation":                    llx.ResourceData(mqlAttestation, "gcp.project.kmsService.keyring.cryptokey.version.attestation"),
		"created":                        llx.TimeDataPtr(timestampAsTimePtr(v.CreateTime)),
		"generated":                      llx.TimeDataPtr(timestampAsTimePtr(v.GenerateTime)),
		"destroyed":                      llx.TimeDataPtr(timestampAsTimePtr(v.DestroyTime)),
		"destroyEventTime":               llx.TimeDataPtr(timestampAsTimePtr(v.DestroyEventTime)),
		"importJob":                      llx.StringData(v.ImportJob),
		"importTime":                     llx.TimeDataPtr(timestampAsTimePtr(v.ImportTime)),
		"importFailureReason":            llx.StringData(v.ImportFailureReason),
		"externalProtectionLevelOptions": llx.ResourceData(mqlExtProtOpts, "gcp.project.kmsService.keyring.cryptokey.version.externalProtectionLevelOptions"),
		"reimportEligible":               llx.BoolData(v.ReimportEligible),
	})
}
