// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package gcpinstancesnapshot

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers/os/id/gce"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	googleoauth "golang.org/x/oauth2/google"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v2"
	"google.golang.org/api/option"
)

const (
	createdByLabel = "created-by"
	createdValue   = "cnspec"
)

func NewInstanceUrl(projectID, zone, instanceName string) string {
	return fmt.Sprintf(
		"projects/%s/zones/%s/instances/%s", projectID, zone, instanceName,
	)
}

func NewSourceDiskUrl(projectID, zone, diskName string) string {
	return fmt.Sprintf(
		"projects/%s/zones/%s/disks/%s", projectID, zone, diskName,
	)
}

func NewSnapshotCreator() (*SnapshotCreator, error) {
	scope := []string{cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope}
	client, err := googleoauth.DefaultClient(context.Background(), scope...)
	if err != nil {
		return nil, err
	}

	sc := &SnapshotCreator{
		client: client,
		labels: map[string]string{
			createdByLabel: createdValue,
		},
	}
	return sc, nil
}

type SnapshotCreator struct {
	client *http.Client
	labels map[string]string
}

// computeService returns a new Compute Service instance
func (sc *SnapshotCreator) computeServiceClient(ctx context.Context) (*compute.Service, error) {
	return compute.NewService(ctx, option.WithHTTPClient(sc.client))
}

type instanceInfo struct {
	PlatformMrn       string
	ProjectID         string
	Zone              string
	InstanceName      string
	BootDiskSourceURL string
}

func (sc *SnapshotCreator) InstanceInfo(projectID, zone, instanceName string) (instanceInfo, error) {
	ctx := context.Background()
	ii := instanceInfo{}

	computeService, err := sc.computeServiceClient(ctx)
	if err != nil {
		return ii, err
	}

	instance, err := computeService.Instances.Get(projectID, zone, instanceName).Context(ctx).Do()
	if err != nil {
		return ii, err
	}

	ii.ProjectID = projectID
	ii.Zone = zone
	ii.InstanceName = instance.Name
	ii.PlatformMrn = gce.MondooGcpInstancePlatformMrn(projectID, zone, instance.Name)

	// search for boot disk
	var bootDisk *compute.AttachedDisk
	for i := range instance.Disks {
		dsk := instance.Disks[i]
		if dsk.Boot {
			bootDisk = dsk
			break
		}
	}

	if bootDisk != nil {
		ii.BootDiskSourceURL = bootDisk.Source
	}

	return ii, nil
}

type snapshotInfo struct {
	PlatformMrn  string
	ProjectID    string
	SnapshotName string
	SnapshotUrl  string
}

func (sc *SnapshotCreator) SnapshotInfo(projectID, snapshotName string) (snapshotInfo, error) {
	ctx := context.Background()
	si := snapshotInfo{}

	computeService, err := sc.computeServiceClient(ctx)
	if err != nil {
		return si, err
	}

	snapshot, err := computeService.Snapshots.Get(projectID, snapshotName).Context(ctx).Do()
	if err != nil {
		return si, err
	}

	si.ProjectID = projectID
	si.SnapshotName = snapshot.Name
	si.SnapshotUrl = snapshot.SelfLink
	si.PlatformMrn = SnapshotPlatformMrn(projectID, snapshot.Name)

	return si, nil
}

// searchLatestSnapshot looks for the latest available snapshot for the instance
func (sc *SnapshotCreator) searchLatestSnapshot(projectID, sourceDiskUrl string) (string, time.Time, error) {
	ctx := context.Background()
	latestSnapshotTimestamp := time.UnixMilli(0)

	computeService, err := sc.computeServiceClient(ctx)
	if err != nil {
		return "", latestSnapshotTimestamp, err
	}

	var latestSnapshot *compute.Snapshot

	req := computeService.Snapshots.List(projectID)
	if err := req.Pages(ctx, func(page *compute.SnapshotList) error {
		for _, snapshot := range page.Items {
			// we are only interested in disks that are attached to the
			if snapshot.SourceDisk != sourceDiskUrl {
				continue
			}

			// RFC3339 encoded like 2021-02-28T02:31:38.654-08:00
			snapshotCreated, err := time.Parse(time.RFC3339, snapshot.CreationTimestamp)
			if err != nil {
				log.Err(err).Str("snapshot", snapshot.Name).Str("creation-timestamp", snapshot.CreationTimestamp).Msg("snapshot timestamp is not parsable")
				// we ignore snapshots that we cannot parse
				continue
			}

			if latestSnapshotTimestamp.Before(snapshotCreated) {
				latestSnapshot = snapshot
				latestSnapshotTimestamp = snapshotCreated
			}
		}
		return nil
	}); err != nil {
		return "", latestSnapshotTimestamp, err
	}

	if latestSnapshot == nil {
		return "", latestSnapshotTimestamp, status.Error(codes.NotFound, "no snapshot found")
	}

	return latestSnapshot.SelfLink, latestSnapshotTimestamp, nil
}

