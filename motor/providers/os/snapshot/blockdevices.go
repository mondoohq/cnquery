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

func getRootBlockEntry(blockEntries blockdevices) (*fsInfo, error) {
	log.Debug().Msg("get root block entry")
	for i := range blockEntries.Blockdevices {
		d := blockEntries.Blockdevices[i]
		log.Debug().Str("name", d.Name).Interface("children", d.Children).Interface("mountpoint", d.MountPoint).Msg("found block device")
		for i := range d.Children {
			entry := d.Children[i]
			if validateBlockEntryValid(entry) {
				devFsName := "/dev/" + entry.Name
				return &fsInfo{name: devFsName, fstype: entry.FsType}, nil
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
			if validateBlockEntryValidAndUnmounted(entry) {
				devFsName := "/dev/" + entry.Name
				return &fsInfo{name: devFsName, fstype: entry.FsType}, nil
			}
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func validateBlockEntryValid(entry blockdevice) bool {
	return entry.Uuid != "" && entry.FsType != "" && entry.Label != "EFI"
}

func validateBlockEntryValidAndUnmounted(entry blockdevice) bool {
	return entry.Uuid != "" && entry.FsType != "" && entry.Label != "EFI" && entry.MountPoint == ""
}
