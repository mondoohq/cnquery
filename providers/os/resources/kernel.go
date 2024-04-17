// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/kernel"
)

func initKernel(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// this resource is only supported on linux
	conn := runtime.Connection.(shared.Connection)
	platform := conn.Asset().Platform

	supported := false
	if platform.IsFamily("linux") || platform.IsFamily("darwin") || platform.Name == "freebsd" {
		supported = true
	}

	if supported == false {
		return nil, nil, errors.New("kernel resource is only supported for unix platforms")
	}

	return args, nil, nil
}

type mqlKernelInternal struct {
	moduleByName map[string]*mqlKernelModule
	lock         sync.Mutex
}

type KernelVersion struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Running bool   `json:"running"`
}

func (k *mqlKernel) installed() ([]interface{}, error) {
	res := []KernelVersion{}

	conn := k.MqlRuntime.Connection.(shared.Connection)
	platform := conn.Asset().Platform

	if platform.IsFamily(inventory.FAMILY_LINUX) {

		// 1. gather running kernel information
		info := k.GetInfo()
		if info.Error != nil {
			return nil, errors.New("could not determine kernel version")
		}

		kernelInfo, ok := info.Data.(map[string]interface{})
		if !ok {
			return nil, errors.New("no structured kernel information found")
		}

		runningKernelVersion := kernelInfo["version"].(string)

		// 2. get all packages
		raw, err := CreateResource(k.MqlRuntime, "packages", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}
		packages := raw.(*mqlPackages)

		tlist := packages.GetList()
		if tlist.Error != nil {
			return nil, tlist.Error
		}
		mqlPkgs := tlist.Data

		filterKernel := func(pkg *mqlPackage) {}

		if platform.IsFamily("debian") {
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
			filterKernel = func(pkg *mqlPackage) {
				if strings.HasPrefix(pkg.Name.Data, "linux-image") {
					kernelName := strings.TrimPrefix(pkg.Name.Data, "linux-image-")
					running := false
					if kernelName == runningKernelVersion {
						running = true
					}

					res = append(res, KernelVersion{
						Name:    kernelName,
						Version: pkg.Version.Data,
						Running: running,
					})
				}
			}
		} else if platform.IsFamily("redhat") || platform.Name == "amazonlinux" {
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
			filterKernel = func(pkg *mqlPackage) {
				if pkg.Name.Data == "kernel" {
					version := pkg.Version.Data
					arch := pkg.Arch.Data

					kernelName := version + "." + arch
					running := false
					if kernelName == runningKernelVersion {
						running = true
					}

					res = append(res, KernelVersion{
						Name:    pkg.Name.Data,
						Version: version,
						Running: running,
					})
				}
			}
		} else if platform.Name == "photon" {
			filterKernel = func(pkg *mqlPackage) {
				name := pkg.Name.Data
				if strings.HasPrefix(name, "linux") {
					version := pkg.Version.Data

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
		} else if platform.IsFamily("suse") {
			// kernel.info[version] == "4.12.14-122.23-default"
			// rpm -qa | grep -i kernel
			// kernel-default-4.12.14-122.23.1.x86_64
			// kernel-firmware-20190618-5.14.1.noarch
			// kernel-default-4.12.14-122.60.1.x86_64
			// cat /proc/version
			// Linux version 4.12.14-122.23-default (geeko@buildhost)
			filterKernel = func(pkg *mqlPackage) {
				name := pkg.Name.Data
				if strings.HasPrefix(name, "kernel-") {
					version := pkg.Version.Data

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
			pkg := mqlPkg.(*mqlPackage)
			filterKernel(pkg)
		}
	}

	// empty when there is no kernel information found
	return convert.JsonToDictSlice(res)
}

func (k *mqlKernel) info() (interface{}, error) {
	// find suitable kernel module manager
	conn := k.MqlRuntime.Connection.(shared.Connection)
	mm, err := kernel.ResolveManager(conn)
	if mm == nil || err != nil {
		return nil, errors.Wrap(err, "could not detect suitable kernel module manager for platform")
	}

	// retrieve all kernel modules
	kernelInfo, err := mm.Info()
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(kernelInfo)
}

func (k *mqlKernel) parameters() (map[string]interface{}, error) {
	// find suitable kernel module manager
	conn := k.MqlRuntime.Connection.(shared.Connection)
	mm, err := kernel.ResolveManager(conn)
	if mm == nil || err != nil {
		return nil, errors.Wrap(err, "could not detect suitable kernel module manager for platform")
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

func (k *mqlKernel) modules() ([]interface{}, error) {
	k.lock.Lock()
	defer k.lock.Unlock()

	// find suitable kernel module manager
	conn := k.MqlRuntime.Connection.(shared.Connection)
	mm, err := kernel.ResolveManager(conn)
	if mm == nil || err != nil {
		return nil, errors.Wrap(err, "could not detect suitable kernel module manager for platform")
	}

	// retrieve all kernel modules
	kernelModules, err := mm.Modules()
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve kernel module list for platform")
	}
	log.Debug().Int("modules", len(kernelModules)).Msg("[kernel.modules]> modules")

	// create MQL kernel module entry resources for each entry
	moduleEntries := make([]interface{}, len(kernelModules))
	for i, kernelModule := range kernelModules {

		raw, err := CreateResource(k.MqlRuntime, "kernel.module", map[string]*llx.RawData{
			"name":   llx.StringData(kernelModule.Name),
			"size":   llx.StringData(kernelModule.Size),
			"loaded": llx.BoolTrue,
		})
		if err != nil {
			return nil, err
		}

		moduleEntries[i] = raw.(*mqlKernelModule)
	}

	return moduleEntries, k.refreshCache(moduleEntries)
}

func (x *mqlKernel) refreshCache(all []interface{}) error {
	if all == nil {
		raw := x.GetModules()
		if raw.Error != nil {
			return raw.Error
		}
		all = raw.Data
	}

	x.moduleByName = map[string]*mqlKernelModule{}

	for i := range all {
		u := all[i].(*mqlKernelModule)
		x.moduleByName[u.Name.Data] = u
	}

	return nil
}

func initKernelModule(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	nameRaw := args["name"]
	if nameRaw == nil {
		return args, nil, nil
	}
	name := nameRaw.Value.(string)

	obj, err := CreateResource(runtime, "kernel", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	kernel := obj.(*mqlKernel)

	if err = kernel.refreshCache(nil); err != nil {
		return nil, nil, err
	}

	if res, ok := kernel.moduleByName[name]; ok {
		return nil, res, nil
	}

	res := &mqlKernelModule{}
	res.Name = plugin.TValue[string]{Data: name, State: plugin.StateIsSet}
	res.Size.State = plugin.StateIsSet | plugin.StateIsNull
	res.Loaded = plugin.TValue[bool]{Data: false, State: plugin.StateIsSet}
	return nil, res, nil
}

func (k *mqlKernelModule) id() (string, error) {
	return k.Name.Data, nil
}
