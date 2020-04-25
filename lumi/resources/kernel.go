package resources

import (
	"github.com/pkg/errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/kernelmodule"
	"go.mondoo.io/mondoo/lumi/resources/procfs"
	"go.mondoo.io/mondoo/motor/motoros/fsutil"
)

func (k *lumiKernel) init(args *lumi.Args) (*lumi.Args, error) {
	// this resource is only supported on linux
	platform, err := k.Runtime.Motor.Platform()
	if err != nil {
		return nil, err
	}

	supported := false
	if platform.IsFamily("linux") || platform.IsFamily("darwin") || platform.Name == "freebsd" {
		supported = true
	}

	if supported == false {
		return nil, errors.New("kernel resource is only supported for linux platforms")
	}

	return args, nil
}

func (s *lumiKernel) id() (string, error) {
	return "kernel", nil
}

func (k *lumiKernel) GetParameters() (map[string]interface{}, error) {
	// TODO: consider registration for directory changes
	sysctlPath := "/proc/sys/"

	fs := k.Runtime.Motor.Transport.FS()

	f, err := k.Runtime.Motor.Transport.File(sysctlPath)
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

func (k *lumiKernel) GetModules() ([]interface{}, error) {
	// find suitable kernel module manager
	mm, err := kernelmodule.ResolveManager(k.Runtime.Motor)
	if mm == nil || err != nil {
		return nil, errors.Wrap(err, "Could not detect suiteable kernel module manager for platform")
	}

	// retrieve all kernel modules
	kernelModules, err := mm.List()
	if err != nil {
		return nil, errors.Wrap(err, "Could not retrieve kernel module list for platform")
	}
	log.Debug().Int("modules", len(kernelModules)).Msg("lumi[kernel.modules]> modules")

	// create lumi kernel module entry resources for each entry
	moduleEntries := make([]interface{}, len(kernelModules))
	for i, kernelModule := range kernelModules {

		lumiKernelModule, err := k.Runtime.CreateResource("kernel_module",
			"name", kernelModule.Name,
			"size", kernelModule.Size,
		)
		if err != nil {
			return nil, err
		}

		moduleEntries[i] = lumiKernelModule.(Kernel_module)
	}

	// return the kernel modules as new entries
	return moduleEntries, nil
}

func (k *lumiKernel_module) id() (string, error) {
	return k.Name()
}
