package providers

import (
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory/manager"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/types"
	protobuf "google.golang.org/protobuf/proto"
)

// Runtimes are associated with one asset and carry all providers
// and open connections for that asset.
type Runtime struct {
	Provider       *ConnectedProvider
	UpstreamConfig *upstream.UpstreamConfig
	Recording      Recording

	features []byte
	// coordinator is used to grab providers
	coordinator *coordinator
	// providers for with open connections
	providers map[string]*ConnectedProvider
	// schema aggregates all resources executable on this asset
	schema   extensibleSchema
	isClosed bool
}

type ConnectedProvider struct {
	Instance   *RunningProvider
	Connection *plugin.ConnectRes
}

func (c *coordinator) NewRuntime() *Runtime {
	res := &Runtime{
		coordinator: c,
		providers:   map[string]*ConnectedProvider{},
		schema: extensibleSchema{
			loaded: map[string]struct{}{},
			Schema: resources.Schema{
				Resources: map[string]*resources.ResourceInfo{},
			},
		},
		Recording: nullRecording{},
	}
	res.schema.runtime = res

	// TODO: do this dynamically in the future
	res.schema.loadAllSchemas()
	return res
}

func (r *Runtime) Close() {
	if r.isClosed {
		return
	}
	r.isClosed = true

	if err := r.Recording.Save(); err != nil {
		log.Error().Err(err).Msg("failed to save recording")
	}

	r.coordinator.Close(r.Provider.Instance)
	r.schema.Close()
}

func (r *Runtime) DeactivateProviderDiscovery() {
	r.schema.allLoaded = true
}

func (r *Runtime) AssetMRN() string {
	if r.Provider != nil && r.Provider.Connection != nil && r.Provider.Connection.Asset != nil {
		return r.Provider.Connection.Asset.Mrn
	}
	return ""
}

// UseProvider sets the main provider for this runtime.
func (r *Runtime) UseProvider(id string) error {
	res, err := r.addProvider(id)
	if err != nil {
		return err
	}

	r.Provider = res
	return nil
}

func (r *Runtime) AddConnectedProvider(c *ConnectedProvider) {
	r.providers[c.Instance.ID] = c
	r.schema.Add(c.Instance.Name, c.Instance.Schema)
}

func (r *Runtime) addProvider(id string) (*ConnectedProvider, error) {
	var running *RunningProvider
	for _, p := range r.coordinator.Running {
		if p.ID == id {
			running = p
			break
		}
	}

	if running == nil {
		var err error
		running, err = r.coordinator.Start(id)
		if err != nil {
			return nil, err
		}
	}

	res := &ConnectedProvider{Instance: running}
	r.AddConnectedProvider(res)

	return res, nil
}

// DetectProvider will try to detect and start the right provider for this
// runtime. Generally recommended when you receive an asset to be scanned,
// but haven't initialized any provider. It will also try to install providers
// if necessary (and enabled)
func (r *Runtime) DetectProvider(asset *inventory.Asset) error {
	if asset == nil {
		return errors.New("please provide an asset to detect the provider")
	}
	if len(asset.Connections) == 0 {
		return errors.New("asset has no connections, can't detect provider")
	}

	var errs []string
	for i := range asset.Connections {
		conn := asset.Connections[i]
		if conn.Type == "" {
			continue
		}

		provider, err := EnsureProvider(r.coordinator.Providers, conn.Type)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}

		return r.UseProvider(provider.ID)
	}

	return errors.New("cannot find provider for this asset (" + strings.Join(errs, ",") + ")")
}

