package awsec2ebs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

func (t *Ec2EbsTransport) UnmountVolumeFromInstance() error {
	log.Info().Msg("unmount volume")
	if err := unix.Unmount(ScanDir, unix.MNT_DETACH); err != nil && err != unix.EBUSY { // does not compile on mac bc mount is not implemented for darwin
		log.Error().Err(err).Msg("failed to unmount dir")
		return err
	}

	return nil
}

func (t *Ec2EbsTransport) DetachVolumeFromInstance(ctx context.Context, volume *VolumeId) error {
	log.Info().Msg("detach volume")
	_, err := t.ec2svc.DetachVolume(ctx, &ec2.DetachVolumeInput{
		Device: aws.String(mountDir), VolumeId: &volume.Id,
		InstanceId: &t.scannerInstance.Id,
	})
	if err != nil {
		return err
	}
	return nil
}

// todo: remove created volume
