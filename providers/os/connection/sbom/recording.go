// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"errors"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/recording"
	"go.mondoo.com/cnquery/v11/sbom"
	"go.mondoo.com/cnquery/v11/types"
)

func newRecordingCallbacks(recording *recording.Asset) *recordingCallbacks {
	return &recordingCallbacks{
		recording: recording,
	}
}

type recordingCallbacks struct {
	recording *recording.Asset
}

func (p *recordingCallbacks) GetRecording(req *plugin.DataReq) (*plugin.ResourceData, error) {
	resource, ok := p.recording.GetResource(req.Resource, req.ResourceId)
	if !ok {
		return nil, nil
	}

	res := plugin.ResourceData{
		Name:   req.Resource,
		Id:     req.ResourceId,
		Fields: make(map[string]*llx.Result, len(resource.Fields)),
	}
	for k, v := range resource.Fields {
		res.Fields[k] = v.Result()
	}

	return &res, nil
}

func (p *recordingCallbacks) GetData(req *plugin.DataReq) (*plugin.DataRes, error) {
	return nil, errors.New("not implemented")
}

func (p *recordingCallbacks) Collect(req *plugin.DataRes) error {
	return errors.New("not implemented")
}

func newRecording(asset *inventory.Asset, s *sbom.Sbom) (*recording.Asset, error) {
	r := recording.NewAssetRecording(asset)

	// add assetRecording resource information
	r.Resources = append(r.Resources, recording.Resource{
		Resource: "assetRecording",
		ID:       "",
		Fields: map[string]*llx.RawData{
			"name": {
				Type:  types.String,
				Value: s.Asset.Name,
			},
			"title": {
				Type:  types.String,
				Value: s.Asset.Platform.Title,
			},
			"platform": {
				Type:  types.String,
				Value: s.Asset.Platform.Name,
			},
			"version": {
				Type:  types.String,
				Value: s.Asset.Platform.Version,
			},
			"build": {
				Type:  types.String,
				Value: s.Asset.Platform.Build,
			},
			"arch": {
				Type:  types.String,
				Value: s.Asset.Platform.Arch,
			},
			"family": {
				Type:  types.StringSlice,
				Value: s.Asset.Platform.Family,
			},
			"kind": {
				Type:  types.String,
				Value: "container-image", // TODO: adjust based on device type
			},
			"runtime": {
				Type:  types.String,
				Value: "container", // TODO: adjust based on device type
			},
		},
	})

	osPackageReferences := []interface{}{}
	npmPackageReferences := []interface{}{}
	pythonPackageReferences := []interface{}{}

	for i := range s.Packages {
		pkg := s.Packages[i]

		if pkg.Type == "deb" || pkg.Type == "rpm" || pkg.Type == "apk" {
			pkgResource, otherResources := newOsPackage(pkg)
			r.Resources = append(r.Resources, pkgResource)
			r.Resources = append(r.Resources, otherResources...)
			osPackageReferences = append(osPackageReferences, &llx.MockResource{
				Name: "package",
				ID:   pkgResource.ID,
			})
		}

		if pkg.Type == "npm" {
			pkgResource := newNpmPackage(pkg)
			r.Resources = append(r.Resources, pkgResource)
			npmPackageReferences = append(npmPackageReferences, &llx.MockResource{
				Name: "package",
				ID:   pkgResource.ID,
			})
		}

		if pkg.Type == "pypi" {
			pkgResource := newPythonPackage(pkg)
			r.Resources = append(r.Resources, pkgResource)
			pythonPackageReferences = append(pythonPackageReferences, &llx.MockResource{
				Name: "package",
				ID:   pkgResource.ID,
			})
		}
	}

	r.Resources = append(r.Resources, recording.Resource{
		Resource: "packages",
		ID:       "",
		Fields: map[string]*llx.RawData{
			"list": {
				Type:  types.Array(types.Resource("package")),
				Value: osPackageReferences,
			},
		},
	})

	// npm packages
	r.Resources = append(r.Resources, recording.Resource{
		Resource: "npm.packages",
		ID:       "npm.packages",
		Fields: map[string]*llx.RawData{
			"list": {
				Type:  types.Array(types.Resource("npm.package")),
				Value: npmPackageReferences,
			},
		},
	})

	// python packages
	r.Resources = append(r.Resources, recording.Resource{
		Resource: "python",
		ID:       "python",
		Fields: map[string]*llx.RawData{
			"packages": {
				Type:  types.Array(types.Resource("python.package")),
				Value: pythonPackageReferences,
			},
		},
	})

	r.Resources = append(r.Resources, recording.Resource{
		Resource: "kernel",
		ID:       "",
		Fields: map[string]*llx.RawData{
			"installed": {
				Type:  types.Array(types.Dict),
				Error: errors.New("could not determine kernel version"),
			},
		},
	})

	r.RefreshCache()
	return r, nil
}