// Connect to an asset using the main provider
func (r *Runtime) Connect(req *plugin.ConnectReq) error {
	if r.Provider == nil {
		return errors.New("cannot connect, please select a provider first")
	}

	if req.Asset == nil {
		return errors.New("cannot connect, no asset info provided")
	}

	asset := req.Asset
	if len(asset.Connections) == 0 {
		return errors.New("cannot connect to asset, no connection info provided")
	}

	r.features = req.Features
	conn := asset.Connections[0]

	manager, err := manager.NewManager(manager.WithInventory(&inventory.Inventory{
		Spec: &inventory.InventorySpec{
			Assets: []*inventory.Asset{asset},
		},
	}, r))
	if err != nil {
		return errors.New("failed to resolve inventory for connection: " + err.Error())
	}

	inventoryAsset := manager.GetAssets()[0]

	creds := manager.GetCredsResolver()
	if creds != nil {
		inventoryAsset = protobuf.Clone(inventoryAsset).(*inventory.Asset)
		req = &plugin.ConnectReq{
			Features:     req.Features,
			Asset:        inventoryAsset,
			HasRecording: req.HasRecording,
			Upstream:     req.Upstream,
		}

		for j := range inventoryAsset.Connections {
			conn := inventoryAsset.Connections[j]
			for k := range conn.Credentials {
				credential := conn.Credentials[k]
				if credential.SecretId == "" {
					continue
				}

				resolvedCredential, err := creds.GetCredential(credential)
				if err != nil {
					log.Debug().Str("secret-id", credential.SecretId).Err(err).Msg("could not fetch secret for motor connection")
					return err
				}

				conn.Credentials[k] = resolvedCredential
			}
		}
	}

	callbacks := providerCallbacks{
		runtime: r,
	}

	r.Provider.Connection, err = r.Provider.Instance.Plugin.Connect(req, &callbacks)
	if err != nil {
		return err
	}

	r.Recording.EnsureAsset(r.Provider.Connection.Asset, r.Provider.Instance.Name, r.Provider.Connection.Id, conn)
	return nil
}

func (r *Runtime) CreateResource(name string, args map[string]*llx.Primitive) (llx.Resource, error) {
	provider, _, err := r.lookupResourceProvider(name)
	if err != nil {
		return nil, err
	}

	res, err := provider.Instance.Plugin.GetData(&plugin.DataReq{
		Connection: provider.Connection.Id,
		Resource:   name,
		Args:       args,
	})
	if err != nil {
		return nil, err
	}

	if _, ok := r.Recording.GetResource(provider.Connection.Id, name, string(res.Data.Value)); !ok {
		r.Recording.AddData(provider.Connection.Id, name, string(res.Data.Value), "", nil)
	}

	typ := types.Type(res.Data.Type)
	return &llx.MockResource{Name: typ.ResourceName(), ID: string(res.Data.Value)}, nil
}

func (r *Runtime) CreateResourceWithID(name string, id string, args map[string]*llx.Primitive) (llx.Resource, error) {
	provider, _, err := r.lookupResourceProvider(name)
	if err != nil {
		return nil, err
	}

	args["__id"] = llx.StringPrimitive(id)

	_, err = provider.Instance.Plugin.StoreData(&plugin.StoreReq{
		Connection: provider.Connection.Id,
		Resources: []*plugin.ResourceData{{
			Name:   name,
			Id:     id,
			Fields: PrimitiveArgsToResultArgs(args),
		}},
	})
	if err != nil {
		return nil, err
	}

	return &llx.MockResource{Name: name, ID: id}, nil
}

func (r *Runtime) Unregister(watcherUID string) error {
	// TODO: we don't unregister just yet...
	return nil
}

func fieldUID(resource string, id string, field string) string {
	return resource + "\x00" + id + "\x00" + field
}

// WatchAndUpdate a resource field and call the function if it changes with its current value
func (r *Runtime) WatchAndUpdate(resource llx.Resource, field string, watcherUID string, callback func(res interface{}, err error)) error {
	raw, err := r.watchAndUpdate(resource.MqlName(), resource.MqlID(), field, watcherUID)
	if raw != nil {
		callback(raw.Value, raw.Error)
	}
	return err
}

