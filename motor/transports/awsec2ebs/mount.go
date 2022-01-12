package awsec2ebs

import (
	"encoding/json"
	"io/ioutil"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/transports/awsec2ebs/custommount"
)

func (t *Ec2EbsTransport) Mount() error {
	err := t.CreateScanDir()
	if err != nil {
		return err
	}
	fsInfo, err := t.GetFsType()
	if err != nil {
		return err
	}
	err = t.MountVolume(fsInfo)
	if err != nil {
		return err
	}
	return err
}

func (t *Ec2EbsTransport) CreateScanDir() error {
	log.Info().Msg("create tmp scan dir")
	dir, err := ioutil.TempDir("", "mondooscan")
	if err != nil {
		log.Error().Err(err).Msg("error creating directory")
		return err
	}
	t.tmpInfo.scanDir = dir
	return nil
}

func (t *Ec2EbsTransport) GetFsType() (*fsInfo, error) {
	log.Info().Msg("get fs type")
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
	for i := range blockEntries.Blockdevices {
		d := blockEntries.Blockdevices[i]
		log.Debug().Msgf("found block devices %v", d.Children)
		for i := range d.Children {
			entry := d.Children[i]
			if entry.Mountpoint == "" && entry.Uuid != "" && entry.Fstype != "" && entry.Label != "EFI" {
				devFsName := "/dev/" + entry.Name
				return &fsInfo{name: devFsName, fstype: entry.Fstype}, nil
			}

		}
	}
	return nil, err
}

func (t *Ec2EbsTransport) MountVolume(fsInfo *fsInfo) error {
	opts := ""
	if fsInfo.fstype == "xfs" {
		opts = "nouuid"
	}
	log.Info().Str("fstype", fsInfo.fstype).Str("device", fsInfo.name).Str("scandir", t.tmpInfo.scanDir).Str("opts", opts).Msg("mount volume to scan dir")
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
