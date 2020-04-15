package resources

import (
	"errors"

	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/procfs"
	"go.mondoo.io/mondoo/motor/motoros/fsutil"
)

func (s *lumiKernel) init(args *lumi.Args) (*lumi.Args, error) {
	// this resource is only supported on linux
	platform, err := s.Runtime.Motor.Platform()
	if err != nil {
		return nil, err
	}

	supported := false
	for _, f := range platform.Family {
		if f == "linux" {
			supported = true
		}
	}
	if supported == false {
		return nil, errors.New("kernel resource is only supported for linux platforms")
	}

	return args, nil
}

func (s *lumiKernel) id() (string, error) {
	return "kernel_parameters", nil
}

func (s *lumiKernel) GetParameters() (map[string]interface{}, error) {
	// TODO: consider registration for directory changes
	sysctlPath := "/proc/sys/"

	fs := s.Runtime.Motor.Transport.FS()

	f, err := s.Runtime.Motor.Transport.File(sysctlPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tarStream, err := fsutil.Tar(fs, f)
	if err != nil {
		return nil, err
	}
	defer tarStream.Close()

	kernelParameters, err := procfs.ParseLinuxSysctl(sysctlPath, tarStream)
	if err != nil {
		return nil, err
	}

	// copy values to fulfill the interface
	res := make(map[string]interface{})
	for key, value := range kernelParameters {
		res[key] = value
	}

	return res, nil
}
