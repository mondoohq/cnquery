package resources

import (
	"github.com/cockroachdb/errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/kernel"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

func (k *lumiKernel) init(args *lumi.Args) (*lumi.Args, Kernel, error) {
	// this resource is only supported on linux
	platform, err := k.Runtime.Motor.Platform()
	if err != nil {
		return nil, nil, err
	}

	supported := false
	if platform.IsFamily("linux") || platform.IsFamily("darwin") || platform.Name == "freebsd" {
		supported = true
	}

	if supported == false {
		return nil, nil, errors.New("kernel resource is only supported for linux platforms")
	}

	return args, nil, nil
}

func (k *lumiKernel) id() (string, error) {
	return "kernel", nil
}

func (k *lumiKernel) GetParameters() (map[string]interface{}, error) {
	// find suitable kernel module manager
	mm, err := kernel.ResolveManager(k.Runtime.Motor)
	if mm == nil || err != nil {
		return nil, errors.Wrap(err, "could not detect suiteable kernel module manager for platform")
	}

	// retrieve all kernel modules
	kernelParameters, err := mm.Parameters()
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

// TODO: something is going wrong with proc file fetching, get this back to work
func (k *lumiKernel) getParametersFromProc() (map[string]interface{}, error) {
	// TODO: consider registration for directory changes
	sysctlPath := "/proc/sys/"

	fs := k.Runtime.Motor.Transport.FS()

	f, err := fs.Open(sysctlPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tarStream, err := fsutil.Tar(fs, f)
	if err != nil {
		return nil, err
	}
	defer tarStream.Close()

	kernelParameters, err := kernel.ParseLinuxSysctlProc(sysctlPath, tarStream)
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
	mm, err := kernel.ResolveManager(k.Runtime.Motor)
	if mm == nil || err != nil {
		return nil, errors.Wrap(err, "could not detect suiteable kernel module manager for platform")
	}

	// retrieve all kernel modules
	kernelModules, err := mm.Modules()
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve kernel module list for platform")
	}
	log.Debug().Int("modules", len(kernelModules)).Msg("lumi[kernel.modules]> modules")

	// create lumi kernel module entry resources for each entry
	moduleEntries := make([]interface{}, len(kernelModules))
	namedMap := make(map[string]KernelModule, len(kernelModules))
	for i, kernelModule := range kernelModules {

		lumiKernelModule, err := k.Runtime.CreateResource("kernel.module",
			"name", kernelModule.Name,
			"size", kernelModule.Size,
			"loaded", true,
		)
		if err != nil {
			return nil, err
		}

		moduleEntries[i] = lumiKernelModule.(KernelModule)
		namedMap[kernelModule.Name] = lumiKernelModule.(KernelModule)
	}

	k.Cache.Store("_modules", &lumi.CacheEntry{Data: namedMap})

	// return the kernel modules as new entries
	return moduleEntries, nil
}

func (k *lumiKernelModule) init(args *lumi.Args) (*lumi.Args, KernelModule, error) {
	// TODO: look at the args and determine if we init all or ask for listing of all modules
	if len(*args) > 2 {
		return args, nil, nil
	}

	nameRaw := (*args)["name"]
	if nameRaw == nil {
		return args, nil, nil
	}
	name := nameRaw.(string)

	obj, err := k.Runtime.CreateResource("kernel")
	if err != nil {
		return nil, nil, err
	}
	kernel := obj.(Kernel)

	_, err = kernel.Modules()
	if err != nil {
		return nil, nil, err
	}

	c, ok := kernel.LumiResource().Cache.Load("_modules")
	if !ok {
		return nil, nil, errors.New("cannot get map of kernel modules")
	}
	cmap := c.Data.(map[string]KernelModule)

	// TODO: this won't be necessary if we can reference the ID
	(*args)["name"] = name
	(*args)["size"] = ""
	(*args)["loaded"] = false

	item := cmap[name]
	if item != nil {
		// TODO: do this instead of duplicating it!
		// (*args)["id"] = pkg.LumiResource().Id
		// Workaround: we fill in the fields we need to make the id() method
		// generate the same ID
		(*args)["size"], _ = item.Size()
		(*args)["loaded"] = true
	}

	return args, nil, nil
}

func (k *lumiKernelModule) id() (string, error) {
	return k.Name()
}
