// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package gcpinstancesnapshot

import (
	"fmt"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/motorid/gce"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/fs"
	"go.mondoo.com/cnquery/motor/providers/local"
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/os/snapshot"
	"go.mondoo.com/cnquery/mrn"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
	_ os.OperatingSystemProvider   = (*Provider)(nil)
)

type scanTarget struct {
	TargetType   string
	ProjectID    string
	Zone         string
	InstanceName string
	SnapshotName string
}

type scannerInstance struct {
	projectID    string
	zone         string
	instanceName string
}

type mountInfo struct {
	deviceName string
	diskUrl    string
}

func determineScannerInstanceInfo() (*scannerInstance, error) {
	localProvider, err := local.New()
	if err != nil {
		return nil, err
	}
	localProviderDetector := detector.New(localProvider)
	pf, err := localProviderDetector.Platform()
	if err != nil {
		return nil, err
	}
	scannerInstanceInfo, err := gce.Resolve(localProvider, pf)
	if err != nil {
		return nil, errors.New("gcp snapshot provider needs to run on a gcp instance")
	}
	identity, err := scannerInstanceInfo.Identify()
	if err != nil {
		return nil, errors.New("gcp snapshot provider needs to run on a gcp instance")
	}
	instanceID := identity.PlatformMrn

	// parse the platform id
	// //platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/project-id/zones/us-central1-a/instances/123456789
	platformMrn, err := mrn.NewMRN(instanceID)
	if err != nil {
		return nil, err
	}
	projectID, err := platformMrn.ResourceID("projects")
	if err != nil {
		return nil, err
	}
	zone, err := platformMrn.ResourceID("zones")
	if err != nil {
		return nil, err
	}
	instanceName, err := platformMrn.ResourceID("instances")
	if err != nil {
		return nil, err
	}

	return &scannerInstance{
		projectID:    projectID,
		zone:         zone,
		instanceName: instanceName,
	}, nil
}

func ParseTarget(pCfg *providers.Config) scanTarget {
	return scanTarget{
		TargetType:   pCfg.Options["type"],
		ProjectID:    pCfg.Options["project-id"],
		Zone:         pCfg.Options["zone"],
		InstanceName: pCfg.Options["instance-name"],
		SnapshotName: pCfg.Options["snapshot-name"],
	}
}

