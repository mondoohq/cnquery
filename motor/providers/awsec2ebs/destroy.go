package awsec2ebs

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/providers/awsec2ebs/custommount"
)

func (t *Provider) UnmountVolumeFromInstance() error {
	log.Info().Msg("unmount volume")
	if err := custommount.Unmount(t.tmpInfo.scanDir); err != nil {
		log.Error().Err(err).Msg("failed to unmount dir")
		return err
	}
	return nil
}

func (t *Provider) DetachVolumeFromInstance(ctx context.Context, volume *VolumeId) error {
	log.Info().Msg("detach volume")
	res, err := t.scannerRegionEc2svc.DetachVolume(ctx, &ec2.DetachVolumeInput{
		Device: aws.String(t.tmpInfo.volumeAttachmentLoc), VolumeId: &volume.Id,
		InstanceId: &t.scannerInstance.Id,
	})
	if err != nil {
		return err
	}
	if res.State != types.VolumeAttachmentStateDetached { // check if it's detached already
		var volState types.VolumeState
		for volState != types.VolumeStateAvailable {
			time.Sleep(10 * time.Second)
			resp, err := t.scannerRegionEc2svc.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{VolumeIds: []string{volume.Id}})
			if err != nil {
				return err
			}
			if len(resp.Volumes) == 1 {
				volState = resp.Volumes[0].State
			}
			log.Info().Interface("state", volState).Msg("waiting for volume detachment completion")
		}
	}
	return nil
}

func (t *Provider) DeleteCreatedVolume(ctx context.Context, volume *VolumeId) error {
	log.Info().Msg("delete created volume")
	_, err := t.scannerRegionEc2svc.DeleteVolume(ctx, &ec2.DeleteVolumeInput{VolumeId: &volume.Id})
	return err
}

func (t *Provider) RemoveCreatedDir() error {
	log.Info().Msg("remove created dir")
	return os.RemoveAll(t.tmpInfo.scanDir)
}
