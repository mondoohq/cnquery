package gcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (g *mqlGcpProjectKmsservice) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/gcp.project.kmsservice", projectId), nil
}

func (g *mqlGcpProjectKmsservice) init(args *resources.Args) (*resources.Args, GcpProjectKmsservice, error) {
	if len(*args) > 0 {
		return args, nil, nil
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	projectId := provider.ResourceID()
	(*args)["projectId"] = projectId

	return args, nil, nil
}

func (g *mqlGcpProject) GetKms() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.kmsservice",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectKmsserviceKeyring) id() (string, error) {
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return name, nil
}

func (g *mqlGcpProjectKmsserviceKeyringCryptokey) id() (string, error) {
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return name, nil
}

func (g *mqlGcpProjectKmsserviceKeyringCryptokeyVersion) id() (string, error) {
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return name, nil
}

func (g *mqlGcpProjectKmsserviceKeyringCryptokeyVersionAttestation) id() (string, error) {
	name, err := g.CryptoKeyVersionName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/attestation", name), nil
}

func (g *mqlGcpProjectKmsserviceKeyringCryptokeyVersionExternalProtectionLevelOptions) id() (string, error) {
	name, err := g.CryptoKeyVersionName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/externalProtectionLevelOptions", name), nil
}

func (g *mqlGcpProjectKmsserviceKeyringCryptokeyVersionAttestationCertificatechains) id() (string, error) {
	name, err := g.CryptoKeyVersionName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/attestation/certchains", name), nil
}

func (g *mqlGcpProjectKmsservice) GetLocations() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(kms.DefaultAuthScopes()...)
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

func (g *mqlGcpProjectKmsservice) GetKeyrings() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	locations, err := g.Locations()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(kms.DefaultAuthScopes()...)
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
	wg.Add(len(locations))
	mux := &sync.Mutex{}

	for _, location := range locations {
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
				mqlKeyring, err := g.MotorRuntime.CreateResource("gcp.project.kmsservice.keyring",
					"projectId", projectId,
					"name", k.Name,
					"created", &created,
					"location", location,
				)
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

func (g *mqlGcpProjectKmsserviceKeyring) GetCryptokeys() ([]interface{}, error) {
	keyring, err := g.Name()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(kms.DefaultAuthScopes()...)
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

	for {
		k, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		mqlPrimary, err := cryptoKeyVersionToMql(g.MotorRuntime, k.Primary)
		if err != nil {
			return nil, err
		}

		mqlKey, err := g.MotorRuntime.CreateResource("gcp.project.kmsservice.keyring.cryptokey",
			"name", k.Name,
			"primary", mqlPrimary,
			"purpose", k.Purpose.String(),
		)

		keys = append(keys, mqlKey)
	}
	return keys, nil
}

func (g *mqlGcpProjectKmsserviceKeyringCryptokey) GetVersions() ([]interface{}, error) {
	cryptokey, err := g.Name()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(kms.DefaultAuthScopes()...)
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

		mqlVersion, err := cryptoKeyVersionToMql(g.MotorRuntime, v)
		versions = append(versions, mqlVersion)
	}
	return versions, nil
}

func cryptoKeyVersionToMql(runtime *resources.Runtime, v *kmspb.CryptoKeyVersion) (resources.ResourceType, error) {
	var mqlAttestation resources.ResourceType
	if v.Attestation != nil {
		mqlAttestationCertChains, err := runtime.CreateResource("gcp.project.kmsservice.keyring.cryptokey.version.attestation.certificatechains",
			"cryptoKeyVersionName", v.Name,
			"caviumCerts", core.StrSliceToInterface(v.Attestation.CertChains.CaviumCerts),
			"googleCardCerts", core.StrSliceToInterface(v.Attestation.CertChains.GoogleCardCerts),
			"googlePartitionCerts", core.StrSliceToInterface(v.Attestation.CertChains.GooglePartitionCerts),
		)
		if err != nil {
			return nil, err
		}

		mqlAttestation, err = runtime.CreateResource("gcp.project.kmsservice.keyring.cryptokey.version.attestation",
			"cryptoKeyVersionName", v.Name,
			"format", v.Attestation.Format.String(),
			"certificateChains", mqlAttestationCertChains,
		)
		if err != nil {
			return nil, err
		}
	}

	var mqlExtProtOpts resources.ResourceType
	var err error
	if v.ExternalProtectionLevelOptions != nil {
		mqlExtProtOpts, err = runtime.CreateResource("gcp.project.kmsservice.keyring.cryptokey.version.externalProtectionLevelOptions",
			"cryptoKeyVersionName", v.Name,
			"externalKeyUri", v.ExternalProtectionLevelOptions.ExternalKeyUri,
			"ekmConnectionKeyPath", v.ExternalProtectionLevelOptions.EkmConnectionKeyPath,
		)
		if err != nil {
			return nil, err
		}
	}
	return runtime.CreateResource("gcp.project.kmsservice.keyring.cryptokey.version",
		"name", v.Name,
		"state", v.State.String(),
		"protectionLevel", v.ProtectionLevel.String(),
		"algorithm", v.Algorithm.String(),
		"attestation", mqlAttestation,
		"created", timestampAsTimePtr(v.CreateTime),
		"generated", timestampAsTimePtr(v.GenerateTime),
		"destroyed", timestampAsTimePtr(v.DestroyTime),
		"destroyEventTime", timestampAsTimePtr(v.DestroyEventTime),
		"importJob", v.ImportJob,
		"importTime", timestampAsTimePtr(v.ImportTime),
		"importFailureReason", v.ImportFailureReason,
		"externalProtectionLevelOptions", mqlExtProtOpts,
		"reimportEligible", v.ReimportEligible,
	)
}

func timestampAsTimePtr(t *timestamppb.Timestamp) *time.Time {
	if t == nil {
		return nil
	}
	tm := t.AsTime()
	return &tm
}
