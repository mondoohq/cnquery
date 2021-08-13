package awsec2ebs

import (
	"os"
	"syscall"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
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
		return err
	}
	return err
}

const mountDir string = "/dev/xvdk"
const mountDirLoc string = mountDir + "1"
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
	return Ext4
}

func (t *Ec2EbsTransport) MountVolumeToScanDir(fsType FsType) error {
	log.Info().Str("fs type", fsType.String()).Str("mount dir", mountDirLoc).Str("scan dir", ScanDir).Msg("mount volume to scan dir")
	var flags uintptr = syscall.MS_MGC_VAL
	if err := unix.Mount(mountDirLoc, ScanDir, fsType.String(), flags, ""); err != nil && err != unix.EBUSY { // does not compile on mac bc mount is not implemented for darwin
		log.Error().Err(err).Msg("failed to mount dir")
		return err
	}
	return nil
}
