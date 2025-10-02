// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package recording

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/types"
	"go.mondoo.com/cnquery/v12/utils/multierr"
)

type recording struct {
	Assets []*Asset `json:"assets"`
	Path   string   `json:"-"`
	// assets is used for fast connection to asset lookup
	assets          map[uint32]*Asset `json:"-"`
	prettyPrintJSON bool              `json:"-"`
	// this mode is used when we use the recording layer for data,
	// but not for storing it on disk
	doNotSave bool `json:"-"`
}

func NewAssetRecording(asset *inventory.Asset) *Asset {
	id := asset.Mrn
	if id == "" {
		id = asset.Id
	}
	if id == "" && asset.Platform != nil {
		id = asset.Platform.Title
	}
	ai := &assetInfo{
		ID:          id,
		Name:        asset.Name,
		PlatformIDs: asset.PlatformIds,
		Annotations: asset.Annotations,
	}

	if asset.Platform != nil {
		ai.Arch = asset.Platform.Arch
		ai.Title = asset.Platform.Title
		ai.Family = asset.Platform.Family
		ai.Build = asset.Platform.Build
		ai.Version = asset.Platform.Version
		ai.Kind = asset.Platform.Kind
		ai.Runtime = asset.Platform.Runtime
		ai.Labels = asset.Platform.Labels
	}
	return &Asset{
		Asset:       ai,
		connections: map[string]*connection{},
		resources:   map[string]*Resource{},
	}
}

// Creates a recording that holds only the specified asset
func FromAsset(asset *inventory.Asset) (*recording, error) {
	r := &recording{
		Assets: []*Asset{
			NewAssetRecording(asset),
		},
	}

	r.refreshCache()
	if err := r.reconnectResources(); err != nil {
		return nil, err
	}

	return r, nil
}

// ReadOnly converts the recording into a read-only recording
func (r *recording) ReadOnly() *readOnly {
	return &readOnly{r}
}

type readOnly struct {
	*recording
}

func (n *readOnly) Save() error {
	return nil
}

func (n *readOnly) EnsureAsset(asset *inventory.Asset, provider string, connectionID uint32, conf *inventory.Config) {
	// For read-only recordings we are still loading from file, so that means
	// we are severely lacking connection IDs.
	existing := n.getExistingAsset(asset)
	if existing != nil {
		n.assets[connectionID] = existing
	}
}

func (n *readOnly) AddData(connectionID uint32, resource string, id string, field string, data *llx.RawData) {
}

type RecordingOptions struct {
	DoRecord        bool
	PrettyPrintJSON bool
	DoNotSave       bool
}

// NewWithFile loads and creates a new recording based on user settings.
// If no file is available and users don't wish to record, it throws an error.
// If users don't wish to record and no recording is available, it will return
// the null-recording.
func NewWithFile(path string, opts RecordingOptions) (llx.Recording, error) {
	if path == "" && !opts.DoNotSave {
		// we don't want to record and we don't want to load a recording path...
		// so there is nothing to do, so return nil
		if !opts.DoRecord {
			return Null{}, nil
		}
		// for all remaining cases we do want to record and we want to check
		// if the recording exists at the default location
		path = "recording.json"
	}

	if _, err := os.Stat(path); err == nil {
		res, err := LoadRecordingFile(path)
		if err != nil {
			return nil, multierr.Wrap(err, "failed to load recording")
		}
		res.Path = path

		if opts.DoRecord {
			res.prettyPrintJSON = opts.PrettyPrintJSON
			return res, nil
		}
		return &readOnly{res}, nil

	} else if errors.Is(err, os.ErrNotExist) {
		if opts.DoRecord {
			res := &recording{
				Path:            path,
				prettyPrintJSON: opts.PrettyPrintJSON,
				doNotSave:       opts.DoNotSave,
				assets:          map[uint32]*Asset{},
			}
			res.refreshCache() // only for initialization
			return res, nil
		}
		return nil, errors.New("failed to load recording: '" + path + "' does not exist")

	} else {
		// Schrodinger's file, may be permissions or something else...
		return nil, multierr.Wrap(err, "failed to access recording in '"+path+"'")
	}
}

