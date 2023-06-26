package snapshot

import (
	"encoding/json"
	"io"
	"os"
	"strings"

	"go.mondoo.com/cnquery/stringx"

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
	return m.mountVolume(fsInfo)
}

func (m *VolumeMounter) createScanDir() error {
	log.Info().Msg("create tmp scan dir")
	dir, err := os.MkdirTemp("", "cnspec-scan")
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
	blockEntries := blockDevices{}
	if err := json.Unmarshal(data, &blockEntries); err != nil {
		return nil, err
	}
	var fsInfo *fsInfo
	if m.opts[NoSetup] == "true" {
		// this means we didn't attach the volume to the instance
		// so we need to make a best effort guess
		return blockEntries.GetUnnamedBlockEntry()
	}

	fsInfo, err = blockEntries.GetBlockEntryByName(m.VolumeAttachmentLoc)
	if err == nil && fsInfo != nil {
		return fsInfo, nil
	} else {
		// if we get here, we couldn't find a fs loaded at the expected location
		// AWS does not guarantee this, so that's expected. fallback to find non-boot and non-mounted volume
		fsInfo, err = blockEntries.GetUnmountedBlockEntry()
		if err == nil && fsInfo != nil {
			return fsInfo, nil
		}
	}
	return nil, err
}

func (m *VolumeMounter) mountVolume(fsInfo *fsInfo) error {
	log.Info().Msg("mount volume")
	opts := []string{}
	if fsInfo.fstype == "xfs" {
		opts = append(opts, "nouuid")
	}
	opts = stringx.DedupStringArray(opts)
	log.Debug().Str("fstype", fsInfo.fstype).Str("device", fsInfo.name).Str("scandir", m.ScanDir).Str("opts", strings.Join(opts, ",")).Msg("mount volume to scan dir")
	return Mount(fsInfo.name, m.ScanDir, fsInfo.fstype, opts)
}

func (m *VolumeMounter) UnmountVolumeFromInstance() error {
	log.Info().Msg("unmount volume")
	if err := Unmount(m.ScanDir); err != nil {
		log.Error().Err(err).Msg("failed to unmount dir")
		return err
	}
	return nil
}

func (m *VolumeMounter) RemoveTempScanDir() error {
	log.Info().Msg("remove created dir")
	return os.RemoveAll(m.ScanDir)
}
