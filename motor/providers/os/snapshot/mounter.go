package snapshot

import (
	"encoding/json"
	"io"
	"os"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	osProvider "go.mondoo.com/cnquery/motor/providers/os"
)

const NoSetup = "no-setup"

type VolumeMounter struct {
	// the tmp dir we create; serves as the directory we mount the volume to
	ScanDir string
	// where we tell AWS to attach the volume; it doesn't necessarily get attached there, but we have to reference this same location when detaching
	VolumeAttachmentLoc string
	opts                map[string]string
	cmdRunner           osProvider.CommandRunner
}

func NewVolumeMounter(shell []string) *VolumeMounter {
	return &VolumeMounter{
		cmdRunner: &LocalCommandRunner{shell: shell},
	}
}

func (m *VolumeMounter) Mount() error {
	err := m.createScanDir()
	if err != nil {
		return err
	}
	fsInfo, err := m.getFsInfo()
	if err != nil {
		return err
	}
	if fsInfo == nil {
		return errors.New("unable to find target volume on instance")
	}
	log.Info().Str("device name", fsInfo.name).Msg("found target volume")
	err = m.mountVolume(fsInfo)
	if err != nil {
		return err
	}
	return err
}

func (m *VolumeMounter) createScanDir() error {
	log.Info().Msg("create tmp scan dir")
	dir, err := os.MkdirTemp("", "mondooscan")
	if err != nil {
		log.Error().Err(err).Msg("error creating directory")
		return err
	}
	m.ScanDir = dir
	return nil
}

func (m *VolumeMounter) getFsInfo() (*fsInfo, error) {
	log.Info().Msg("search for target volume")

	// TODO: replace with mql query once version with lsblk resource is released
	// TODO: only use sudo if we are not root
	cmd, err := m.cmdRunner.RunCommand("sudo lsblk -f --json")
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}
	blockEntries := blockdevices{}
	if err := json.Unmarshal(data, &blockEntries); err != nil {
		return nil, err
	}
	var fsInfo *fsInfo
	if m.opts[NoSetup] == "true" {
		// this means we didn't attach the volume to the instance
		// so we need to make a best effort guess
		return getUnnamedBlockEntry(blockEntries)
	}

	fsInfo, err = getMatchingBlockEntryByName(blockEntries, m.VolumeAttachmentLoc)
	if err == nil && fsInfo != nil {
		return fsInfo, nil
	} else {
		// if we get here, we couldn't find an fs loaded at the expected location
		// AWS does not guarantee this, so that's expected. fallback to the non-root volume
		fsInfo, err = getNonRootBlockEntry(blockEntries)
		if err == nil && fsInfo != nil {
			return fsInfo, nil
		}
	}
	return nil, err
}

func getUnnamedBlockEntry(blockEntries blockdevices) (*fsInfo, error) {
	fsInfo, err := getNonRootBlockEntry(blockEntries)
	if err == nil && fsInfo != nil {
		return fsInfo, nil
	} else {
		// if we get here, there was no non-root, non-mounted volume on the instance
		// this is expected in the "no setup" case where we start an instance with the target
		// volume attached and only that volume attached
		fsInfo, err = getRootBlockEntry(blockEntries)
		if err == nil && fsInfo != nil {
			return fsInfo, nil
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func getNonRootBlockEntry(blockEntries blockdevices) (*fsInfo, error) {
	log.Debug().Msg("get non root block entry")
	for i := range blockEntries.Blockdevices {
		d := blockEntries.Blockdevices[i]
		log.Debug().Str("name", d.Name).Interface("children", d.Children).Interface("mountpoint", d.MountPoint).Msg("found block device")
		if d.MountPoint != "" { // empty string means it is not mounted
			continue
		}
		for i := range d.Children {
			entry := d.Children[i]
			if validateBlockEntryValidAndUnmounted(entry) {
				devFsName := "/dev/" + entry.Name
				return &fsInfo{name: devFsName, fstype: entry.FsType}, nil
			}
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func (m *VolumeMounter) mountVolume(fsInfo *fsInfo) error {
	log.Info().Msg("mount volume")
	opts := ""
	if fsInfo.fstype == "xfs" {
		opts = "nouuid"
	}
	log.Debug().Str("fstype", fsInfo.fstype).Str("device", fsInfo.name).Str("scandir", m.ScanDir).Str("opts", opts).Msg("mount volume to scan dir")
	if err := Mount(fsInfo.name, m.ScanDir, fsInfo.fstype, opts); err != nil {
		return err
	}
	return nil
}

func (m *VolumeMounter) UnmountVolumeFromInstance() error {
	log.Info().Msg("unmount volume")
	if err := Unmount(m.ScanDir); err != nil {
		log.Error().Err(err).Msg("failed to unmount dir")
		return err
	}
	return nil
}

func (m *VolumeMounter) RemoveCreatedDir() error {
	log.Info().Msg("remove created dir")
	return os.RemoveAll(m.ScanDir)
}
