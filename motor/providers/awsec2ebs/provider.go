package awsec2ebs

import (
	"context"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/fs"
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/os/snapshot"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
	_ os.OperatingSystemProvider   = (*Provider)(nil)
)

// New creates a new aws-ec2-ebs provider
// It expects to be running on an ec2 instance with ssm iam role and
// permissions for copy snapshot, create snapshot, create volume, attach volume, detach volume
// TODO: validate the expected permissions here
func New(pCfg *providers.Config) (*Provider, error) {
	rand.Seed(time.Now().UnixNano())

	// 1. validate; load
	// TODO allow custom aws config
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "could not load aws configuration")
	}
	i, err := RawInstanceInfo(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "could not load instance info: aws-ec2-ebs provider only valid on ec2 instances")
	}

	// ec2 client for the scanner region
	cfg.Region = i.Region
	scannerSvc := ec2.NewFromConfig(cfg)

	// ec2 client for the target region
	cfgCopy := cfg.Copy()
	cfgCopy.Region = pCfg.Options["region"]
	targetSvc := ec2.NewFromConfig(cfgCopy)

	shell := []string{"sh", "-c"}
	volumeMounter := snapshot.NewVolumeMounter(shell)

	// 2. create provider instance
	p := &Provider{
		config: cfg,
		opts:   pCfg.Options,
		target: TargetInfo{
			PlatformId: pCfg.PlatformId,
			AccountId:  pCfg.Options["account"],
			Region:     pCfg.Options["region"],
			Id:         pCfg.Options["id"],
		},
		targetType: pCfg.Options["type"],
		scannerInstance: &InstanceId{
			Id:      i.InstanceID,
			Region:  i.Region,
			Account: i.AccountID,
			Zone:    i.AvailabilityZone,
		},
		targetRegionEc2svc:  targetSvc,
		scannerRegionEc2svc: scannerSvc,
		volumeMounter:       volumeMounter,
	}
	log.Debug().Interface("info", p.target).Str("type", p.targetType).Msg("target")

	ctx := context.Background()

	// 3. validate
	instanceinfo, volumeid, snapshotid, err := p.Validate(ctx)
	if err != nil {
		return p, errors.Wrap(err, "unable to validate")
	}

	// 4. setup the volume for scanning
	// check if we got the no setup override option. this implies the target volume is already attached to the instance
	// this is used in cases where we need to test a snapshot created from a public marketplace image. the volume gets attached to a brand
	// new instance, and then that instance is started and we scan the attached fs
	if pCfg.Options[snapshot.NoSetup] == "true" {
		log.Info().Msg("skipping setup step")
	} else {
		var ok bool
		var err error
		switch p.targetType {
		case EBSTargetInstance:
			ok, err = p.SetupForTargetInstance(ctx, instanceinfo)
		case EBSTargetVolume:
			ok, err = p.SetupForTargetVolume(ctx, *volumeid)
		case EBSTargetSnapshot:
			ok, err = p.SetupForTargetSnapshot(ctx, *snapshotid)
		default:
			return p, errors.New("invalid target type")
		}
		if err != nil {
			log.Error().Err(err).Msg("unable to complete setup step")
			p.Close()
			return p, err
		}
		if !ok {
			return p, errors.New("something went wrong; unable to complete setup for ebs volume scan")
		}
	}

	// Mount Volume
	err = p.volumeMounter.Mount()
	if err != nil {
		log.Error().Err(err).Msg("unable to complete mount step")
		p.Close()
		return p, err
	}

	// Create and initialize fs provider
	fsProvider, err := fs.NewWithClose(&providers.Config{
		Path:       p.volumeMounter.ScanDir,
		Backend:    providers.ProviderType_FS,
		PlatformId: pCfg.PlatformId,
		Options:    pCfg.Options,
	}, p.Close)
	if err != nil {
		return nil, err
	}
	p.FsProvider = fsProvider
	return p, nil
}

type Provider struct {
	FsProvider          *fs.Provider
	scannerRegionEc2svc *ec2.Client
	targetRegionEc2svc  *ec2.Client
	config              aws.Config
	opts                map[string]string
	scannerInstance     *InstanceId // the instance the transport is running on
	scanVolumeInfo      *VolumeInfo // the info of the volume we attached to the instance
	target              TargetInfo  // info about the target
	targetType          string      // the type of object we're targeting (instance, volume, snapshot)
	volumeMounter       *snapshot.VolumeMounter
}

type TargetInfo struct {
	PlatformId string
	AccountId  string
	Region     string
	Id         string
}

func (p *Provider) RunCommand(command string) (*os.Command, error) {
	return nil, errors.New("RunCommand not implemented")
}

func (p *Provider) FileInfo(path string) (os.FileInfoDetails, error) {
	return os.FileInfoDetails{}, errors.New("FileInfo not implemented")
}

func (p *Provider) FS() afero.Fs {
	return p.FsProvider.FS()
}

func (p *Provider) Close() {
	if p.opts != nil {
		if p.opts[snapshot.NoSetup] == "true" {
			return
		}
	}
	ctx := context.Background()
	err := p.volumeMounter.UnmountVolumeFromInstance()
	if err != nil {
		log.Error().Err(err).Msg("unable to unmount volume")
	}
	err = p.DetachVolumeFromInstance(ctx, p.scanVolumeInfo)
	if err != nil {
		log.Error().Err(err).Msg("unable to detach volume")
	}
	// only delete the volume if we created it, e.g., if we're scanning a snapshot
	if val, ok := p.scanVolumeInfo.Tags["createdBy"]; ok {
		if val == "Mondoo" {
			err = p.DeleteCreatedVolume(ctx, p.scanVolumeInfo)
			if err != nil {
				log.Error().Err(err).Msg("unable to delete volume")
			}
			log.Info().Str("vol-id", p.scanVolumeInfo.Id).Msg("deleted temporary volume created by Mondoo")
		}
	} else {
		log.Debug().Str("vol-id", p.scanVolumeInfo.Id).Msg("skipping volume deletion, not created by Mondoo")
	}
	err = p.volumeMounter.RemoveTempScanDir()
	if err != nil {
		log.Error().Err(err).Msg("unable to remove dir")
	}
}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_Aws_Ebs,
	}
}

func (p *Provider) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (p *Provider) Runtime() string {
	return providers.RUNTIME_AWS_EC2_EBS
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func RawInstanceInfo(cfg aws.Config) (*imds.InstanceIdentityDocument, error) {
	metadata := imds.NewFromConfig(cfg)
	ctx := context.Background()
	doc, err := metadata.GetInstanceIdentityDocument(ctx, &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		return nil, err
	}
	return &doc.InstanceIdentityDocument, nil
}

func (p *Provider) Identifier() (string, error) {
	return p.target.PlatformId, nil
}

func GetRawInstanceInfo(profile string) (*imds.InstanceIdentityDocument, error) {
	ctx := context.Background()
	var cfg aws.Config
	var err error
	if profile == "" {
		cfg, err = config.LoadDefaultConfig(ctx)
	} else {
		cfg, err = config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile))
	}
	if err != nil {
		return nil, errors.Wrap(err, "could not load aws configuration")
	}
	i, err := RawInstanceInfo(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "could not load instance info: aws-ec2-ebs provider is only valid on ec2 instances")
	}
	return i, nil
}
