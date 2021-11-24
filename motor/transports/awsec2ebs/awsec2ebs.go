package awsec2ebs

import (
	"context"
	"math/rand"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fs"
	"go.mondoo.io/mondoo/motor/transports/shared"
)

var _ transports.Transport = (*Ec2EbsTransport)(nil)
var _ transports.TransportPlatformIdentifier = (*Ec2EbsTransport)(nil)

func New(tc *transports.TransportConfig) (*Ec2EbsTransport, error) {
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
		return nil, errors.Wrap(err, "could not load instance info: aws-ec2-ebs transport only valid on ec2 instances")
	}
	cfg.Region = i.Region
	svc := ec2.NewFromConfig(cfg)

	shell := []string{"sh", "-c"}

	ti := &InstanceId{
		Account: tc.Options["account"],
		Region:  tc.Options["region"],
		Id:      tc.Options["id"],
	}

	// 2. create transport
	t := &Ec2EbsTransport{
		config: cfg,
		opts:   tc.Options,
		target: targetInfo{instance: ti, platformId: tc.PlatformId},
		scannerInstance: &InstanceId{
			Id:      i.InstanceID,
			Region:  i.Region,
			Account: i.AccountID,
			Zone:    i.AvailabilityZone,
		},
		scannerRegionEc2svc: svc,
		shell:               shell,
	}

	// 3. setup
	ok, err := t.Setup()
	if err != nil {
		return t, err
	}
	if !ok {
		return t, errors.New("something went wrong; unable to complete setup for ebs volume scan")
	}

	// 4. mount
	err = t.Mount()
	if err != nil {
		t.Close()
		return t, err
	}

	// 5. create and initialize fs transport (nested transport)
	fsTransport, err := fs.NewWithClose(&transports.TransportConfig{
		Path:       t.tmpInfo.scanDir,
		Backend:    transports.TransportBackend_CONNECTION_FS,
		PlatformId: tc.PlatformId,
		Options:    tc.Options,
	}, t.Close)
	if err != nil {
		return nil, err
	}
	t.FsTransport = fsTransport
	return t, nil
}

type Ec2EbsTransport struct {
	FsTransport         *fs.FsTransport
	scannerRegionEc2svc *ec2.Client
	config              aws.Config
	opts                map[string]string
	shell               []string
	scannerInstance     *InstanceId // the instance the transport is running on
	tmpInfo             tmpInfo
	target              targetInfo // info about the target instance
}

type targetInfo struct {
	platformId string
	instance   *InstanceId
}

type tmpInfo struct {
	// these fields are referenced during setup/mount and close
	scanVolumeId        *VolumeId // the volume id of the volume we attached to the instance
	scanDir             string    // the tmp dir we create; serves as the directory we mount the volume to
	volumeAttachmentLoc string    // where we tell AWS to attach the volume; it doesn't necessarily get attached there, but we have to reference this same location when detaching
}

func (t *Ec2EbsTransport) RunCommand(command string) (*transports.Command, error) {
	c := shared.Command{Shell: t.shell}
	args := []string{}

	res, err := c.Exec(command, args)
	return res, err
}

func (t *Ec2EbsTransport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("FileInfo not implemented")
}

func (t *Ec2EbsTransport) FS() afero.Fs {
	return t.FsTransport.FS()
}

func (t *Ec2EbsTransport) Close() {
	ctx := context.Background()
	err := t.UnmountVolumeFromInstance()
	if err != nil {
		log.Error().Err(err).Msg("unable to unmount volume")
	}
	err = t.DetachVolumeFromInstance(ctx, t.tmpInfo.scanVolumeId)
	if err != nil {
		log.Error().Err(err).Msg("unable to detach volume")
	}
	err = t.DeleteCreatedVolume(ctx, t.tmpInfo.scanVolumeId)
	if err != nil {
		log.Error().Err(err).Msg("unable to delete volume")
	}
	err = t.RemoveCreatedDir()
	if err != nil {
		log.Error().Err(err).Msg("unable to remove dir")
	}
}

func (t *Ec2EbsTransport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Capability_Aws_Ebs,
	}
}

func (t *Ec2EbsTransport) Kind() transports.Kind {
	return transports.Kind_KIND_API
}

func (t *Ec2EbsTransport) Runtime() string {
	return transports.RUNTIME_AWS_EC2_EBS
}

func (t *Ec2EbsTransport) PlatformIdDetectors() []transports.PlatformIdDetector {
	return []transports.PlatformIdDetector{
		transports.TransportPlatformIdentifierDetector,
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

func (t *Ec2EbsTransport) Identifier() (string, error) {
	return t.target.platformId, nil
}
