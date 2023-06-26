package awsec2ebs

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/rs/zerolog/log"
)

func (p *Provider) DetachVolumeFromInstance(ctx context.Context, volume *VolumeInfo) error {
	log.Info().Msg("detach volume")
	res, err := p.scannerRegionEc2svc.DetachVolume(ctx, &ec2.DetachVolumeInput{
		Device: aws.String(p.volumeMounter.VolumeAttachmentLoc), VolumeId: &volume.Id,
		InstanceId: &p.scannerInstance.Id,
	})
	if err != nil {
		return err
	}
	if res.State != types.VolumeAttachmentStateDetached { // check if it's detached already
		var volState types.VolumeState
		for volState != types.VolumeStateAvailable {
			time.Sleep(10 * time.Second)
			resp, err := p.scannerRegionEc2svc.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{VolumeIds: []string{volume.Id}})
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

func (p *Provider) DeleteCreatedVolume(ctx context.Context, volume *VolumeInfo) error {
	log.Info().Msg("delete created volume")
	_, err := p.scannerRegionEc2svc.DeleteVolume(ctx, &ec2.DeleteVolumeInput{VolumeId: &volume.Id})
	return err
}
