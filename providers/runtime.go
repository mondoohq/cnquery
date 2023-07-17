package providers

import (
	"net/http"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory/manager"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/types"
	"go.mondoo.com/ranger-rpc"
	protobuf "google.golang.org/protobuf/proto"
)

// Runtimes are associated with one asset and carry all providers
// and open connections for that asset.
type Runtime struct {
	Provider       *ConnectedProvider
	UpstreamConfig *UpstreamConfig
	Recording      Recording

	features []byte
	asset    *inventory.Asset
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

// mondoo platform config so that resource scan talk upstream
// TODO: this configuration struct does not belong into the MQL package
// nevertheless the MQL runtime needs to have something that allows users
// to store additional credentials so that resource can use those for
// their resources.
type UpstreamConfig struct {
	AssetMrn    string
	SpaceMrn    string
	ApiEndpoint string
	Plugins     []ranger.ClientPlugin
	Incognito   bool
	HttpClient  *http.Client
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

// Connect to an asset using the main provider
func (r *Runtime) Connect(req *plugin.ConnectReq) error {
	if r.Provider == nil {
		return errors.New("cannot connect, please select a provider first")
	}

	if req.Asset == nil || req.Asset.Spec == nil || len(req.Asset.Spec.Assets) == 0 {
		return errors.New("cannot connect, no asset info provided")
	}

	asset := req.Asset.Spec.Assets[0]
	if len(asset.Connections) == 0 {
		return errors.New("cannot connect to asset, no connection info provided")
	}

	r.asset = asset
	r.features = req.Features

	// if we already have a connection, which typically happens during CLI parsing
	// when there is discovery of assets going on as well...
	conn := asset.Connections[0]
	if conn.Id != 0 {
		r.Provider.Connection = &plugin.ConnectRes{Id: conn.Id, Name: conn.ToUrl()}
		r.Recording.EnsureAsset(asset, r.Provider.Instance.Name, conn)
		return nil
	}

	manager, err := manager.NewManager(manager.WithInventory(req.Asset, r))
	if err != nil {
		return errors.New("failed to resolve inventory for connection: " + err.Error())
	}

	inventoryAsset := manager.GetAssets()[0]

	creds := manager.GetCredsResolver()
	if creds != nil {
		inventoryAsset = protobuf.Clone(inventoryAsset).(*inventory.Asset)
		req = &plugin.ConnectReq{
			Features: req.Features,
			Asset: &inventory.Inventory{
				Spec: &inventory.InventorySpec{
					Assets: []*inventory.Asset{inventoryAsset},
				},
			},
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

	r.Provider.Connection, err = r.Provider.Instance.Plugin.Connect(req)
	if err != nil {
		return err
	}

	r.Recording.EnsureAsset(asset, r.Provider.Instance.Name, conn)
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
	}, nil)
	if err != nil {
		return nil, err
	}

	if cached, ok := r.Recording.GetResource(provider.Connection.Id, name, string(res.Data.Value)); ok {
		fields, err := RawDataArgsToPrimitiveArgs(cached)
		if err != nil {
			log.Error().Str("resource", name).Str("id", string(res.Data.Value)).Msg("failed to load resource from recording")
		} else {
			provider.Instance.Plugin.StoreData(&plugin.StoreReq{
				Connection: provider.Connection.Id,
				Resources: []*plugin.ResourceData{{
					Name:   name,
					Id:     string(res.Data.Value),
					Fields: fields,
				}},
			})
		}
	} else {
		r.Recording.AddData(provider.Connection.Id, name, string(res.Data.Value), "", nil)
	}

	typ := types.Type(res.Data.Type)
	return &llx.MockResource{Name: typ.ResourceName(), ID: string(res.Data.Value)}, nil
}

func (r *Runtime) CreateResourceWithID(name string, id string, args map[string]*llx.Primitive) (llx.Resource, error) {
	panic("NOT YET")
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
	name := resource.MqlName()
	id := resource.MqlID()

	provider, info, err := r.lookupResourceProvider(name)
	if err != nil {
		return err
	}

	if _, ok := info.Fields[field]; !ok {
		return errors.New("cannot get field '" + field + "' for resource '" + name + "'")
	}

	if cached, ok := r.Recording.GetData(provider.Connection.Id, name, id, field); ok {
		callback(cached.Value, cached.Error)
		return nil
	}

	data, err := provider.Instance.Plugin.GetData(&plugin.DataReq{
		Connection: provider.Connection.Id,
		Resource:   name,
		ResourceId: id,
		Field:      field,
	}, nil)
	if err != nil {
		return err
	}

	if data.Error != "" {
		err = errors.New(data.Error)
	}
	raw := data.Data.RawData()

	r.Recording.AddData(provider.Connection.Id, name, id, field, raw)

	callback(raw.Value, err)
	return nil
}

func (r *Runtime) SetMockConnection(id string) (uint32, error) {
	provider, ok := r.providers[id]
	if !ok {
		return 0, errors.New("cannot find provider to mock connection: " + id)
	}

	res, err := provider.Instance.Plugin.Connect(&plugin.ConnectReq{
		Asset: &inventory.Inventory{
			Spec: &inventory.InventorySpec{
				Assets: []*inventory.Asset{{
					Id: "mock-asset",
					Connections: []*inventory.Config{{
						Type: "mock",
					}},
				}},
			},
		},
	})
	if err != nil {
		return 0, errors.New("failed to set mock connection: " + err.Error())
	}

	provider.Connection = res
	return res.Id, nil
}

func (r *Runtime) SetRecording(recording *recording, providerID string) error {
	r.Recording = recording
	provider, ok := r.providers[providerID]
	if !ok {
		return errors.New("cannot set recording, provider '" + providerID + "' not found")
	}

	recording.assets[provider.Connection.Id] = &recording.Assets[0]
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
		Asset:    &inventory.Inventory{Spec: &inventory.InventorySpec{Assets: []*inventory.Asset{r.asset}}},
	})
	if err != nil {
		return nil, nil, err
	}

	res.Connection = conn

	return res, info, nil
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

	providers, err := List()
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
	for k, v := range schema.Resources {
		x.Schema.Resources[k] = v
	}
}

func (x *extensibleSchema) AllResources() map[string]*resources.ResourceInfo {
	if !x.allLoaded {
		x.loadAllSchemas()
	}

	return x.Resources
}
