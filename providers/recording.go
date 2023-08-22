// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"encoding/json"
	"errors"
	"os"
	"sort"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/types"
	"go.mondoo.com/cnquery/utils/multierr"
)

type Recording interface {
	Save() error
	EnsureAsset(asset *inventory.Asset, provider string, connectionID uint32, conf *inventory.Config)
	AddData(connectionID uint32, resource string, id string, field string, data *llx.RawData)
	GetData(connectionID uint32, resource string, id string, field string) (*llx.RawData, bool)
	GetResource(connectionID uint32, resource string, id string) (map[string]*llx.RawData, bool)
}

type recording struct {
	Assets []assetRecording `json:"assets"`
	Path   string           `json:"-"`
	// assets is used for fast connection to asset lookup
	assets          map[uint32]*assetRecording `json:"-"`
	prettyPrintJSON bool                       `json:"-"`
}

type assetRecording struct {
	Asset       assetInfo             `json:"asset"`
	Connections []connectionRecording `json:"connections"`
	Resources   []resourceRecording   `json:"resources"`

	connections map[string]*connectionRecording `json:"-"`
	resources   map[string]*resourceRecording   `json:"-"`
}

type assetInfo struct {
	ID          string            `json:"id"`
	PlatformIDs []string          `json:"platformIDs,omitempty"`
	Name        string            `json:"name,omitempty"`
	Arch        string            `json:"arch,omitempty"`
	Title       string            `json:"title,omitempty"`
	Family      []string          `json:"family,omitempty"`
	Build       string            `json:"build,omitempty"`
	Version     string            `json:"version,omitempty"`
	Kind        string            `json:"kind,omitempty"`
	Runtime     string            `json:"runtime,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type connectionRecording struct {
	Url       string `json:"url"`
	Provider  string `json:"provider"`
	Connector string `json:"connector"`
	Version   string `json:"version"`
	id        uint32 `json:"-"`
}

type resourceRecording struct {
	Resource string
	ID       string
	Fields   map[string]*llx.RawData
}

type nullRecording struct{}

func (n nullRecording) Save() error {
	return nil
}

func (n nullRecording) EnsureAsset(asset *inventory.Asset, provider string, connectionID uint32, conf *inventory.Config) {
}

func (n nullRecording) AddData(connectionID uint32, resource string, id string, field string, data *llx.RawData) {
}

func (n nullRecording) GetData(connectionID uint32, resource string, id string, field string) (*llx.RawData, bool) {
	return nil, false
}

func (n nullRecording) GetResource(connectionID uint32, resource string, id string) (map[string]*llx.RawData, bool) {
	return nil, false
}

type readOnlyRecording struct {
	*recording
}

func (n *readOnlyRecording) Save() error {
	return nil
}

func (n *readOnlyRecording) EnsureAsset(asset *inventory.Asset, provider string, connectionID uint32, conf *inventory.Config) {
	// For read-only recordings we are still loading from file, so that means
	// we are severly lacking connection IDs.
	found, _ := n.findAssetConnID(asset, conf)
	if found != -1 {
		n.assets[connectionID] = &n.Assets[found]
	}
}

func (n *readOnlyRecording) AddData(connectionID uint32, resource string, id string, field string, data *llx.RawData) {
}

type RecordingOptions struct {
	DoRecord        bool
	PrettyPrintJSON bool
}

// NewRecording loads and creates a new recording based on user settings.
// If no recording is available and users don't wish to record, it throws an error.
// If users don't wish to record and no recording is available, it will return
// the null-recording.
func NewRecording(path string, opts RecordingOptions) (Recording, error) {
	if path == "" {
		// we don't want to record and we don't want to load a recording path...
		// so there is nothing to do, so return nil
		if !opts.DoRecord {
			return nullRecording{}, nil
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
		return &readOnlyRecording{res}, nil

	} else if errors.Is(err, os.ErrNotExist) {
		if opts.DoRecord {
			res := &recording{
				Path:            path,
				prettyPrintJSON: opts.PrettyPrintJSON,
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
	pres.fixTypes()

	if err = pres.reconnectResources(); err != nil {
		return nil, err
	}

	return pres, err
}

func (r *recording) Save() error {
	r.finalize()

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
	r.assets = make(map[uint32]*assetRecording, len(r.Assets))
	for i := range r.Assets {
		asset := &r.Assets[i]
		asset.resources = make(map[string]*resourceRecording, len(asset.Resources))
		asset.connections = make(map[string]*connectionRecording, len(asset.Connections))

		for j := range asset.Resources {
			resource := &asset.Resources[j]
			asset.resources[resource.Resource+"\x00"+resource.ID] = resource
		}

		for j := range asset.Connections {
			conn := &asset.Connections[j]
			asset.connections[conn.Url] = conn

			// only connection ID's != 0 are valid IDs. We get lots of 0 when we
			// initially load this object, so we won't know yet which asset belongs
			// to which connection.
			if conn.id != 0 {
				r.assets[conn.id] = asset
			}
		}
	}
}

// json during the unmarshal step will load some things in a way that we
// can't process. For example: numbers are loaded as float64, but can also
// be int64's in MQL. This fixes the loaded types.
func (r *recording) fixTypes() {
	for i := range r.Assets {
		asset := r.Assets[i]
		for j := range asset.Resources {
			fixResourceTypes(&asset.Resources[j])
		}
	}
}

func fixResourceTypes(r *resourceRecording) {
	for _, v := range r.Fields {
		if v.Type == types.Int {
			v.Value = int64(v.Value.(float64))
		}
	}
}

func (r *recording) reconnectResources() error {
	var err error
	for i := range r.Assets {
		asset := r.Assets[i]
		for j := range asset.Resources {
			if err = r.reconnectResource(&asset, &asset.Resources[j]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *recording) reconnectResource(asset *assetRecording, resource *resourceRecording) error {
	var err error
	for k, v := range resource.Fields {
		typ := types.Type(v.Type)
		resource.Fields[k].Value, err = tryReconnect(typ, v.Value, resource)
		if err != nil {
			return err
		}
	}
	return nil
}

func tryReconnect(typ types.Type, v interface{}, resource *resourceRecording) (interface{}, error) {
	var err error

	if typ.IsArray() {
		arr, ok := v.([]interface{})
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
		m, ok := v.(map[string]interface{})
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

func reconnectResource(v interface{}, resource *resourceRecording) (interface{}, error) {
	vals, ok := v.(map[string]interface{})
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
		asset := &r.Assets[i]
		asset.Resources = make([]resourceRecording, len(asset.resources))
		asset.Connections = make([]connectionRecording, len(asset.connections))

		i := 0
		for _, v := range asset.resources {
			asset.Resources[i] = *v
			i++
		}

		sort.Slice(asset.Resources, func(i, j int) bool {
			a := asset.Resources[i]
			b := asset.Resources[j]
			if a.Resource == b.Resource {
				return a.ID < b.ID
			}
			return a.Resource < b.Resource
		})

		i = 0
		for _, v := range asset.connections {
			asset.Connections[i] = *v
			i++
		}
	}
}

func (r *recording) findAssetConnID(asset *inventory.Asset, conf *inventory.Config) (int, string) {
	var id string
	if asset.Mrn != "" {
		id = asset.Mrn
	} else if asset.Id != "" {
		id = asset.Id
	} else if asset.Platform != nil {
		id = asset.Platform.Title
	}

	found := -1
	for i := range r.Assets {
		if r.Assets[i].Asset.ID == id {
			found = i
			break
		}
	}

	return found, id
}

func (r *recording) EnsureAsset(asset *inventory.Asset, provider string, connectionID uint32, conf *inventory.Config) {
	found, _ := r.findAssetConnID(asset, conf)

	if found == -1 {
		id := asset.Mrn
		if id == "" {
			id = asset.Id
		}
		if id == "" {
			id = asset.Platform.Title
		}
		r.Assets = append(r.Assets, assetRecording{
			Asset: assetInfo{
				ID:          id,
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
			connections: map[string]*connectionRecording{},
			resources:   map[string]*resourceRecording{},
		})
		found = len(r.Assets) - 1
	}

	assetObj := &r.Assets[found]

	url := conf.ToUrl()
	assetObj.connections[url] = &connectionRecording{
		Url:       url,
		Provider:  provider,
		Connector: conf.Type,
		id:        conf.Id,
	}
	r.assets[connectionID] = assetObj
}

func (r *recording) AddData(connectionID uint32, resource string, id string, field string, data *llx.RawData) {
	asset, ok := r.assets[connectionID]
	if !ok {
		log.Error().Uint32("connectionID", connectionID).Msg("cannot store recording, cannot find connection ID")
		return
	}

	obj, exist := asset.resources[resource+"\x00"+id]
	if !exist {
		obj = &resourceRecording{
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

func (a assetInfo) ToInventory() *inventory.Asset {
	return &inventory.Asset{
		Id:          a.ID,
		PlatformIds: a.PlatformIDs,
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

func PrimitiveArgsToResultArgs(args map[string]*llx.Primitive) map[string]*llx.Result {
	res := make(map[string]*llx.Result, len(args))
	for k, v := range args {
		res[k] = &llx.Result{Data: v}
	}
	return res
}
