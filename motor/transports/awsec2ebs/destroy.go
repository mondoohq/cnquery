package awsec2ebs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/transports/awsec2ebs/custommount"
)

func (t *Ec2EbsTransport) UnmountVolumeFromInstance() error {
	log.Info().Msg("unmount volume")
	if err := custommount.Unmount(ScanDir); err != nil {
		log.Error().Err(err).Msg("failed to unmount dir")
		return err
	}
	return nil
}

func (t *Ec2EbsTransport) DetachVolumeFromInstance(ctx context.Context, volume *VolumeId) error {
	log.Info().Msg("detach volume")
	_, err := t.scannerRegionEc2svc.DetachVolume(ctx, &ec2.DetachVolumeInput{
		Device: aws.String(mountDir), VolumeId: &volume.Id,
		InstanceId: &t.scannerInstance.Id,
	})
	if err != nil {
		return err
	}
	return nil
}

func (t *Ec2EbsTransport) DeleteCreatedVolume(ctx context.Context, volume *VolumeId) error {
	log.Info().Msg("delete created volume")
	_, err := t.scannerRegionEc2svc.DeleteVolume(ctx, &ec2.DeleteVolumeInput{VolumeId: &volume.Id})
	return err
}
