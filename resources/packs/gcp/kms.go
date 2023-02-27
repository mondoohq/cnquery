package gcp

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/cloud/location"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
)

func (g *mqlGcpProjectKmsService) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/gcp.project.kmsService", projectId), nil
}

func (g *mqlGcpProjectKmsService) init(args *resources.Args) (*resources.Args, GcpProjectKmsService, error) {
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

	return g.MotorRuntime.CreateResource("gcp.project.kmsService",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectKmsServiceKeyring) id() (string, error) {
	return g.ResourcePath()
}

func (g *mqlGcpProjectKmsServiceKeyringCryptokey) id() (string, error) {
	return g.ResourcePath()
}

func (g *mqlGcpProjectKmsServiceKeyringCryptokeyVersion) id() (string, error) {
	return g.ResourcePath()
}

func (g *mqlGcpProjectKmsServiceKeyringCryptokeyVersionAttestation) id() (string, error) {
	name, err := g.CryptoKeyVersionName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/attestation", name), nil
}

func (g *mqlGcpProjectKmsServiceKeyringCryptokeyVersionExternalProtectionLevelOptions) id() (string, error) {
	name, err := g.CryptoKeyVersionName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/externalProtectionLevelOptions", name), nil
}

func (g *mqlGcpProjectKmsServiceKeyringCryptokeyVersionAttestationCertificatechains) id() (string, error) {
	name, err := g.CryptoKeyVersionName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/attestation/certchains", name), nil
}

func (g *mqlGcpProjectKmsService) GetLocations() ([]interface{}, error) {
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

func (g *mqlGcpProjectKmsService) GetKeyrings() ([]interface{}, error) {
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
				mqlKeyring, err := g.MotorRuntime.CreateResource("gcp.project.kmsService.keyring",
					"projectId", projectId,
					"resourcePath", k.Name,
					"name", parseResourceName(k.Name),
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

func (g *mqlGcpProjectKmsServiceKeyring) GetCryptokeys() ([]interface{}, error) {
	keyring, err := g.ResourcePath()
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

		var mqlPrimary interface{}
		if k.Primary != nil {
			mqlPrimary, err = cryptoKeyVersionToMql(g.MotorRuntime, k.Primary)
			if err != nil {
				return nil, err
			}
		}

		var versionTemplate map[string]interface{}
		if k.VersionTemplate != nil {
			versionTemplate, err = core.JsonToDict(mqlVersionTemplate{
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
			mqlRotationPeriod = core.MqlTime(llx.DurationToTime(rotationPeriod.Seconds))
		}

		var mqlDestroyScheduledDuration *time.Time
		if k.DestroyScheduledDuration != nil {
			mqlDestroyScheduledDuration = core.MqlTime(llx.DurationToTime(k.DestroyScheduledDuration.Seconds))
		}

		mqlKey, err := g.MotorRuntime.CreateResource("gcp.project.kmsService.keyring.cryptokey",
			"resourcePath", k.Name,
			"name", parseResourceName(k.Name),
			"primary", mqlPrimary,
			"purpose", k.Purpose.String(),
			"created", core.MqlTime(k.CreateTime.AsTime()),
			"nextRotation", core.MqlTime(k.NextRotationTime.AsTime()),
			"rotationPeriod", mqlRotationPeriod,
			"versionTemplate", versionTemplate,
			"labels", core.StrMapToInterface(k.Labels),
			"importOnly", k.ImportOnly,
			"destroyScheduledDuration", mqlDestroyScheduledDuration,
			"cryptoKeyBackend", k.CryptoKeyBackend,
		)

		keys = append(keys, mqlKey)
	}
	return keys, nil
}

func (g *mqlGcpProjectKmsServiceKeyringCryptokey) GetVersions() ([]interface{}, error) {
	cryptokey, err := g.ResourcePath()
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

func (g *mqlGcpProjectKmsServiceKeyringCryptokey) GetIamPolicy() ([]interface{}, error) {
	cryptokey, err := g.ResourcePath()
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

	policy, err := kmsSvc.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: cryptokey})
	if err != nil {
		return nil, err
	}
	res := make([]interface{}, 0, len(policy.Bindings))
	for i, b := range policy.Bindings {
		mqlBinding, err := g.MotorRuntime.CreateResource("gcp.resourcemanager.binding",
			"id", cryptokey+"-"+strconv.Itoa(i),
			"role", b.Role,
			"members", core.SliceToInterfaceSlice(b.Members),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBinding)
	}
	return res, nil
}

func cryptoKeyVersionToMql(runtime *resources.Runtime, v *kmspb.CryptoKeyVersion) (resources.ResourceType, error) {
	var mqlAttestation resources.ResourceType
	if v.Attestation != nil {
		mqlAttestationCertChains, err := runtime.CreateResource("gcp.project.kmsService.keyring.cryptokey.version.attestation.certificatechains",
			"cryptoKeyVersionName", v.Name,
			"caviumCerts", core.SliceToInterfaceSlice(v.Attestation.CertChains.CaviumCerts),
			"googleCardCerts", core.SliceToInterfaceSlice(v.Attestation.CertChains.GoogleCardCerts),
			"googlePartitionCerts", core.SliceToInterfaceSlice(v.Attestation.CertChains.GooglePartitionCerts),
		)
		if err != nil {
			return nil, err
		}

		mqlAttestation, err = runtime.CreateResource("gcp.project.kmsService.keyring.cryptokey.version.attestation",
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
		mqlExtProtOpts, err = runtime.CreateResource("gcp.project.kmsService.keyring.cryptokey.version.externalProtectionLevelOptions",
			"cryptoKeyVersionName", v.Name,
			"externalKeyUri", v.ExternalProtectionLevelOptions.ExternalKeyUri,
			"ekmConnectionKeyPath", v.ExternalProtectionLevelOptions.EkmConnectionKeyPath,
		)
		if err != nil {
			return nil, err
		}
	}
	return runtime.CreateResource("gcp.project.kmsService.keyring.cryptokey.version",
		"resourcePath", v.Name,
		"name", parseResourceName(v.Name),
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
