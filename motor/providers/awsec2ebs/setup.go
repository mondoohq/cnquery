package awsec2ebs

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	motoraws "go.mondoo.com/cnquery/motor/discovery/aws"
)

func (t *Provider) Validate(ctx context.Context) (*types.Instance, *VolumeId, *SnapshotId, error) {
	target := t.target
	switch t.targetType {
	case EBSTargetInstance:
		log.Info().Interface("instance", target).Msg("validate state")
		resp, err := t.targetRegionEc2svc.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{target.Id}})
		if err != nil {
			return nil, nil, nil, err
		}
		if !motoraws.InstanceIsInRunningOrStoppedState(resp.Reservations[0].Instances[0].State) {
			return nil, nil, nil, errors.New("instance must be in running or stopped state")
		}
		return &resp.Reservations[0].Instances[0], nil, nil, nil
	case EBSTargetVolume:
		log.Info().Interface("volume", target).Msg("validate exists")
		vols, err := t.targetRegionEc2svc.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{VolumeIds: []string{target.Id}})
		if err != nil {
			return nil, nil, nil, err
		}
		if len(vols.Volumes) > 0 {
			vol := vols.Volumes[0]
			if vol.State != types.VolumeStateAvailable {
				// we can still scan it, it just means we have to do the whole snapshot/create volume dance
				log.Warn().Msg("volume specified is not in available state")
				return nil, &VolumeId{Id: t.target.Id, Account: t.target.AccountId, Region: t.target.Region, IsAvailable: false}, nil, nil
			}
			return nil, &VolumeId{Id: t.target.Id, Account: t.target.AccountId, Region: t.target.Region, IsAvailable: true}, nil, nil
		}
	case EBSTargetSnapshot:
		log.Info().Interface("snapshot", target).Msg("validate exists")
		snaps, err := t.targetRegionEc2svc.DescribeSnapshots(ctx, &ec2.DescribeSnapshotsInput{SnapshotIds: []string{target.Id}})
		if err != nil {
			return nil, nil, nil, err
		}
		if len(snaps.Snapshots) > 0 {
			return nil, nil, &SnapshotId{Id: t.target.Id, Account: t.target.AccountId, Region: t.target.Region}, nil
		}
	default:
		return nil, nil, nil, errors.New("cannot validate; unrecognized ebs target")
	}
	return nil, nil, nil, errors.New("cannot validate; unrecognized ebs target")
}

func (t *Provider) SetupForTargetVolume(ctx context.Context, volume VolumeId) (bool, error) {
	log.Debug().Interface("volume", volume).Msg("setup for target volume")
	if !volume.IsAvailable {
		return t.SetupForTargetVolumeUnavailable(ctx, volume)
	}
	t.tmpInfo.scanVolumeId = &volume
	return t.AttachVolumeToInstance(ctx, volume)
}

func (t *Provider) SetupForTargetVolumeUnavailable(ctx context.Context, volume VolumeId) (bool, error) {
	found, snapId, err := t.FindRecentSnapshotForVolume(ctx, volume)
	if err != nil {
		// only log the error here, this is not a blocker
		log.Error().Err(err).Msg("unable to find recent snapshot for volume")
	}
	if !found {
		snapId, err = t.CreateSnapshotFromVolume(ctx, volume)
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

func (t *Provider) SetupForTargetSnapshot(ctx context.Context, snapshot SnapshotId) (bool, error) {
	log.Debug().Interface("snapshot", snapshot).Msg("setup for target snapshot")
	snapId, err := t.CopySnapshotToRegion(ctx, snapshot)
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

func (t *Provider) SetupForTargetInstance(ctx context.Context, instanceinfo *types.Instance) (bool, error) {
	log.Debug().Str("instance id", *instanceinfo.InstanceId).Msg("setup for target instance")
	var err error
	v, err := t.GetVolumeIdForInstance(ctx, instanceinfo)
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

func (t *Provider) GetVolumeIdForInstance(ctx context.Context, instanceinfo *types.Instance) (VolumeId, error) {
	i := t.target
	log.Info().Interface("instance", i).Msg("find volume id")

	if volID := GetVolumeIdForInstance(instanceinfo); volID != nil {
		return VolumeId{Id: *volID, Region: i.Region, Account: i.AccountId}, nil
	}
	return VolumeId{}, errors.New("no volume id found for instance")
}

func GetVolumeIdForInstance(instanceinfo *types.Instance) *string {
	if len(instanceinfo.BlockDeviceMappings) == 1 {
		return instanceinfo.BlockDeviceMappings[0].Ebs.VolumeId
	}
	if len(instanceinfo.BlockDeviceMappings) > 1 {
		for bi := range instanceinfo.BlockDeviceMappings {
			log.Info().Interface("device", *instanceinfo.BlockDeviceMappings[bi].DeviceName).Msg("found instance block devices")
			// todo: revisit this. this works for the standard ec2 instance setup, but no guarantees outside of that..
			if strings.Contains(*instanceinfo.BlockDeviceMappings[bi].DeviceName, "xvda") { // xvda is the root volume
				return instanceinfo.BlockDeviceMappings[bi].Ebs.VolumeId
			}
			if strings.Contains(*instanceinfo.BlockDeviceMappings[bi].DeviceName, "sda1") {
				return instanceinfo.BlockDeviceMappings[bi].Ebs.VolumeId
			}
		}
	}
	return nil
}

func (t *Provider) FindRecentSnapshotForVolume(ctx context.Context, v VolumeId) (bool, SnapshotId, error) {
	log.Info().Msg("find recent snapshot")
	res, err := t.scannerRegionEc2svc.DescribeSnapshots(ctx,
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
					var ae smithy.APIError
					if errors.As(err, &ae) {
						if ae.ErrorCode() == "InvalidSnapshot.NotFound" {
							return false, SnapshotId{}, nil
						}
					}
					return false, SnapshotId{}, err
				}
				snapState = snaps.Snapshots[0].State
			}
			return true, s, nil
		}
	}
	return false, SnapshotId{}, nil
}

func (t *Provider) CreateSnapshotFromVolume(ctx context.Context, v VolumeId) (SnapshotId, error) {
	log.Info().Msg("create snapshot")
	// snapshot the volume
	// use region from volume for aws config
	cfgCopy := t.config.Copy()
	cfgCopy.Region = v.Region
	snapId, err := CreateSnapshotFromVolume(ctx, cfgCopy, v.Id, resourceTags(types.ResourceTypeSnapshot, t.target.Id))
	if err != nil {
		return SnapshotId{}, err
	}

	return SnapshotId{Id: *snapId, Region: v.Region, Account: v.Account}, nil
}

func CreateSnapshotFromVolume(ctx context.Context, cfg aws.Config, volID string, tags []types.TagSpecification) (*string, error) {
	ec2svc := ec2.NewFromConfig(cfg)
	res, err := ec2svc.CreateSnapshot(ctx, &ec2.CreateSnapshotInput{VolumeId: &volID, TagSpecifications: tags})
	if err != nil {
		return nil, err
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
			return nil, err
		}
		snapProgress = *snaps.Snapshots[0].Progress
	}
	return res.SnapshotId, nil
}

