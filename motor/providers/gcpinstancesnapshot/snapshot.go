package gcpinstancesnapshot

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/motorid/gce"
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
	PlatformMrn    string
	ProjectID      string
	Zone           string
	InstanceName   string
	BootDiskSource string
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
	ii.Zone = instance.Zone
	ii.InstanceName = instance.Name
	ii.PlatformMrn = gce.MondooGcpInstancePlatformMrn(projectID, instance.Zone, instance.Name)

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
		ii.BootDiskSource = bootDisk.Source
	}

	return ii, nil
}

// cloneDisk clones a provided disk
func (sc *SnapshotCreator) cloneDisk(sourceDisk, projectID, zone, diskName string) (string, error) {
	var clonedDiskUrl string
	ctx := context.Background()

	computeService, err := sc.computeServiceClient(ctx)
	if err != nil {
		return "", err
	}

	// create a new disk clone
	disk := &compute.Disk{
		Name:       diskName,
		SourceDisk: sourceDisk,
		Labels:     sc.labels,
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
				return clonedDiskUrl, fmt.Errorf("operation failed: %+v", operation.Error.Errors)
			}
			clonedDiskUrl = operation.TargetLink
			break
		}
	}

	return clonedDiskUrl, nil
}

// attachDisk attaches a disk to an instanc
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
				return fmt.Errorf("operation failed: %+v", operation.Error.Errors)
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

	// attach the disk to the instance
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
				return fmt.Errorf("operation failed: %+v", operation.Error.Errors)
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
		log.Info().Str("disk", diskName).Msg("deleted temporary disk created by cnspec")
	} else {
		log.Debug().Str("disk", diskName).Msg("skipping disk deletion, not created by cnspec")
	}

	return nil
}
