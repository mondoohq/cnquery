// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"bytes"
	"context"
	"os"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"go.mondoo.com/cnquery/v11/explorer/resources"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/recording"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/sbom"
	"go.mondoo.com/cnquery/v11/types"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

var sbomProvider = Provider{
	Provider: &plugin.Provider{
		Name:    "sbom",
		ID:      "go.mondoo.com/cnquery/v11/providers/sbom",
		Version: "11.0.0",
		Connectors: []plugin.Connector{
			{
				Name:    "sbom",
				Use:     "sbom [flags]",
				Short:   "read SBOM file on disk",
				MinArgs: 1,
				MaxArgs: 1,
				Flags: []plugin.Flag{
					{
						Long:    "format",
						Type:    plugin.FlagType_String,
						Default: "",
						Desc:    "Format of the sbom file (default is auto-detect).",
					},
				},
			},
		},
	},
}

type sbomProviderService struct {
	initialized bool
	runtime     *Runtime
	recording   *recording.Asset
}

func (s *sbomProviderService) Heartbeat(req *plugin.HeartbeatReq) (*plugin.HeartbeatRes, error) {
	return nil, nil
}

func (s *sbomProviderService) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	filePath := req.Args[0]

	return &plugin.ParseCLIRes{
		Asset: &inventory.Asset{
			Connections: []*inventory.Config{
				{
					Type: "sbom",
					Path: filePath,
				},
			},
		},
	}, nil
}

func (s *sbomProviderService) Connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing config")
	}

	// load file
	// TODO: make more robust
	asset := req.Asset
	conf := req.Asset.Connections[0]

	// convert sbom to recording
	data, err := os.ReadFile(conf.Path)
	if err != nil {
		return nil, err
	}

	decoder := []sbom.Decoder{}

	decoder = append(decoder,
		sbom.NewCycloneDX(sbom.FormatCycloneDxJSON),
		sbom.NewCycloneDX(sbom.FormatCycloneDxXML),
		sbom.NewSPDX(sbom.FormatSpdxTagValue),
		sbom.NewSPDX(sbom.FormatSpdxJSON),
	)

	var sbomReport *sbom.Sbom
	found := false
	for i := range decoder {
		sbomReport, err = decoder[i].Parse(bytes.NewReader(data))
		if err == nil {
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("unsupported sbom format")
	}

	// Platform need to be set, otherwise it will panic
	asset.Platform = &inventory.Platform{}
	if sbomReport.Asset.Platform != nil {
		asset.Name = sbomReport.Asset.Name
		asset.PlatformIds = sbomReport.Asset.PlatformIds
		asset.Platform = &inventory.Platform{
			Title:   sbomReport.Asset.Platform.Title,
			Name:    sbomReport.Asset.Platform.Name,
			Version: sbomReport.Asset.Platform.Version,
			Build:   sbomReport.Asset.Platform.Build,
			Arch:    sbomReport.Asset.Platform.Arch,
			Family:  sbomReport.Asset.Platform.Family,
		}
		asset.PlatformIds = []string{"//platformid.api.mondoo.app/runtime/sbom/uuid/" + uuid.New().String()}
	}

	// generate recording from sbom and store it into the provider
	s.recording, err = newRecording(asset, sbomReport)
	if err != nil {
		return nil, err
	}

	// set the recording
	ctx := context.Background()
	urecording, err := recording.NewUpstreamRecording(ctx, s, req.Asset.Mrn)
	if err != nil {
		return nil, err
	}

	providerName := "os"
	provider := Coordinator.Providers().Lookup(ProviderLookup{ProviderName: providerName})
	if provider == nil {
		return nil, errors.New("failed to look up provider for upstream recording with name=" + providerName)
	}

	addedProvider, err := s.runtime.addProvider(provider.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init provider for connection in recording")
	}

	conn, err := addedProvider.Instance.Plugin.MockConnect(&plugin.ConnectReq{
		Asset:    asset,
		Features: req.Features,
		Upstream: req.Upstream,
	}, callback)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init referenced provider")
	}

	// overwrite the mock type
	conn.Asset.Connections[0].Type = "sbom"
	addedProvider.Connection = conn
	err = s.runtime.SetRecording(urecording)
	return conn, err
}

func (s *sbomProviderService) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	// Should never happen: the mock provider should not be called with MockConnect.
	// It is the only thing that should ever call MockConnect to other providers
	// (outside of tests).
	return nil, errors.New("the sbom provider does not support the mock connect call, this is an internal error")
}

func (s *sbomProviderService) Disconnect(req *plugin.DisconnectReq) (*plugin.DisconnectRes, error) {
	// Nothing to do yet...
	return nil, nil
}

func (s *sbomProviderService) Shutdown(req *plugin.ShutdownReq) (*plugin.ShutdownRes, error) {
	// Nothing to do yet...
	return nil, nil
}

func (s *sbomProviderService) GetData(req *plugin.DataReq) (*plugin.DataRes, error) {
	panic("NO")
}

func (s *sbomProviderService) StoreData(req *plugin.StoreReq) (*plugin.StoreRes, error) {
	panic("NO")
}

func (s *sbomProviderService) Init(running *RunningProvider) {
	if s.initialized {
		return
	}
	s.initialized = true

	// TODO: Currently not needed, as the coordinator loads all schemas right now.
	// Once it doesn't do that anymore, remember to load all schemas here
	// rt.schema.unsafeLoadAll()
	// rt.schema.unsafeRefresh()
}

// implement resource explorer interface
func (s *sbomProviderService) GetResourcesData(ctx context.Context, req *resources.EntityResourcesReq) (*resources.EntityResourcesRes, error) {
	list := []*llx.ResourceRecording{}
	for i := range req.Resources {
		request := req.Resources[i]

		resource, ok := s.recording.GetResource(request.Resource, request.Id)
		if ok {
			res := &llx.ResourceRecording{
				Resource: request.Resource,
				Id:       request.Id,
				Fields:   make(map[string]*llx.Result, len(resource.Fields)),
			}

			for k, v := range resource.Fields {
				res.Fields[k] = v.Result()
			}
			list = append(list, res)
		}
	}

	return &resources.EntityResourcesRes{
		EntityMrn: req.EntityMrn,
		Resources: list,
	}, nil
}

func (s *sbomProviderService) ListResources(ctx context.Context, req *resources.ListResourcesReq) (*resources.ListResourcesRes, error) {
	list := make([]*llx.ResourceRecording, len(s.recording.Resources))
	for i := range s.recording.Resources {
		cur := s.recording.Resources[i]
		list[i] = &llx.ResourceRecording{
			Resource: cur.Resource,
			Id:       cur.ID,
		}
	}
	return &resources.ListResourcesRes{
		EntityMrn: req.EntityMrn,
		Resources: list,
	}, nil
}

func newRecording(asset *inventory.Asset, s *sbom.Sbom) (*recording.Asset, error) {
	r := recording.NewAssetRecording(asset)

	// add assetRecording resource information
	r.Resources = append(r.Resources, recording.Resource{
		Resource: "asset",
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
				Type:  types.Array(types.String),
				Value: convert.SliceAnyToInterface(s.Asset.Platform.Family),
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
