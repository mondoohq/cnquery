package awsec2ebs

import (
	"context"
	"math/rand"
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
	v, err := t.GetVolumeIdForInstance(ctx, t.target.instance)
	if err != nil {
		return false, err
	}
	found, snapId, err := t.FindRecentSnapshotForVolume(ctx, v)
	if err != nil {
		// only log the error here, this is not a blocker
		log.Error().Err(err).Msg("unable to find recent snapshot for volume")
	}
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
	t.tmpInfo.scanVolumeId = &volId
	return t.AttachVolumeToInstance(ctx, volId)
}

func (t *Ec2EbsTransport) GetVolumeIdForInstance(ctx context.Context, i *InstanceId) (VolumeId, error) {
	log.Info().Interface("instance", i).Msg("find volume id")
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
			if len(resp.Reservations[0].Instances[0].BlockDeviceMappings) == 1 {
				volId := resp.Reservations[0].Instances[0].BlockDeviceMappings[0].Ebs.VolumeId
				return VolumeId{Id: *volId, Region: i.Region, Account: i.Account}, nil
			}
		}
	}
	return VolumeId{}, errors.New("no volume id found for instance")
}

func (t *Ec2EbsTransport) FindRecentSnapshotForVolume(ctx context.Context, v VolumeId) (bool, SnapshotId, error) {
	log.Info().Msg("find recent snapshot")
	// todo: use mql query for this
	// aws.ec2.snapshots.where(volumeId == v.Id) { id startTime < time.now - 8*time.hour}
	// return true, SnapshotId{Id: "snap-06bdec45af7e648c5", Region: "us-east-1", Account: "185972265011"} // i-09aafd4819fc62703
	// checking in scanner region
	cfgCopy := t.config.Copy()
	cfgCopy.Region = v.Region
	ec2svc := ec2.NewFromConfig(cfgCopy)
	res, err := ec2svc.DescribeSnapshots(ctx,
		&ec2.DescribeSnapshotsInput{Filters: []types.Filter{
			{Name: aws.String("volume-id"), Values: []string{v.Id}},
		}})
	if err != nil {
		return false, SnapshotId{}, err
	}

	eighthrsago := time.Now().Add(-8 * time.Hour)
	for i := range res.Snapshots {
		// check the start time on all the snapshots
		snapshot := res.Snapshots[i]
		if snapshot.StartTime.After(eighthrsago) {
			s := SnapshotId{Account: v.Account, Region: v.Region, Id: *snapshot.SnapshotId}
			log.Info().Interface("snapshot", s).Msg("found snapshot")
			snapState := snapshot.State
			for snapState != types.SnapshotStateCompleted {
				log.Info().Interface("state", snapState).Msg("waiting for snapshot copy completion; sleeping 10 seconds")
				time.Sleep(10 * time.Second)
				snaps, err := t.scannerRegionEc2svc.DescribeSnapshots(ctx, &ec2.DescribeSnapshotsInput{SnapshotIds: []string{s.Id}})
				if err != nil {
					return false, SnapshotId{}, err
				}
				snapState = snaps.Snapshots[0].State
			}
			return true, s, nil
		}
	}
	return false, SnapshotId{}, nil
}

