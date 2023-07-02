package core

import (
	"strings"

	"errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core/kernel"
)

func (k *mqlKernel) init(args *resources.Args) (*resources.Args, Kernel, error) {
	// this resource is only supported on linux
	platform, err := k.MotorRuntime.Motor.Platform()
	if err != nil {
		return nil, nil, err
	}

	supported := false
	if platform.IsFamily("linux") || platform.IsFamily("darwin") || platform.Name == "freebsd" {
		supported = true
	}

	if supported == false {
		return nil, nil, errors.New("kernel resource is only supported for unix platforms")
	}

	return args, nil, nil
}

func (k *mqlKernel) id() (string, error) {
	return "kernel", nil
}

type KernelVersion struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Running bool   `json:"running"`
}

func (k *mqlKernel) GetInstalled() ([]interface{}, error) {
	res := []KernelVersion{}

	pf, err := k.MotorRuntime.Motor.Platform()
	if err != nil {
		return nil, err
	}

	if pf.IsFamily(platform.FAMILY_LINUX) {

		// 1. gather running kernel information
		kernelInfoRaw, err := k.Info()
		if err != nil {
			return nil, errors.New("could not determine kernel version")
		}

		kernelInfo, ok := kernelInfoRaw.(map[string]interface{})
		if !ok {
			return nil, errors.New("no structured kernel information found")
		}

		runningKernelVersion := kernelInfo["version"].(string)

		// 2. get all packages
		obj, err := k.MotorRuntime.CreateResource("packages")
		if err != nil {
			return nil, err
		}
		packages := obj.(Packages)

		mqlPkgs, err := packages.List()
		if err != nil {
			return nil, err
		}

		filterKernel := func(pkg Package) {}

		if pf.IsFamily("debian") {
			// debian based systems
			// kernel version is  "4.19.0-13-cloud-amd64"
			// filter by packages named "linux-image-*"
			//[{
			//	name: "linux-image-4.19.0-12-cloud-amd64"
			//	version: "4.19.152-1"
			//}, {
			//	name: "linux-image-4.19.0-13-cloud-amd64"
			//	version: "4.19.160-2"
			//}, {
			//	name: "linux-image-cloud-amd64"
			//	version: "4.19+105+deb10u8"
			//}]
			filterKernel = func(pkg Package) {
				name, _ := pkg.Name()

				if strings.HasPrefix(name, "linux-image") {
					version, _ := pkg.Version()

					kernelName := strings.TrimPrefix(name, "linux-image-")
					running := false
					if kernelName == runningKernelVersion {
						running = true
					}

					res = append(res, KernelVersion{
						Name:    kernelName,
						Version: version,
						Running: running,
					})
				}
			}
		} else if pf.IsFamily("redhat") || pf.Name == "amazonlinux" {
			// rpm based systems
			// kernel version is  "3.10.0-1160.11.1.el7.x86_64"
			// filter by packages named "kernel"
			//[{
			//	name: "kernel"
			//	version: "3.10.0-1127.el7"
			//}, {
			//	name: "kernel"
			//	version: "3.10.0-1160.11.1.el7"
			//}, {
			//	name: "kernel"
			//	version: "3.10.0-1127.19.1.el7"
			//}]
			filterKernel = func(pkg Package) {
				name, _ := pkg.Name()

				if name == "kernel" {
					version, _ := pkg.Version()
					arch, _ := pkg.Arch()

					kernelName := version + "." + arch
					running := false
					if kernelName == runningKernelVersion {
						running = true
					}

					res = append(res, KernelVersion{
						Name:    name,
						Version: version,
						Running: running,
					})
				}
			}
		} else if pf.Name == "photon" {
			filterKernel = func(pkg Package) {
				name, _ := pkg.Name()

				if strings.HasPrefix(name, "linux") {
					version, _ := pkg.Version()

					kernelName := version + strings.TrimPrefix(name, "linux")
					running := false
					if kernelName == runningKernelVersion {
						running = true
					}

					res = append(res, KernelVersion{
						Name:    name,
						Version: version + strings.TrimPrefix(name, "linux"),
						Running: running,
					})
				}
			}
		} else if pf.IsFamily("suse") {
			// kernel.info[version] == "4.12.14-122.23-default"
			// rpm -qa | grep -i kernel
			// kernel-default-4.12.14-122.23.1.x86_64
			// kernel-firmware-20190618-5.14.1.noarch
			// kernel-default-4.12.14-122.60.1.x86_64
			// cat /proc/version
			// Linux version 4.12.14-122.23-default (geeko@buildhost)
			filterKernel = func(pkg Package) {
				name, _ := pkg.Name()
				if strings.HasPrefix(name, "kernel-") {
					version, _ := pkg.Version()

					kernelType := strings.TrimPrefix(name, "kernel")
					running := false

					// NOTE: pkg version is 4.12.14-122.23.1 while the kernel version is 4.12.14-122.23
					if strings.HasSuffix(runningKernelVersion, kernelType) && strings.HasPrefix(version, strings.TrimSuffix(runningKernelVersion, kernelType)) {
						running = true
					}

					res = append(res, KernelVersion{
						Name:    name,
						Version: version + strings.TrimPrefix(name, "kernel"),
						Running: running,
					})
				}
			}
		}

		for i := range mqlPkgs {
			mqlPkg := mqlPkgs[i]
			pkg := mqlPkg.(Package)
			filterKernel(pkg)
		}
	}

	// empty when there is no kernel information found
	return JsonToDictSlice(res)
}

