package providers

import (
	"net/http"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers/proto"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/types"
	"go.mondoo.com/ranger-rpc"
)

type Runtime struct {
	coordinator    *coordinator
	Provider       *RunningProvider
	Connection     *proto.Connection
	schema         *resources.Schema
	UpstreamConfig *UpstreamConfig
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
	return &Runtime{
		coordinator: c,
	}
}

func (r *Runtime) Close() {
	r.coordinator.Close(r.Provider)
	r.schema = nil
}

// UseProvider sets the main provider for this runtime.
func (r *Runtime) UseProvider(name string) error {
	var running *RunningProvider
	for _, p := range r.coordinator.Running {
		if p.Name == name {
			running = p
			break
		}
	}

	if running == nil {
		var err error
		running, err = r.coordinator.Start(name)
		if err != nil {
			return err
		}
	}

	r.Provider = running
	r.schema = running.Schema

	return nil
}

// Connect to an asset using the main provider
func (r *Runtime) Connect(req *proto.ConnectReq) error {
	if r.Provider == nil {
		return errors.New("cannot connect, please select a provider first")
	}

	var err error
	r.Connection, err = r.Provider.Plugin.Connect(req)
	return err
}

func (r *Runtime) CreateResource(name string, args map[string]*llx.Primitive) (llx.Resource, error) {
	res, err := r.Provider.Plugin.GetData(&proto.DataReq{
		Connection: r.Connection.Id,
		Resource:   name,
		Args:       args,
	}, nil)
	if err != nil {
		return nil, err
	}

	typ := types.Type(res.Data.Type)
	return &llx.MockResource{Name: typ.ResourceName(), ID: string(res.Data.Value)}, nil
}

func (r *Runtime) CreateResourceWithID(name string, id string, args map[string]*llx.Primitive) (llx.Resource, error) {
	panic("NOT YET")
}

func (r *Runtime) Resource(name string) (*resources.ResourceInfo, bool) {
	x, ok := r.schema.Resources[name]
	return x, ok
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
	info, ok := r.schema.Resources[name]
	if !ok {
		return errors.New("cannot get resource info on " + name)
	}
	if _, ok := info.Fields[field]; !ok {
		return errors.New("cannot get field '" + field + "' for resource '" + name + "'")
	}

	data, err := r.Provider.Plugin.GetData(&proto.DataReq{
		Connection: r.Connection.Id,
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
	callback(raw.Value, err)
	return nil
}

func (r *Runtime) Schema() *resources.Schema {
	return r.schema
}
