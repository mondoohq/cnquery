package snapshot

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
)

type blockdevices struct {
	Blockdevices []blockdevice `json:"blockdevices,omitempty"`
}

type blockdevice struct {
	Name       string        `json:"name,omitempty"`
	FsType     string        `json:"fstype,omitempty"`
	Label      string        `json:"label,omitempty"`
	Uuid       string        `json:"uuid,omitempty"`
	MountPoint string        `json:"mountpoint,omitempty"`
	Children   []blockdevice `json:"children,omitempty"`
}

type fsInfo struct {
	name   string
	fstype string
}

func (blockEntries blockdevices) GetRootBlockEntry() (*fsInfo, error) {
	log.Debug().Msg("get root block entry")
	for i := range blockEntries.Blockdevices {
		d := blockEntries.Blockdevices[i]
		log.Debug().Str("name", d.Name).Interface("children", d.Children).Interface("mountpoint", d.MountPoint).Msg("found block device")
		for i := range d.Children {
			entry := d.Children[i]
			if entry.IsValid() {
				devFsName := "/dev/" + entry.Name
				return &fsInfo{name: devFsName, fstype: entry.FsType}, nil
			}
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func (blockEntries blockdevices) GetBlockEntryByName(name string) (*fsInfo, error) {
	log.Debug().Str("name", name).Msg("get matching block entry")
	var secondName string
	if strings.HasPrefix(name, "/dev/sd") {
		// sdh and xvdh are interchangeable
		end := strings.TrimPrefix(name, "/dev/sd")
		secondName = "/dev/xvd" + end
	}
	for i := range blockEntries.Blockdevices {
		d := blockEntries.Blockdevices[i]
		log.Debug().Str("name", d.Name).Interface("children", d.Children).Interface("mountpoint", d.MountPoint).Msg("found block device")
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
			if entry.IsValidAndUnmounted() {
				devFsName := "/dev/" + entry.Name
				return &fsInfo{name: devFsName, fstype: entry.FsType}, nil
			}
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func (blockEntries blockdevices) GetUnnamedBlockEntry() (*fsInfo, error) {
	fsInfo, err := blockEntries.GetNonRootBlockEntry()
	if err == nil && fsInfo != nil {
		return fsInfo, nil
	} else {
		// if we get here, there was no non-root, non-mounted volume on the instance
		// this is expected in the "no setup" case where we start an instance with the target
		// volume attached and only that volume attached
		fsInfo, err = blockEntries.GetRootBlockEntry()
		if err == nil && fsInfo != nil {
			return fsInfo, nil
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func (blockEntries blockdevices) GetNonRootBlockEntry() (*fsInfo, error) {
	log.Debug().Msg("get non root block entry")
	for i := range blockEntries.Blockdevices {
		d := blockEntries.Blockdevices[i]
		log.Debug().Str("name", d.Name).Interface("children", d.Children).Interface("mountpoint", d.MountPoint).Msg("found block device")
		if d.MountPoint != "" { // empty string means it is not mounted
			continue
		}
		for i := range d.Children {
			entry := d.Children[i]
			if entry.IsValidAndUnmounted() {
				devFsName := "/dev/" + entry.Name
				return &fsInfo{name: devFsName, fstype: entry.FsType}, nil
			}
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func (entry blockdevice) IsValid() bool {
	return entry.Uuid != "" && entry.FsType != "" && entry.Label != "EFI"
}

func (entry blockdevice) IsValidAndUnmounted() bool {
	return entry.Uuid != "" && entry.FsType != "" && entry.Label != "EFI" && entry.MountPoint == ""
}