func (t *Ec2EbsTransport) CreateSnapshotFromVolume(ctx context.Context, v VolumeId) (SnapshotId, error) {
	log.Info().Msg("create snapshot")
	// snapshot the volume
	// use region from volume for aws config
	cfgCopy := t.config.Copy()
	cfgCopy.Region = v.Region
	ec2svc := ec2.NewFromConfig(cfgCopy)
	res, err := ec2svc.CreateSnapshot(ctx, &ec2.CreateSnapshotInput{VolumeId: &v.Id, TagSpecifications: resourceTags(types.ResourceTypeSnapshot, t.target.instance.Id)})
	if err != nil {
		return SnapshotId{}, err
	}

	/*
		NOTE re: encrypted snapshots
		Snapshots that are taken from encrypted volumes are
		automatically encrypted/decrypted. Volumes that are created from encrypted snapshots are
		also automatically encrypted/decrypted.
	*/

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
	log.Info().Str("snapshot", snapshot.Region).Str("scanner instance", t.scannerInstance.Region).Msg("checking snapshot region")
	if snapshot.Region == t.scannerInstance.Region {
		// we only need to copy the snapshot to the scanner region if it is not already in the same region
		return snapshot, nil
	}
	var newSnapshot SnapshotId
	log.Info().Msg("copy snapshot")
	// snapshot the volume
	res, err := t.scannerRegionEc2svc.CopySnapshot(ctx, &ec2.CopySnapshotInput{SourceRegion: &snapshot.Region, SourceSnapshotId: &snapshot.Id, TagSpecifications: resourceTags(types.ResourceTypeSnapshot, t.target.instance.Id)})
	if err != nil {
		return newSnapshot, err
	}

	// wait for snapshot to be ready
	snaps, err := t.scannerRegionEc2svc.DescribeSnapshots(ctx, &ec2.DescribeSnapshotsInput{SnapshotIds: []string{*res.SnapshotId}})
	if err != nil {
		return newSnapshot, err
	}
	snapState := snaps.Snapshots[0].State
	for snapState != types.SnapshotStateCompleted {
		log.Info().Interface("state", snapState).Msg("waiting for snapshot copy completion; sleeping 10 seconds")
		time.Sleep(10 * time.Second)
		snaps, err := t.scannerRegionEc2svc.DescribeSnapshots(ctx, &ec2.DescribeSnapshotsInput{SnapshotIds: []string{*res.SnapshotId}})
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

	out, err := t.scannerRegionEc2svc.CreateVolume(ctx, &ec2.CreateVolumeInput{
		SnapshotId:        &snapshot.Id,
		AvailabilityZone:  &t.scannerInstance.Zone,
		TagSpecifications: resourceTags(types.ResourceTypeVolume, t.target.instance.Id),
	})
	if err != nil {
		return vol, err
	}

	/*
		NOTE re: encrypted snapshots
		Snapshots that are taken from encrypted volumes are
		automatically encrypted/decrypted. Volumes that are created from encrypted snapshots are
		also automatically encrypted/decrypted.
	*/

	state := out.State
	for state != types.VolumeStateAvailable {
		log.Info().Interface("state", state).Msg("waiting for volume creation completion; sleeping 10 seconds")
		time.Sleep(10 * time.Second)
		vols, err := t.scannerRegionEc2svc.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{VolumeIds: []string{*out.VolumeId}})
		if err != nil {
			return vol, err
		}
		state = vols.Volumes[0].State
	}
	return VolumeId{Id: *out.VolumeId, Region: t.config.Region, Account: t.scannerInstance.Account}, nil
}

func newVolumeAttachmentLoc() string {
	chars := []rune("bcdefghijklmnopqrstuvwxyz") // a is reserved for the root volume
	randomIndex := rand.Intn(len(chars))
	c := chars[randomIndex]
	return "/dev/xvd" + string(c)
}

func (t *Ec2EbsTransport) AttachVolumeToInstance(ctx context.Context, volume VolumeId) (bool, error) {
	log.Info().Msg("attach volume")
	t.tmpInfo.volumeAttachmentLoc = newVolumeAttachmentLoc()
	ready := false
	res, err := t.scannerRegionEc2svc.AttachVolume(ctx, &ec2.AttachVolumeInput{
		Device: aws.String(t.tmpInfo.volumeAttachmentLoc), VolumeId: &volume.Id,
		InstanceId: &t.scannerInstance.Id,
	})
	if err != nil {
		return ready, err
	}

	/*
		NOTE: re: encrypted volumes
		Encrypted EBS volumes must be attached
		to instances that support Amazon EBS encryption: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/EBSEncryption.html
	*/

	// here we have the attachment state
	if res.State != types.VolumeAttachmentStateAttached {
		var volState types.VolumeState
		for volState != types.VolumeStateInUse {
			time.Sleep(10 * time.Second)
			resp, err := t.scannerRegionEc2svc.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{VolumeIds: []string{volume.Id}})
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