func newOsPackage(pkg *sbom.Package) (recording.Resource, []recording.Resource) {
	resources := []recording.Resource{}
	cpeReferences := []interface{}{}

	for _, cpe := range pkg.Cpes {
		cpeResource := newCpeResource(cpe)
		resources = append(resources, cpeResource)
		cpeReferences = append(cpeReferences, &llx.MockResource{
			Name: "cpe",
			ID:   cpeResource.ID,
		})
	}

	return recording.Resource{
		Resource: "package",
		ID:       pkg.Purl,
		Fields: map[string]*llx.RawData{
			"name": {
				Type:  types.String,
				Value: pkg.Name,
			},
			"version": {
				Type:  types.String,
				Value: pkg.Version,
			},
			"purl": {
				Type:  types.String,
				Value: pkg.Purl,
			},
			"arch": {
				Type:  types.String,
				Value: pkg.Architecture,
			},
			"format": {
				Type:  types.String,
				Value: pkg.Type,
			},
			"origin": {
				Type:  types.String,
				Value: pkg.Origin,
			},
			"cpes": {
				Type:  types.Array(types.Resource("cpe")),
				Value: cpeReferences,
			},
			"files": {
				Type:  types.Array(types.Resource("pkgFileInfo")),
				Value: []interface{}{},
			},
		},
	}, resources
}

func newNpmPackage(pkg *sbom.Package) recording.Resource {
	return recording.Resource{
		Resource: "npm.package",
		ID:       pkg.Purl,
		Fields: map[string]*llx.RawData{
			"id": {
				Type:  types.String,
				Value: pkg.Purl,
			},
			"name": {
				Type:  types.String,
				Value: pkg.Name,
			},
			"version": {
				Type:  types.String,
				Value: pkg.Version,
			},
			"purl": {
				Type:  types.String,
				Value: pkg.Purl,
			},
			"cpes": {
				Type: types.Array(types.Resource("cpe")),
				// TODO: add support for CPEs
				Value: []interface{}{},
			},
			"files": {
				Type:  types.Array(types.Resource("pkgFileInfo")),
				Value: []interface{}{},
			},
		},
	}
}

func newPythonPackage(pkg *sbom.Package) recording.Resource {
	return recording.Resource{
		Resource: "python.package",
		// TODO: we should use the purl as the ID in the resource as well
		ID: pkg.Purl,
		Fields: map[string]*llx.RawData{
			"id": {
				Type:  types.String,
				Value: pkg.Purl,
			},
			"file": {
				Type:  types.Resource("file"),
				Value: nil,
			},
			"name": {
				Type:  types.String,
				Value: pkg.Name,
			},
			"version": {
				Type:  types.String,
				Value: pkg.Version,
			},
			"license": {
				Type:  types.String,
				Value: "",
			},
			"author": {
				Type:  types.String,
				Value: "",
			},
			"authorEmail": {
				Type:  types.String,
				Value: "",
			},
			"summary": {
				Type:  types.String,
				Value: "",
			},
			"purl": {
				Type:  types.String,
				Value: pkg.Purl,
			},
			"cpes": {
				Type: types.Array(types.Resource("cpe")),
				// TODO: add support for CPEs
				Value: []interface{}{},
			},
			"files": {
				Type:  types.Array(types.Resource("pkgFileInfo")),
				Value: []interface{}{},
			},
			"dependencies": {
				Type:  types.Array(types.Resource("python.package")),
				Value: []interface{}{},
			},
		},
	}
}

func newCpeResource(cpe string) recording.Resource {
	return recording.Resource{
		Resource: "cpe",
		ID:       cpe,
		Fields: map[string]*llx.RawData{
			"uri": {
				Type:  types.String,
				Value: cpe,
			},
		},
	}
}