func New(pCfg *providers.Config) (*Provider, error) {
	target := ParseTarget(pCfg)

	// check if we run on a gcp instance
	scanner, err := determineScannerInstanceInfo()
	if err != nil {
		return nil, err
	}

	// determine the target
	sc, err := NewSnapshotCreator()
	if err != nil {
		return nil, err
	}

	// setup disk image so and attach it to the instance
	var diskUrl string
	mi := mountInfo{
		deviceName: "cnspec",
	}
	switch target.TargetType {
	case "instance":
		instanceInfo, err := sc.InstanceInfo(target.ProjectID, target.Zone, target.InstanceName)
		if err != nil {
			return nil, err
		}
		if instanceInfo.BootDiskSourceURL == "" {
			return nil, fmt.Errorf("could not find boot disk for instance %s", target.InstanceName)
		}

		if pCfg.Options["create-snapshot"] != "true" {
			// search for the latest snapshot for this machine
			snapshotUrl, created, err := sc.searchLatestSnapshot(target.ProjectID, instanceInfo.BootDiskSourceURL)
			if status.Code(err) == codes.NotFound {
				// expected behaviour if no snapshot exists, we fall back to cloning the disk
				log.Debug().Msg("no snapshot found, cloning disk from instance")
			} else if err != nil {
				// real error occurred, we abort
				return nil, errors.Wrap(err, "could not search for gcp instance snapshot")
			} else if err == nil && time.Now().Sub(created).Hours() < 8 {
				// use the snapshot if it was created less than 8 hours ago
				log.Debug().Str("snapshot", snapshotUrl).Msg("found latest snapshot")
				diskUrl, err = sc.createSnapshotDisk(snapshotUrl, scanner.projectID, scanner.zone, "cnspec-"+target.InstanceName+"-snapshot-"+time.Now().Format("2006-01-02t15-04-05z00-00"))
				if err != nil {
					log.Error().Err(err).Str("disk", diskUrl).Msg("could not complete snapshot disk creation")
					return nil, errors.Wrap(err, "could not create disk from snapshot")
				}
				log.Debug().Str("disk", diskUrl).Msg("created disk from snapshot")
				mi.diskUrl = diskUrl
			} else {
				log.Debug().Msg("no recent snapshot found, cloning disk from instance")
			}
		}

		// if no disk was defined or found, clone the disk attached to the instance
		if mi.diskUrl == "" {
			// clone the disk of the instance to the zone where the scanner runs
			// disk name does not allow colons, therefore we need a custom format
			diskUrl, err = sc.cloneDisk(instanceInfo.BootDiskSourceURL, scanner.projectID, scanner.zone, "cnspec-"+target.InstanceName+"-snapshot-"+time.Now().Format("2006-01-02t15-04-05z00-00"))
			if err != nil {
				log.Error().Err(err).Str("disk", diskUrl).Msg("could not complete snapshot creation")
				return nil, errors.Wrap(err, "could not create gcp instance snapshot")
			}
			log.Debug().Str("disk", diskUrl).Msg("cloned disk from instance disk")
			mi.diskUrl = diskUrl

		}
	case "snapshot":
		snapshotInfo, err := sc.SnapshotInfo(target.ProjectID, target.SnapshotName)
		if err != nil {
			return nil, err
		}

		diskUrl, err = sc.createSnapshotDisk(snapshotInfo.SnapshotUrl, scanner.projectID, scanner.zone, "cnspec-"+target.InstanceName+"-snapshot-"+time.Now().Format("2006-01-02t15-04-05z00-00"))
		if err != nil {
			log.Error().Err(err).Str("disk", diskUrl).Msg("could not complete snapshot disk creation")
			return nil, errors.Wrap(err, "could not create disk from snapshot")
		}
		log.Debug().Str("disk", diskUrl).Msg("created disk from snapshot")
		mi.diskUrl = diskUrl
	default:
		return nil, errors.New("invalid target type")
	}

	// attach created disk to the scanner instance
	err = sc.attachDisk(scanner.projectID, scanner.zone, scanner.instanceName, mi.diskUrl, mi.deviceName)
	if err != nil {
		return nil, err
	}

	errorHandler := func() {
		// use different err variable to ensure it does not overshadow the real error
		dErr := sc.detachDisk(scanner.projectID, scanner.zone, scanner.instanceName, mi.deviceName)
		if dErr != nil {
			log.Error().Err(dErr).Msg("could not detach created disk")
		}

		dErr = sc.deleteCreatedDisk(mi.diskUrl)
		if dErr != nil {
			log.Error().Err(dErr).Msg("could not delete created disk")
		}
	}

	// mount volume
	shell := []string{"sh", "-c"}
	volumeMounter := snapshot.NewVolumeMounter(shell)
	err = volumeMounter.Mount()
	if err != nil {
		log.Error().Err(err).Msg("unable to complete mount step")
		errorHandler()
		return nil, err
	}

	// create and initialize fs provider
	fsProvider, err := fs.New(&providers.Config{
		Path:       volumeMounter.ScanDir,
		Backend:    providers.ProviderType_FS,
		PlatformId: pCfg.PlatformId,
		Options:    pCfg.Options,
	})
	if err != nil {
		errorHandler()
		return nil, err
	}

	p := &Provider{
		Provider:        fsProvider,
		opts:            pCfg.Options,
		targetType:      target.TargetType,
		volumeMounter:   volumeMounter,
		snapshotCreator: sc,
		target:          target,
		scanner:         *scanner,
		mountInfo:       mi,
		identifier:      pCfg.PlatformId,
	}

	return p, nil
}

type Provider struct {
	*fs.Provider
	opts map[string]string
	// the type of object we're targeting (instance, disk, snapshot)
	targetType      string
	volumeMounter   *snapshot.VolumeMounter
	snapshotCreator *SnapshotCreator
	target          scanTarget
	scanner         scannerInstance
	mountInfo       mountInfo
	identifier      string
}

func (p *Provider) Close() {
	if p == nil {
		return
	}

	if p.opts != nil {
		if p.opts[snapshot.NoSetup] == "true" {
			return
		}
	}

	err := p.volumeMounter.UnmountVolumeFromInstance()
	if err != nil {
		log.Error().Err(err).Msg("unable to unmount volume")
	}

	if p.snapshotCreator != nil {
		err = p.snapshotCreator.detachDisk(p.scanner.projectID, p.scanner.zone, p.scanner.instanceName, p.mountInfo.deviceName)
		if err != nil {
			log.Error().Err(err).Msg("unable to detach volume")
		}

		err = p.snapshotCreator.deleteCreatedDisk(p.mountInfo.diskUrl)
		if err != nil {
			log.Error().Err(err).Msg("could not delete created disk")
		}
	}

	err = p.volumeMounter.RemoveTempScanDir()
	if err != nil {
		log.Error().Err(err).Msg("unable to remove dir")
	}
}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_Aws_Ebs,
	}
}

func (p *Provider) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (p *Provider) Runtime() string {
	return providers.RUNTIME_GCP_COMPUTE
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func (p *Provider) Identifier() (string, error) {
	return p.identifier, nil
}
