package awsec2ebs

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
)

func (t *Ec2EbsTransport) Setup() (bool, error) {
	var err error
	ctx := context.Background()
	v, err := t.GetVolumeIdForInstance(ctx, t.targetInstance)
	if err != nil {
		return false, err
	}
	found, snapId := t.FindRecentSnapshotForVolume(v)
	if !found {
		snapId, err = t.CreateSnapshotFromVolume(ctx, v)
		if err != nil {
			return false, err
		}
	}
	snapId, err = t.CopySnapshotToRegion(ctx, snapId)
	if err != nil {
		return false, err
	}
	volId, err := t.CreateVolumeFromSnapshot(ctx, snapId)
	if err != nil {
		return false, err
	}
	t.scanVolumeId = &volId
	return t.AttachVolumeToInstance(ctx, volId)
}

func (t *Ec2EbsTransport) GetVolumeIdForInstance(ctx context.Context, i *InstanceId) (VolumeId, error) {
	// use region from instance for aws config
	cfgCopy := t.config.Copy()
	cfgCopy.Region = i.Region
	ec2svc := ec2.NewFromConfig(cfgCopy)
	resp, err := ec2svc.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{i.Id}})
	if err != nil {
		return VolumeId{}, err
	}

	if len(resp.Reservations) == 1 {
		if len(resp.Reservations[0].Instances) == 1 {
			volId := resp.Reservations[0].Instances[0].BlockDeviceMappings[0].Ebs.VolumeId
			return VolumeId{Id: *volId, Region: i.Region, Account: i.Account}, nil
		}
	}
	return VolumeId{}, errors.New("no volume id found for instance")
}

func (t *Ec2EbsTransport) FindRecentSnapshotForVolume(v VolumeId) (bool, SnapshotId) {
	// use mql query for this
	// aws.ec2.snapshots.where(volumeId == v.Id) { id startTime < time.now - 8*time.hour}
	// return true, SnapshotId{Id: "snap-06bdec45af7e648c5", Region: "us-east-1", Account: "185972265011"} // i-09aafd4819fc62703
	return false, SnapshotId{}
}

func (t *Ec2EbsTransport) CreateSnapshotFromVolume(ctx context.Context, v VolumeId) (SnapshotId, error) {
	log.Info().Msg("create snapshot")
	// snapshot the volume
	// use region from volume for aws config
	cfgCopy := t.config.Copy()
	cfgCopy.Region = v.Region
	ec2svc := ec2.NewFromConfig(cfgCopy)
	res, err := ec2svc.CreateSnapshot(ctx, &ec2.CreateSnapshotInput{VolumeId: &v.Id})
	if err != nil {
		return SnapshotId{}, err
	}

	// wait for snapshot to be ready
	snapProgress := *res.Progress
	for !strings.Contains(snapProgress, "100") {
		log.Info().Str("progress", snapProgress).Msg("waiting for snapshot completion; sleeping 10 seconds")
		time.Sleep(10 * time.Second)
		snaps, err := ec2svc.DescribeSnapshots(ctx, &ec2.DescribeSnapshotsInput{SnapshotIds: []string{*res.SnapshotId}})
		if err != nil {
			return SnapshotId{}, err
		}
		snapProgress = *snaps.Snapshots[0].Progress
	}
	return SnapshotId{Id: *res.SnapshotId, Region: v.Region, Account: v.Account}, nil
}

func (t *Ec2EbsTransport) CopySnapshotToRegion(ctx context.Context, snapshot SnapshotId) (SnapshotId, error) {
	if snapshot.Region == t.scannerInstance.Region {
		// we only need to copy the snapshot to the scanner region if it is not already in the same region
		return snapshot, nil
	}
	var newSnapshot SnapshotId
	log.Info().Msg("copy snapshot")
	// snapshot the volume
	res, err := t.ec2svc.CopySnapshot(ctx, &ec2.CopySnapshotInput{SourceRegion: &snapshot.Region, SourceSnapshotId: &snapshot.Id})
	if err != nil {
		return newSnapshot, err
	}

	// wait for snapshot to be ready
	snaps, err := t.ec2svc.DescribeSnapshots(ctx, &ec2.DescribeSnapshotsInput{SnapshotIds: []string{*res.SnapshotId}})
	if err != nil {
		return newSnapshot, err
	}
	snapState := snaps.Snapshots[0].State
	for snapState != types.SnapshotStateCompleted {
		log.Info().Interface("state", snapState).Msg("waiting for snapshot copy completion; sleeping 10 seconds")
		time.Sleep(10 * time.Second)
		snaps, err := t.ec2svc.DescribeSnapshots(ctx, &ec2.DescribeSnapshotsInput{SnapshotIds: []string{*res.SnapshotId}})
		if err != nil {
			return newSnapshot, err
		}
		snapState = snaps.Snapshots[0].State
	}
	return SnapshotId{Id: *res.SnapshotId, Region: t.config.Region, Account: t.scannerInstance.Account}, nil
}

func (t *Ec2EbsTransport) CreateVolumeFromSnapshot(ctx context.Context, snapshot SnapshotId) (VolumeId, error) {
	log.Info().Msg("create volume")
	var vol VolumeId

	out, err := t.ec2svc.CreateVolume(ctx, &ec2.CreateVolumeInput{
		SnapshotId:       &snapshot.Id,
		AvailabilityZone: &t.scannerInstance.Zone,
	})
	if err != nil {
		return vol, err
	}
	state := out.State
	for state != types.VolumeStateAvailable {
		log.Info().Interface("state", state).Msg("waiting for volume creation completion; sleeping 10 seconds")
		time.Sleep(10 * time.Second)
		vols, err := t.ec2svc.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{VolumeIds: []string{*out.VolumeId}})
		if err != nil {
			return vol, err
		}
		state = vols.Volumes[0].State
	}
	return VolumeId{Id: *out.VolumeId, Region: t.config.Region, Account: t.scannerInstance.Account}, nil
}

func (t *Ec2EbsTransport) AttachVolumeToInstance(ctx context.Context, volume VolumeId) (bool, error) {
	log.Info().Msg("attach volume")
	ready := false
	res, err := t.ec2svc.AttachVolume(ctx, &ec2.AttachVolumeInput{
		Device: aws.String(mountDir), VolumeId: &volume.Id,
		InstanceId: &t.scannerInstance.Id,
	})
	if err != nil {
		return ready, err
	}
	// here we have the attachment state
	if res.State != types.VolumeAttachmentStateAttached {
		var volState types.VolumeState
		for volState != types.VolumeStateInUse {
			time.Sleep(10 * time.Second)
			resp, err := t.ec2svc.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{VolumeIds: []string{volume.Id}})
			if err != nil {
				return ready, err
			}
			if len(resp.Volumes) == 1 {
				volState = resp.Volumes[0].State
			}
			log.Info().Interface("state", volState).Msg("waiting for volume attachment completion")
		}
	}
	return true, nil
}