func (k *mqlKernel) GetInfo() (interface{}, error) {
	// find suitable kernel module manager
	mm, err := kernel.ResolveManager(k.MotorRuntime.Motor)
	if mm == nil || err != nil {
		return nil, errors.Join(err, errors.New("could not detect suitable kernel module manager for platform"))
	}

	// retrieve all kernel modules
	kernelInfo, err := mm.Info()
	if err != nil {
		return nil, err
	}

	return JsonToDict(kernelInfo)
}

func (k *mqlKernel) GetParameters() (map[string]interface{}, error) {
	// find suitable kernel module manager
	mm, err := kernel.ResolveManager(k.MotorRuntime.Motor)
	if mm == nil || err != nil {
		return nil, errors.Join(err, errors.New("could not detect suitable kernel module manager for platform"))
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

func (k *mqlKernel) GetModules() ([]interface{}, error) {
	// find suitable kernel module manager
	mm, err := kernel.ResolveManager(k.MotorRuntime.Motor)
	if mm == nil || err != nil {
		return nil, errors.Join(err, errors.New("could not detect suitable kernel module manager for platform"))
	}

	// retrieve all kernel modules
	kernelModules, err := mm.Modules()
	if err != nil {
		return nil, errors.Join(err, errors.New("could not retrieve kernel module list for platform"))
	}
	log.Debug().Int("modules", len(kernelModules)).Msg("[kernel.modules]> modules")

	// create MQL kernel module entry resources for each entry
	moduleEntries := make([]interface{}, len(kernelModules))
	namedMap := make(map[string]KernelModule, len(kernelModules))
	for i, kernelModule := range kernelModules {

		mqlKernelModule, err := k.MotorRuntime.CreateResource("kernel.module",
			"name", kernelModule.Name,
			"size", kernelModule.Size,
			"loaded", true,
		)
		if err != nil {
			return nil, err
		}

		moduleEntries[i] = mqlKernelModule.(KernelModule)
		namedMap[kernelModule.Name] = mqlKernelModule.(KernelModule)
	}

	k.Cache.Store("_modules", &resources.CacheEntry{Data: namedMap})

	// return the kernel modules as new entries
	return moduleEntries, nil
}

func (k *mqlKernelModule) init(args *resources.Args) (*resources.Args, KernelModule, error) {
	// TODO: look at the args and determine if we init all or ask for listing of all modules
	if len(*args) > 2 {
		return args, nil, nil
	}

	nameRaw := (*args)["name"]
	if nameRaw == nil {
		return args, nil, nil
	}
	name := nameRaw.(string)

	obj, err := k.MotorRuntime.CreateResource("kernel")
	if err != nil {
		return nil, nil, err
	}
	kernel := obj.(Kernel)

	_, err = kernel.Modules()
	if err != nil {
		return nil, nil, err
	}

	c, ok := kernel.MqlResource().Cache.Load("_modules")
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
		// (*args)["id"] = pkg.MqlResource().Id
		// Workaround: we fill in the fields we need to make the id() method
		// generate the same ID
		(*args)["size"], _ = item.Size()
		(*args)["loaded"] = true
	}

	return args, nil, nil
}

func (k *mqlKernelModule) id() (string, error) {
	return k.Name()
}
