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
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/fs"
	"go.mondoo.io/mondoo/motor/providers/os/cmd"
)

var (
	_ providers.Transport                   = (*Provider)(nil)
	_ providers.TransportPlatformIdentifier = (*Provider)(nil)
)

func New(pCfg *providers.Config) (*Provider, error) {
	rand.Seed(time.Now().UnixNano())

	// get aws config
	// expect to be running on an ec2 instance with ssm iam role
	// && perms for copy snapshot, create snapshot, create volume, attach volume, detach volume
	// todo: validate the expected permissions here

	// 1. validate; load
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

	// 2. create provider instance
	t := &Provider{
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
		shell:               shell,
	}
	log.Debug().Interface("info", t.target).Str("type", t.targetType).Msg("target")

	ctx := context.Background()
	// 3. validate
	instanceinfo, volumeid, snapshotid, err := t.Validate(ctx)
	if err != nil {
		return t, errors.Wrap(err, "unable to validate")
	}

	// 4. setup
	// check if we got the no setup override option. this implies the target volume is already attached to the instance
	// this is used in cases where we need to test a snapshot created from a public marketplace image. the volume gets attached to a brand
	// new instance, and then that instance is started and we scan the attached fs
	if pCfg.Options[NoSetup] == "true" {
		log.Info().Msg("skipping setup step")
	} else {
		var ok bool
		var err error
		switch t.targetType {
		case EBSTargetInstance:
			ok, err = t.SetupForTargetInstance(ctx, instanceinfo)
		case EBSTargetVolume:
			ok, err = t.SetupForTargetVolume(ctx, *volumeid)
		case EBSTargetSnapshot:
			ok, err = t.SetupForTargetSnapshot(ctx, *snapshotid)
		}
		if err != nil {
			log.Error().Err(err).Msg("unable to complete setup step")
			t.Close()
			return t, err
		}
		if !ok {
			return t, errors.New("something went wrong; unable to complete setup for ebs volume scan")
		}
	}

	// 5. mount
	err = t.Mount()
	if err != nil {
		log.Error().Err(err).Msg("unable to complete mount step")
		t.Close()
		return t, err
	}

	// 5. create and initialize fs provider (we nest it)
	fsProvider, err := fs.NewWithClose(&providers.Config{
		Path:       t.tmpInfo.scanDir,
		Backend:    providers.ProviderType_FS,
		PlatformId: pCfg.PlatformId,
		Options:    pCfg.Options,
	}, t.Close)
	if err != nil {
		return nil, err
	}
	t.FsProvider = fsProvider
	return t, nil
}

const NoSetup = "no-setup"

type Provider struct {
	FsProvider          *fs.Provider
	scannerRegionEc2svc *ec2.Client
	targetRegionEc2svc  *ec2.Client
	config              aws.Config
	opts                map[string]string
	shell               []string
	scannerInstance     *InstanceId // the instance the transport is running on
	tmpInfo             tmpInfo
	target              TargetInfo // info about the target
	targetType          string     // the type of object we're targeting (instance, volume, snapshot)
}

type TargetInfo struct {
	PlatformId string
	AccountId  string
	Region     string
	Id         string
}

type tmpInfo struct {
	// these fields are referenced during setup/mount and close
	scanVolumeId        *VolumeId // the volume id of the volume we attached to the instance
	scanDir             string    // the tmp dir we create; serves as the directory we mount the volume to
	volumeAttachmentLoc string    // where we tell AWS to attach the volume; it doesn't necessarily get attached there, but we have to reference this same location when detaching
}

func (p *Provider) RunCommand(command string) (*providers.Command, error) {
	c := cmd.Command{Shell: p.shell}
	args := []string{}

	res, err := c.Exec(command, args)
	return res, err
}

func (p *Provider) FileInfo(path string) (providers.FileInfoDetails, error) {
	return providers.FileInfoDetails{}, errors.New("FileInfo not implemented")
}

func (p *Provider) FS() afero.Fs {
	return p.FsProvider.FS()
}

func (p *Provider) Close() {
	if p.opts != nil {
		if p.opts[NoSetup] == "true" || p.targetType == EBSTargetSnapshot {
			return
		}
	}
	ctx := context.Background()
	err := p.UnmountVolumeFromInstance()
	if err != nil {
		log.Error().Err(err).Msg("unable to unmount volume")
	}
	err = p.DetachVolumeFromInstance(ctx, p.tmpInfo.scanVolumeId)
	if err != nil {
		log.Error().Err(err).Msg("unable to detach volume")
	}
	err = p.DeleteCreatedVolume(ctx, p.tmpInfo.scanVolumeId)
	if err != nil {
		log.Error().Err(err).Msg("unable to delete volume")
	}
	err = p.RemoveCreatedDir()
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