func (t *Provider) CopySnapshotToRegion(ctx context.Context, snapshot SnapshotId) (SnapshotId, error) {
	log.Info().Str("snapshot", snapshot.Region).Str("scanner instance", t.scannerInstance.Region).Msg("checking snapshot region")
	if snapshot.Region == t.scannerInstance.Region {
		// we only need to copy the snapshot to the scanner region if it is not already in the same region
		return snapshot, nil
	}
	var newSnapshot SnapshotId
	log.Info().Msg("copy snapshot")
	// snapshot the volume
	res, err := t.scannerRegionEc2svc.CopySnapshot(ctx, &ec2.CopySnapshotInput{SourceRegion: &snapshot.Region, SourceSnapshotId: &snapshot.Id, TagSpecifications: resourceTags(types.ResourceTypeSnapshot, t.target.Id)})
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

func (t *Provider) CreateVolumeFromSnapshot(ctx context.Context, snapshot SnapshotId) (VolumeId, error) {
	log.Info().Msg("create volume")
	var vol VolumeId

	out, err := t.scannerRegionEc2svc.CreateVolume(ctx, &ec2.CreateVolumeInput{
		SnapshotId:        &snapshot.Id,
		AvailabilityZone:  &t.scannerInstance.Zone,
		TagSpecifications: resourceTags(types.ResourceTypeVolume, t.target.Id),
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
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/device_naming.html
	return "/dev/sd" + string(c)
}

func AttachVolume(ctx context.Context, ec2svc *ec2.Client, location string, volID string, instanceID string) (string, types.VolumeAttachmentState, error) {
	res, err := ec2svc.AttachVolume(ctx, &ec2.AttachVolumeInput{
		Device: aws.String(location), VolumeId: &volID,
		InstanceId: &instanceID,
	})
	if err != nil {
		log.Error().Err(err).Str("volume", volID).Msg("attach volume err")
		var ae smithy.APIError
		if errors.As(err, &ae) {
			if ae.ErrorCode() != "InvalidParameterValue" {
				// we don't want to return the err if it's invalid parameter value
				return location, "", err
			}
		}
		// if invalid, it could be something else is using that space, try to mount to diff location
		newlocation := newVolumeAttachmentLoc()
		if location != newlocation {
			location = newlocation
		} else {
			location = newVolumeAttachmentLoc() // we shouldn't have gotten the same one the first go round, but it is randomized, so there is a possibility. try again in that case.
		}
		res, err = ec2svc.AttachVolume(ctx, &ec2.AttachVolumeInput{
			Device: aws.String(location), VolumeId: &volID, // warning: there is no guarantee that aws will place the volume at this location
			InstanceId: &instanceID,
		})
		if err != nil {
			log.Error().Err(err).Str("volume", volID).Msg("attach volume err")
			return location, "", err
		}
	}
	if res.Device != nil {
		log.Debug().Str("location", *res.Device).Msg("attached volume")
		location = *res.Device
	}
	return location, res.State, nil
}

func (t *Provider) AttachVolumeToInstance(ctx context.Context, volume VolumeId) (bool, error) {
	log.Info().Str("volume id", volume.Id).Msg("attach volume")
	t.tmpInfo.volumeAttachmentLoc = newVolumeAttachmentLoc()
	ready := false
	location, state, err := AttachVolume(ctx, t.scannerRegionEc2svc, newVolumeAttachmentLoc(), volume.Id, t.scannerInstance.Id)
	if err != nil {
		return ready, err
	}
	t.tmpInfo.volumeAttachmentLoc = location // warning: there is no guarantee from AWS that the device will be placed therev
	log.Debug().Str("location", location).Msg("target volume")

	/*
		NOTE: re: encrypted volumes
		Encrypted EBS volumes must be attached
		to instances that support Amazon EBS encryption: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/EBSEncryption.html
	*/

	// here we have the attachment state
	if state != types.VolumeAttachmentStateAttached {
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