// createDisk creates a new disk
func (sc *SnapshotCreator) createDisk(disk *compute.Disk, projectID, zone, diskName string) (string, error) {
	var clonedDiskUrl string
	ctx := context.Background()

	computeService, err := sc.computeServiceClient(ctx)
	if err != nil {
		return "", err
	}

	op, err := computeService.Disks.Insert(projectID, zone, disk).Context(ctx).Do()
	if err != nil {
		return clonedDiskUrl, err
	}

	// wait for the disk creation operation to complete
	for {
		operation, err := computeService.ZoneOperations.Get(projectID, zone, op.Name).Context(ctx).Do()
		if err != nil {
			return clonedDiskUrl, err
		}
		if operation.Status == "DONE" {
			if operation.Error != nil {
				errMessage, _ := operation.Error.MarshalJSON()
				log.Debug().Str("error", string(errMessage)).Msg("operation failed")
				if len(operation.Error.Errors) > 0 {
					errMessage = []byte(operation.Error.Errors[0].Message)
				}
				return clonedDiskUrl, fmt.Errorf("create disk failed: %s", errMessage)
			}
			clonedDiskUrl = operation.TargetLink
			break
		}
	}

	return clonedDiskUrl, nil
}

// createSnapshotDisk creates a new disk from a snapshot
func (sc *SnapshotCreator) createSnapshotDisk(snapshotUrl, projectID, zone, diskName string) (string, error) {
	// create a new disk from snapshot
	disk := &compute.Disk{
		Name:           diskName,
		SourceSnapshot: snapshotUrl,
		Labels:         sc.labels,
	}
	return sc.createDisk(disk, projectID, zone, diskName)
}

// cloneDisk clones a provided disk
func (sc *SnapshotCreator) cloneDisk(sourceDisk, projectID, zone, diskName string) (string, error) {
	// create a new disk clone
	disk := &compute.Disk{
		Name:       diskName,
		SourceDisk: sourceDisk,
		Labels:     sc.labels,
	}
	return sc.createDisk(disk, projectID, zone, diskName)
}

// attachDisk attaches a disk to an instance
func (sc *SnapshotCreator) attachDisk(projectID, zone, instanceName, sourceDiskUrl, deviceName string) error {
	ctx := context.Background()

	computeService, err := sc.computeServiceClient(ctx)
	if err != nil {
		return err
	}

	// define the attached disk
	attachedDisk := &compute.AttachedDisk{
		Source:     sourceDiskUrl,
		DeviceName: deviceName,
	}

	// attach the disk to the instance
	op, err := computeService.Instances.AttachDisk(projectID, zone, instanceName, attachedDisk).Context(ctx).Do()
	if err != nil {
		return err
	}

	// wait for the operation to complete
	for {
		operation, err := computeService.ZoneOperations.Get(projectID, zone, op.Name).Context(ctx).Do()
		if err != nil {
			return err
		}
		if operation.Status == "DONE" {
			if operation.Error != nil {
				errMessage, _ := operation.Error.MarshalJSON()
				log.Debug().Str("error", string(errMessage)).Msg("operation failed")
				if len(operation.Error.Errors) > 0 {
					errMessage = []byte(operation.Error.Errors[0].Message)
				}
				return fmt.Errorf("attach disk failed: %s", errMessage)
			}
			break
		}
	}

	return nil
}

func (sc *SnapshotCreator) detachDisk(projectID, zone, instanceName, deviceName string) error {
	ctx := context.Background()
	log.Debug().Str("device-name", deviceName).Msg("detach disk")
	computeService, err := sc.computeServiceClient(ctx)
	if err != nil {
		return err
	}

	// detach the disk from the instance
	op, err := computeService.Instances.DetachDisk(projectID, zone, instanceName, deviceName).Context(ctx).Do()
	if err != nil {
		return err
	}

	// wait for the operation to complete
	for {
		operation, err := computeService.ZoneOperations.Get(projectID, zone, op.Name).Context(ctx).Do()
		if err != nil {
			return err
		}
		if operation.Status == "DONE" {
			if operation.Error != nil {
				errMessage, _ := operation.Error.MarshalJSON()
				log.Debug().Str("error", string(errMessage)).Msg("operation failed")
				if len(operation.Error.Errors) > 0 {
					errMessage = []byte(operation.Error.Errors[0].Message)
				}
				return fmt.Errorf("detach disk failed: %s", errMessage)
			}
			break
		}
	}

	return nil
}

// parseDiskUrl parses a provided GCP Disk URL
func parseDiskUrl(diskURL string) (string, string, string, error) {
	url, err := url.Parse(diskURL)
	if err != nil {
		return "", "", "", err
	}

	// extract the path and split it into components
	pathComponents := strings.Split(url.Path, "/")

	// extract project, zone, and disk names
	projectId := pathComponents[4]
	zone := pathComponents[6]
	disk := pathComponents[8]
	return projectId, zone, disk, nil
}

// deleteCreatedDisk deletes the given disk if it matches the created label
func (sc *SnapshotCreator) deleteCreatedDisk(diskUrl string) error {
	ctx := context.Background()

	computeService, err := sc.computeServiceClient(ctx)
	if err != nil {
		return err
	}

	projectID, zone, diskName, err := parseDiskUrl(diskUrl)
	if err != nil {
		return err
	}

	// attach the disk to the instance
	disk, err := computeService.Disks.Get(projectID, zone, diskName).Context(ctx).Do()
	if err != nil {
		return err
	}

	// only delete the volume if we created it, e.g., if we're scanning a snapshot
	if val, ok := disk.Labels[createdByLabel]; ok && val == createdValue {
		_, err := computeService.Disks.Delete(projectID, zone, diskName).Context(ctx).Do()
		if err != nil {
			return err
		}
		log.Debug().Str("disk", diskName).Msg("deleted temporary disk created by cnspec")
	} else {
		log.Debug().Str("disk", diskName).Msg("skipping disk deletion, not created by cnspec")
	}

	return nil
}
