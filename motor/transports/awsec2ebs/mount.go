package awsec2ebs

import (
	"io/ioutil"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/transports/awsec2ebs/custommount"
)

/* testing command
local: make mondoo/app/linux
local: scp -i ~/.testing_pems/vjSEPTTEST.pem ./dist/mondoo-linux-amd64 ec2-user@54.226.173.170:~
on instance: sudo su && ./mondoo-linux-amd64 scan -t aws-ec2-ebs://account/185972265011/region/us-east-1/instance/i-07f67838ada5879af --config aws-ssm-ps://region/us-east-1/parameter/MondooAgentConfig
*/
func (t *Ec2EbsTransport) Mount() error {
	err := t.CreateScanDir()
	if err != nil {
		return err
	}
	fsType := t.GetFsType()
	err = t.MountVolumeToScanDir(fsType)
	if err != nil {
		t.Close()
		return err
	}
	return err
}

const attachedFS string = "/dev/xvdk"
const attachedFSLoc string = attachedFS + "1"
const attachedFSLoc2 string = attachedFS + "2"

func (t *Ec2EbsTransport) CreateScanDir() error {
	log.Info().Msg("create tmp scan dir")
	dir, err := ioutil.TempDir("", "mondooscan")
	if err != nil {
		log.Error().Err(err).Msg("error creating directory")
		return err
	}
	t.scanDir = dir
	return nil
}

func (t *Ec2EbsTransport) GetFsType() FsType {
	log.Info().Msg("get fs type")
	// use mql for this
	return t.fsType // workaround til we read with mql
}

func (t *Ec2EbsTransport) MountVolumeToScanDir(fsType FsType) error {
	log.Info().Str("fs type", fsType.String()).Str("mount dir", attachedFSLoc).Str("scan dir", t.scanDir).Msg("mount volume to scan dir")
	if fsType == Xfs {
		err := mountXfsVolume(t.scanDir)
		if err != nil {
			// try ext4
			err2 := mountExt4Volume(t.scanDir)
			if err2 != nil {
				return errors.Wrap(err, err2.Error())
			}
		}
	} else {
		err := mountExt4Volume(t.scanDir)
		if err != nil {
			// try xfs
			err2 := mountXfsVolume(t.scanDir)
			if err2 != nil {
				return errors.Wrap(err, err2.Error())
			}
		}
	}

	return nil
}

func mountXfsVolume(scanDir string) error {
	if err := custommount.Mount(attachedFSLoc, scanDir, Xfs.String(), "nouuid"); err != nil {
		if err := custommount.Mount(attachedFSLoc2, scanDir, Xfs.String(), "nouuid"); err != nil {
			return err
		}
		return err
	}
	return nil
}

func mountExt4Volume(scanDir string) error {
	if err := custommount.Mount(attachedFSLoc, scanDir, Ext4.String(), ""); err != nil {
		if err := custommount.Mount(attachedFSLoc2, scanDir, Ext4.String(), ""); err != nil {
			return err
		}
	}
	return nil
}