func (r *Runtime) watchAndUpdate(resource string, resourceID string, field string, watcherUID string) (*llx.RawData, error) {
	provider, info, fieldInfo, err := r.lookupFieldProvider(resource, field)
	if err != nil {
		return nil, err
	}
	if fieldInfo == nil {
		return nil, errors.New("cannot get field '" + field + "' for resource '" + resource + "'")
	}

	if info.Provider != fieldInfo.Provider {
		// technically we don't need to look up the resource provider, since
		// it had to have been called beforehand to get here
		_, err := provider.Instance.Plugin.StoreData(&plugin.StoreReq{
			Connection: provider.Connection.Id,
			Resources: []*plugin.ResourceData{{
				Name: resource,
				Id:   resourceID,
			}},
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create reference resource "+resource+" in provider "+provider.Instance.Name)
		}
	}

	if cached, ok := r.Recording.GetData(provider.Connection.Id, resource, resourceID, field); ok {
		return cached, nil
	}

	data, err := provider.Instance.Plugin.GetData(&plugin.DataReq{
		Connection: provider.Connection.Id,
		Resource:   resource,
		ResourceId: resourceID,
		Field:      field,
	})
	if err != nil {
		return nil, err
	}

	var raw *llx.RawData
	if data.Error != "" {
		raw = &llx.RawData{Error: errors.New(data.Error)}
	} else {
		raw = data.Data.RawData()
	}

	r.Recording.AddData(provider.Connection.Id, resource, resourceID, field, raw)
	return raw, nil
}

type providerCallbacks struct {
	recording *assetRecording
	runtime   *Runtime
}

func (p *providerCallbacks) GetRecording(req *plugin.DataReq) (*plugin.ResourceData, error) {
	resource, ok := p.recording.resources[req.Resource+"\x00"+req.ResourceId]
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

func (p *providerCallbacks) GetData(req *plugin.DataReq) (*plugin.DataRes, error) {
	if req.Field == "" {
		res, err := p.runtime.CreateResource(req.Resource, req.Args)
		if err != nil {
			return nil, err
		}

		return &plugin.DataRes{
			Data: &llx.Primitive{
				Type:  string(types.Resource(res.MqlName())),
				Value: []byte(res.MqlID()),
			},
		}, nil
	}

	raw, err := p.runtime.watchAndUpdate(req.Resource, req.ResourceId, req.Field, "")
	if raw == nil {
		return nil, err
	}
	res := raw.Result()
	return &plugin.DataRes{
		Data:  res.Data,
		Error: res.Error,
	}, err
}

func (p *providerCallbacks) Collect(req *plugin.DataRes) error {
	panic("NOT YET IMPLEMENTED")
	return nil
}

func (r *Runtime) SetRecording(recording *recording, providerID string, readOnly bool, mockConnection bool) error {
	if readOnly {
		r.Recording = &readOnlyRecording{recording}
	} else {
		r.Recording = recording
	}

	provider, ok := r.providers[providerID]
	if !ok {
		return errors.New("cannot set recording, provider '" + providerID + "' not found")
	}

	assetRecording := &recording.Assets[0]
	asset := assetRecording.Asset.ToInventory()

	if mockConnection {
		// Dom: we may need to retain the original asset ID, not sure yet...
		asset.Id = "mock-asset"
		asset.Connections = []*inventory.Config{{
			Type: "mock",
		}}

		callbacks := providerCallbacks{
			recording: assetRecording,
			runtime:   r,
		}

		res, err := provider.Instance.Plugin.Connect(&plugin.ConnectReq{
			Asset:        asset,
			HasRecording: true,
		}, &callbacks)
		if err != nil {
			return errors.New("failed to set mock connection for recording: " + err.Error())
		}
		provider.Connection = res
	}

	if provider.Connection == nil {
		// Dom: we may need to cancel the entire setup here, may need to be reconsidered...
		log.Warn().Msg("recording cannot determine asset, no connection was set up!")
	} else {
		recording.assets[provider.Connection.Id] = assetRecording
	}

	return nil
}

func (r *Runtime) lookupResourceProvider(resource string) (*ConnectedProvider, *resources.ResourceInfo, error) {
	info := r.schema.Lookup(resource)
	if info == nil {
		return nil, nil, errors.New("cannot find resource '" + resource + "' in schema")
	}

	if provider := r.providers[info.Provider]; provider != nil {
		return provider, info, nil
	}

	res, err := r.addProvider(info.Provider)
	if err != nil {
		return nil, nil, errors.New("failed to start provider '" + info.Provider + "': " + err.Error())
	}

	conn, err := res.Instance.Plugin.Connect(&plugin.ConnectReq{
		Features: r.features,
		Asset:    r.Provider.Connection.Asset,
	}, nil)
	if err != nil {
		return nil, nil, err
	}

	res.Connection = conn

	return res, info, nil
}

func (r *Runtime) lookupFieldProvider(resource string, field string) (*ConnectedProvider, *resources.ResourceInfo, *resources.Field, error) {
	resourceInfo, fieldInfo := r.schema.LookupField(resource, field)
	if resourceInfo == nil {
		return nil, nil, nil, errors.New("cannot find resource '" + resource + "' in schema")
	}
	if fieldInfo == nil {
		return nil, nil, nil, errors.New("cannot find field '" + field + "' in resource '" + resource + "'")
	}

	if provider := r.providers[fieldInfo.Provider]; provider != nil {
		return provider, resourceInfo, fieldInfo, nil
	}

	res, err := r.addProvider(fieldInfo.Provider)
	if err != nil {
		return nil, nil, nil, errors.New("failed to start provider '" + fieldInfo.Provider + "': " + err.Error())
	}

	conn, err := res.Instance.Plugin.Connect(&plugin.ConnectReq{
		Features: r.features,
		Asset:    r.Provider.Connection.Asset,
	}, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	res.Connection = conn

	return res, resourceInfo, fieldInfo, nil
}

func (r *Runtime) Schema() llx.Schema {
	return &r.schema
}

func (r *Runtime) AddSchema(name string, schema *resources.Schema) {
	r.schema.Add(name, schema)
}

type extensibleSchema struct {
	resources.Schema

	loaded    map[string]struct{}
	runtime   *Runtime
	allLoaded bool
	lockAll   sync.Mutex // only used in getting all schemas
	lockAdd   sync.Mutex // only used when adding a schema
}

func (x *extensibleSchema) loadAllSchemas() {
	x.lockAll.Lock()
	defer x.lockAll.Unlock()

	// If another goroutine started to load this before us, it will be locked until
	// we complete to load everything and then it will be dumped into this
	// position. At this point, if it has been loaded we can return safely, since
	// we don't unlock until we are finished loading.
	if x.allLoaded {
		return
	}
	x.allLoaded = true

	providers, err := ListActive()
	if err != nil {
		log.Error().Err(err).Msg("failed to list all providers, can't load additional schemas")
		return
	}

	for name := range providers {
		schema := x.runtime.coordinator.LoadSchema(name)
		x.Add(name, schema)
	}
}

func (x *extensibleSchema) Close() {
	x.loaded = map[string]struct{}{}
	x.Schema.Resources = nil
}

func (x *extensibleSchema) Lookup(name string) *resources.ResourceInfo {
	if found, ok := x.Resources[name]; ok {
		return found
	}
	if x.allLoaded {
		return nil
	}

	x.loadAllSchemas()
	return x.Resources[name]
}

func (x *extensibleSchema) LookupField(resource string, field string) (*resources.ResourceInfo, *resources.Field) {
	found, ok := x.Resources[resource]
	if !ok {
		if x.allLoaded {
			return nil, nil
		}

		x.loadAllSchemas()

		found, ok = x.Resources[resource]
		if !ok {
			return nil, nil
		}
		return found, found.Fields[field]
	}

	fieldObj, ok := found.Fields[field]
	if ok {
		return found, fieldObj
	}
	if x.allLoaded {
		return found, nil
	}

	x.loadAllSchemas()
	return found, found.Fields[field]
}

func (x *extensibleSchema) Add(name string, schema *resources.Schema) {
	if schema == nil {
		return
	}
	if name == "" {
		log.Error().Msg("tried to add a schema with no name")
		return
	}

	x.lockAdd.Lock()
	defer x.lockAdd.Unlock()

	if _, ok := x.loaded[name]; ok {
		return
	}

	x.loaded[name] = struct{}{}
	x.Schema.Add(schema)
}

func (x *extensibleSchema) AllResources() map[string]*resources.ResourceInfo {
	if !x.allLoaded {
		x.loadAllSchemas()
	}

	return x.Resources
}
