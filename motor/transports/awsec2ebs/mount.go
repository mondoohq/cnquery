package awsec2ebs

import (
	"os"

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
	err := t.EnsureScanDir()
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

const mountDir string = "/dev/xvdk"
const mountDirLoc string = mountDir + "1"
const mountDirLoc2 string = mountDir + "2"
const ScanDir string = "/mondooscandata"

func (t *Ec2EbsTransport) EnsureScanDir() error {
	log.Info().Msg("ensure scan dir")
	if err := os.MkdirAll(ScanDir, 0555); err != nil && !os.IsExist(err) {
		log.Error().Err(err).Msg("error creating directory")
		return err
	}
	return nil
}

func (t *Ec2EbsTransport) GetFsType() FsType {
	log.Info().Msg("get fs type")
	// use mql for this
	return t.fsType // workaround til we read with mql
}

func (t *Ec2EbsTransport) MountVolumeToScanDir(fsType FsType) error {
	log.Info().Str("fs type", fsType.String()).Str("mount dir", mountDirLoc).Str("scan dir", ScanDir).Msg("mount volume to scan dir")
	if fsType == Xfs {
		err := mountXfsVolume()
		if err != nil {
			// try ext4
			err2 := mountExt4Volume()
			if err2 != nil {
				return errors.Wrap(err, err2.Error())
			}
		}
	} else {
		err := mountExt4Volume()
		if err != nil {
			// try xfs
			err2 := mountXfsVolume()
			if err2 != nil {
				return errors.Wrap(err, err2.Error())
			}
		}
	}

	return nil
}

func mountXfsVolume() error {
	if err := custommount.Mount(mountDirLoc, ScanDir, Xfs.String(), "nouuid"); err != nil {
		if err := custommount.Mount(mountDirLoc2, ScanDir, Xfs.String(), "nouuid"); err != nil {
			return err
		}
		return err
	}
	return nil
}

func mountExt4Volume() error {
	if err := custommount.Mount(mountDirLoc, ScanDir, Ext4.String(), ""); err != nil {
		if err := custommount.Mount(mountDirLoc2, ScanDir, Ext4.String(), ""); err != nil {
			return err
		}
	}
	return nil
}
