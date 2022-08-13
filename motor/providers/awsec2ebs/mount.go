package awsec2ebs

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/providers/awsec2ebs/custommount"
)

func (t *Provider) Mount() error {
	err := t.CreateScanDir()
	if err != nil {
		return err
	}
	fsInfo, err := t.GetFsInfo()
	if err != nil {
		return err
	}
	if fsInfo == nil {
		return errors.New("unable to find target volume on instance")
	}
	log.Info().Str("device name", fsInfo.name).Msg("found target volume")
	err = t.MountVolume(fsInfo)
	if err != nil {
		return err
	}
	return err
}

func (t *Provider) CreateScanDir() error {
	log.Info().Msg("create tmp scan dir")
	dir, err := ioutil.TempDir("", "mondooscan")
	if err != nil {
		log.Error().Err(err).Msg("error creating directory")
		return err
	}
	t.tmpInfo.scanDir = dir
	return nil
}

func (t *Provider) GetFsInfo() (*fsInfo, error) {
	log.Info().Msg("search for target volume")
	cmd, err := t.RunCommand("sudo lsblk -f --json") // replace with mql query once version with lsblk resource is released
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}
	blockEntries := blockdevices{}
	if err := json.Unmarshal(data, &blockEntries); err != nil {
		return nil, err
	}
	var fsInfo *fsInfo
	if t.opts[NoSetup] == "true" {
		// this means we didnt attach the volume to the instance
		// so we need to make a best effort guess
		return getUnnamedBlockEntry(blockEntries)
	}

	fsInfo, err = getMatchingBlockEntryByName(blockEntries, t.tmpInfo.volumeAttachmentLoc)
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
		log.Debug().Str("name", d.Name).Interface("children", d.Children).Interface("mountpoint", d.Mountpoint).Msg("found block device")
		if d.Mountpoint != "" { // empty string means it is not mounted
			continue
		}
		for i := range d.Children {
			entry := d.Children[i]
			if validateBlockEntryValidAndUnmounted(entry) {
				devFsName := "/dev/" + entry.Name
				return &fsInfo{name: devFsName, fstype: entry.Fstype}, nil
			}
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func getRootBlockEntry(blockEntries blockdevices) (*fsInfo, error) {
	log.Debug().Msg("get root block entry")
	for i := range blockEntries.Blockdevices {
		d := blockEntries.Blockdevices[i]
		log.Debug().Str("name", d.Name).Interface("children", d.Children).Interface("mountpoint", d.Mountpoint).Msg("found block device")
		for i := range d.Children {
			entry := d.Children[i]
			if validateBlockEntryValid(entry) {
				devFsName := "/dev/" + entry.Name
				return &fsInfo{name: devFsName, fstype: entry.Fstype}, nil
			}
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func getMatchingBlockEntryByName(blockEntries blockdevices, name string) (*fsInfo, error) {
	log.Debug().Str("name", name).Msg("get matching block entry")
	var secondName string
	if strings.HasPrefix(name, "/dev/sd") {
		// sdh and xvdh are interchangeable
		end := strings.TrimPrefix(name, "/dev/sd")
		secondName = "/dev/xvd" + end
	}
	for i := range blockEntries.Blockdevices {
		d := blockEntries.Blockdevices[i]
		log.Debug().Str("name", d.Name).Interface("children", d.Children).Interface("mountpoint", d.Mountpoint).Msg("found block device")
		fullDeviceName := "/dev/" + d.Name
		if name != fullDeviceName { // check if the device name matches
			if secondName == "" {
				continue
			}
			if secondName != fullDeviceName { // check if the device name matches the second name option (sdh and xvdh are interchangeable)
				continue
			}
		}
		log.Debug().Msg("found match")
		for i := range d.Children {
			entry := d.Children[i]
			if validateBlockEntryValidAndUnmounted(entry) {
				devFsName := "/dev/" + entry.Name
				return &fsInfo{name: devFsName, fstype: entry.Fstype}, nil
			}
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func validateBlockEntryValid(entry blockdevice) bool {
	return entry.Uuid != "" && entry.Fstype != "" && entry.Label != "EFI"
}

func validateBlockEntryValidAndUnmounted(entry blockdevice) bool {
	return entry.Uuid != "" && entry.Fstype != "" && entry.Label != "EFI" && entry.Mountpoint == ""
}

func (t *Provider) MountVolume(fsInfo *fsInfo) error {
	log.Info().Msg("mount volume")
	opts := ""
	if fsInfo.fstype == "xfs" {
		opts = "nouuid"
	}
	log.Debug().Str("fstype", fsInfo.fstype).Str("device", fsInfo.name).Str("scandir", t.tmpInfo.scanDir).Str("opts", opts).Msg("mount volume to scan dir")
	if err := custommount.Mount(fsInfo.name, t.tmpInfo.scanDir, fsInfo.fstype, opts); err != nil {
		return err
	}
	return nil
}

type fsInfo struct {
	name   string
	fstype string
}

type blockdevices struct {
	Blockdevices []blockdevice `json:"blockdevices,omitempty"`
}

type blockdevice struct {
	Name       string        `json:"name,omitempty"`
	Fstype     string        `json:"fstype,omitempty"`
	Label      string        `json:"label,omitempty"`
	Uuid       string        `json:"uuid,omitempty"`
	Mountpoint string        `json:"mountpoint,omitempty"`
	Children   []blockdevice `json:"children,omitempty"`
}