func LoadRecordingFile(path string) (*recording, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var res recording
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return nil, err
	}

	pres := &res
	pres.refreshCache()

	if err = pres.reconnectResources(); err != nil {
		return nil, err
	}

	return pres, err
}

func (r *recording) Save() error {
	r.finalize()
	if r.doNotSave {
		return nil
	}

	var raw []byte
	var err error
	if r.prettyPrintJSON {
		raw, err = json.MarshalIndent(r, "", "  ")
	} else {
		raw, err = json.Marshal(r)
	}
	if err != nil {
		return multierr.Wrap(err, "failed to marshal json for recording")
	}

	if err := os.WriteFile(r.Path, raw, 0o644); err != nil {
		return multierr.Wrap(err, "failed to store recording")
	}

	log.Info().Msg("stored recording in " + r.Path)
	return nil
}

func (r *recording) refreshCache() {
	r.assets = make(map[uint32]*Asset, len(r.Assets))
	for i := range r.Assets {
		asset := r.Assets[i]
		asset.RefreshCache()

		for _, conn := range asset.Connections {
			// only connection ID's != 0 are valid IDs. We get lots of 0 when we
			// initially load this object, so we won't know yet which asset belongs
			// to which connection.
			if conn.Id != 0 {
				r.assets[conn.Id] = asset
			}
		}
	}
}

func (r *recording) reconnectResources() error {
	var err error
	for i := range r.Assets {
		asset := r.Assets[i]
		for j := range asset.Resources {
			if err = r.reconnectResource(&asset.Resources[j]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *recording) reconnectResource(resource *Resource) error {
	var err error
	for k, v := range resource.Fields {
		if v.Error != nil {
			// in this case we have neither type information nor a value
			resource.Fields[k].Error = v.Error
			continue
		}

		typ := types.Type(v.Type)
		resource.Fields[k].Value, err = tryReconnect(typ, v.Value, resource)
		if err != nil {
			return err
		}
	}
	return nil
}

func tryReconnect(typ types.Type, v any, resource *Resource) (any, error) {
	var err error

	if typ.IsArray() {
		arr, ok := v.([]any)
		if !ok {
			return nil, errors.New("failed to reconnect array type")
		}
		ct := typ.Child()
		for i := range arr {
			arr[i], err = tryReconnect(ct, arr[i], resource)
			if err != nil {
				return nil, err
			}
		}
		return arr, nil
	}

	if typ.IsMap() {
		m, ok := v.(map[string]any)
		if !ok {
			return nil, errors.New("failed to reconnect map type")
		}
		ct := typ.Child()
		for i := range m {
			m[i], err = tryReconnect(ct, m[i], resource)
			if err != nil {
				return nil, err
			}
		}
		return m, nil
	}

	if !typ.IsResource() || v == nil {
		return v, nil
	}

	return reconnectResource(v, resource)
}

func reconnectResource(v any, resource *Resource) (any, error) {
	vals, ok := v.(map[string]any)
	if !ok {
		return nil, errors.New("error in recording: resource '" + resource.Resource + "' (ID:" + resource.ID + ") has incorrect reference")
	}
	name, ok := vals["Name"].(string)
	if !ok {
		return nil, errors.New("error in recording: resource '" + resource.Resource + "' (ID:" + resource.ID + ") has incorrect type in Name field")
	}
	id, ok := vals["ID"].(string)
	if !ok {
		return nil, errors.New("error in recording: resource '" + resource.Resource + "' (ID:" + resource.ID + ") has incorrect type in ID field")
	}

	// TODO: Not sure yet if we need to check the recording for the reference.
	// Unless it is used by the code, we may get away with it.
	// if _, ok = asset.resources[name+"\x00"+id]; !ok {
	// 	return errors.New("cannot find resource '" + resource.Resource + "' (ID:" + resource.ID + ") in recording")
	// }

	return &llx.MockResource{Name: name, ID: id}, nil
}

func (r *recording) finalize() {
	for i := range r.Assets {
		r.Assets[i].finalize()
	}
}

func (r *recording) getExistingAsset(asset *inventory.Asset) *Asset {
	for _, existing := range r.Assets {
		id := existing.Asset.ID
		if id == asset.Mrn {
			return existing
		}

		for _, pidExisting := range existing.Asset.PlatformIDs {
			if slices.Contains(asset.PlatformIds, pidExisting) {
				return existing
			}
		}
	}

	return nil
}

func (r *recording) EnsureAsset(asset *inventory.Asset, providerID string, connectionID uint32, conf *inventory.Config) {
	if asset.Platform == nil {
		log.Warn().Msg("cannot store asset in recording, asset has no platform")
		return
	}
	id := getAssetIdForRecording(asset)
	if id == "" {
		log.Debug().Msg("cannot store asset in recording, asset has no mrn or platform ids")
		return
	}
	recordingAsset := r.getExistingAsset(asset)
	if recordingAsset == nil {
		recordingAsset = &Asset{
			Asset: &assetInfo{
				PlatformIDs: asset.PlatformIds,
				Name:        asset.Platform.Name,
				Arch:        asset.Platform.Arch,
				Title:       asset.Platform.Title,
				Family:      asset.Platform.Family,
				Build:       asset.Platform.Build,
				Version:     asset.Platform.Version,
				Kind:        asset.Platform.Kind,
				Runtime:     asset.Platform.Runtime,
				Labels:      asset.Platform.Labels,
			},
			connections: map[string]*connection{},
			resources:   map[string]*Resource{},
		}
		r.Assets = append(r.Assets, recordingAsset)
	}
	// always update the id for the asset, sometimes we get assets by id and then
	// they get updated with an MRN attached
	recordingAsset.Asset.ID = getAssetIdForRecording(asset)
	if len(asset.PlatformIds) != 0 {
		recordingAsset.Asset.PlatformIDs = asset.PlatformIds
	}

	recordingAsset.connections[fmt.Sprintf("%d", conf.Id)] = &connection{
		Url:        conf.ToUrl(),
		ProviderID: providerID,
		Connector:  conf.Type,
		Id:         conf.Id,
	}
	r.assets[connectionID] = recordingAsset
}

func (r *recording) AddData(connectionID uint32, resource string, id string, field string, data *llx.RawData) {
	asset, ok := r.assets[connectionID]
	if !ok {
		return
	}

	obj, exist := asset.resources[resource+"\x00"+id]
	if !exist {
		obj = &Resource{
			Resource: resource,
			ID:       id,
			Fields:   map[string]*llx.RawData{},
		}
		asset.resources[resource+"\x00"+id] = obj
	}

	if field != "" {
		obj.Fields[field] = data
	}
}

func (r *recording) GetData(connectionID uint32, resource string, id string, field string) (*llx.RawData, bool) {
	asset, ok := r.assets[connectionID]
	if !ok {
		return nil, false
	}

	obj, exist := asset.resources[resource+"\x00"+id]
	if !exist {
		return nil, false
	}

	if field == "" {
		return &llx.RawData{Type: types.Resource(resource), Value: id}, true
	}

	data, ok := obj.Fields[field]
	if !ok && field == "id" {
		return llx.StringData(id), true
	}

	return data, ok
}

func (r *recording) GetResource(connectionID uint32, resource string, id string) (map[string]*llx.RawData, bool) {
	asset, ok := r.assets[connectionID]
	if !ok {
		return nil, false
	}

	obj, exist := asset.resources[resource+"\x00"+id]
	if !exist {
		return nil, false
	}

	return obj.Fields, true
}

func (r *recording) GetAssetData(assetMrn string) (map[string]*llx.ResourceRecording, bool) {
	var cur *Asset
	for i := range r.Assets {
		cur = r.Assets[i]
		if assetMrn == "" && len(r.Assets) == 1 {
			// next
		} else if cur.Asset.ID != assetMrn {
			continue
		}

		ensureAssetMetadata(cur.resources, cur.Asset)

		res := make(map[string]*llx.ResourceRecording, len(cur.resources))
		for k, v := range cur.resources {
			fields := make(map[string]*llx.Result, len(v.Fields))
			for fk, fv := range v.Fields {
				fields[fk] = fv.Result()
			}
			res[k] = &llx.ResourceRecording{
				Resource: v.Resource,
				Id:       v.ID,
				Fields:   fields,
			}
		}

		return res, true
	}
	return nil, false
}

func (r *recording) GetAssetRecordings() []*Asset {
	return r.Assets
}

func (r *recording) SetAssetRecording(id uint32, reco *Asset) {
	r.assets[id] = reco
}

// This method makes sure the asset metadata is always included in the data
// dump of a recording
func ensureAssetMetadata(resources map[string]*Resource, asset *assetInfo) {
	id := "asset\x00"
	existing, ok := resources[id]
	if !ok {
		existing = &Resource{
			Resource: "asset",
			ID:       "",
			Fields:   map[string]*llx.RawData{},
		}
		resources[id] = existing
	}

	if _, ok := existing.Fields["platform"]; !ok {
		existing.Fields["platform"] = llx.StringData(asset.Name)
	}
	if _, ok := existing.Fields["version"]; !ok {
		existing.Fields["version"] = llx.StringData(asset.Version)
	}
	if _, ok := existing.Fields["kind"]; !ok {
		existing.Fields["kind"] = llx.StringData(asset.Kind)
	}
	if _, ok := existing.Fields["runtime"]; !ok {
		existing.Fields["runtime"] = llx.StringData(asset.Runtime)
	}
	if _, ok := existing.Fields["arch"]; !ok {
		existing.Fields["arch"] = llx.StringData(asset.Arch)
	}
	if _, ok := existing.Fields["title"]; !ok {
		existing.Fields["title"] = llx.StringData(asset.Title)
	}

	if _, ok := existing.Fields["ids"]; !ok {
		arr := make([]any, len(asset.PlatformIDs))
		for i := range asset.PlatformIDs {
			arr[i] = asset.PlatformIDs[i]
		}
		existing.Fields["ids"] = llx.ArrayData(arr, types.String)
	}
}

func (a assetInfo) ToInventory() *inventory.Asset {
	return &inventory.Asset{
		Id:          a.ID,
		Name:        a.Name,
		Labels:      a.Labels,
		PlatformIds: a.PlatformIDs,
		Annotations: a.Annotations,
		Platform: &inventory.Platform{
			Name:    a.Name,
			Arch:    a.Arch,
			Title:   a.Title,
			Family:  a.Family,
			Build:   a.Build,
			Version: a.Version,
			Kind:    a.Kind,
			Runtime: a.Runtime,
			Labels:  a.Labels,
		},
	}
}

func RawDataArgsToResultArgs(args map[string]*llx.RawData) (map[string]*llx.Result, error) {
	all := make(map[string]*llx.Result, len(args))
	var err multierr.Errors
	for k, v := range args {
		res := v.Result()
		if res.Error != "" {
			err.Add(errors.New("failed to convert '" + k + "': " + res.Error))
		} else {
			all[k] = res
		}
	}

	return all, err.Deduplicate()
}

func getAssetIdForRecording(asset *inventory.Asset) string {
	id := asset.Mrn
	if id == "" && len(asset.PlatformIds) > 0 {
		id = asset.PlatformIds[0]
	}

	return id
}
