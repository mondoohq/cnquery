// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/consul/connection"
	"go.mondoo.com/mql/v13/types"
)

// mqlConsulInternal caches the agent's self report. Nearly every field on this
// resource is read out of that one payload, so a query touching several of them
// costs a single request rather than one per field.
type mqlConsulInternal struct {
	selfOnce sync.Once
	self     *agentSelf
	selfErr  error
}

func (r *mqlConsul) id() (string, error) {
	conn, err := consulConn(r.MqlRuntime)
	if err != nil {
		return "", err
	}
	return connection.NewConsulAgentIdentifier(conn.Host()), nil
}

// consulConn pulls the connection off the runtime.
func consulConn(runtime *plugin.Runtime) (*connection.ConsulConnection, error) {
	conn, ok := runtime.Connection.(*connection.ConsulConnection)
	if !ok {
		return nil, errors.New("no Consul connection on the runtime")
	}
	return conn, nil
}

// consulClient returns the configured API client.
func consulClient(runtime *plugin.Runtime) (*consulapi.Client, error) {
	conn, err := consulConn(runtime)
	if err != nil {
		return nil, err
	}
	return conn.Client(), nil
}

// fetchSelf reads the agent's self report once per resource instance.
func (r *mqlConsul) fetchSelf() (*agentSelf, error) {
	r.selfOnce.Do(func() {
		client, err := consulClient(r.MqlRuntime)
		if err != nil {
			r.selfErr = err
			return
		}
		raw, err := client.Agent().Self()
		if err != nil {
			r.selfErr = err
			return
		}
		r.self, r.selfErr = decodeAgentSelf(raw)
	})
	return r.self, r.selfErr
}

func (r *mqlConsul) datacenter() (string, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return "", err
	}
	return self.Config.Datacenter, nil
}

func (r *mqlConsul) primaryDatacenter() (string, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return "", err
	}
	return self.Config.PrimaryDatacenter, nil
}

func (r *mqlConsul) nodeName() (string, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return "", err
	}
	return self.Config.NodeName, nil
}

func (r *mqlConsul) nodeId() (string, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return "", err
	}
	return self.Config.NodeID, nil
}

func (r *mqlConsul) version() (string, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return "", err
	}
	return self.Config.Version, nil
}

func (r *mqlConsul) revision() (string, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return "", err
	}
	return self.Config.Revision, nil
}

func (r *mqlConsul) buildDate() (*time.Time, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return nil, err
	}
	built := self.Config.buildTime()
	if built == nil {
		// A build reporting no timestamp is reported as null rather than as
		// the zero time, which would render as a date in year one.
		r.BuildDate.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return built, nil
}

func (r *mqlConsul) server() (bool, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return false, err
	}
	return self.Config.Server, nil
}

func (r *mqlConsul) gossipEncrypted() (bool, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return false, err
	}
	encrypted, present, err := self.serfEncrypted("serf_lan")
	if err != nil {
		return false, err
	}
	if !present {
		// Every agent runs a LAN gossip pool, so a missing pool means the
		// payload was not an agent self report. Reporting false here would say
		// "not encrypted" about something that was never read.
		return false, errors.New("the Consul agent reported no LAN gossip pool")
	}
	return encrypted, nil
}

func (r *mqlConsul) wanGossipEncrypted() (bool, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return false, err
	}
	encrypted, present, err := self.serfEncrypted("serf_wan")
	if err != nil {
		return false, err
	}
	if !present {
		// A client agent runs no WAN pool. That is an absent pool rather than
		// an unencrypted one, so the field is null instead of false.
		r.WanGossipEncrypted.State = plugin.StateIsSet | plugin.StateIsNull
		return false, nil
	}
	return encrypted, nil
}

func (r *mqlConsul) verifyIncoming() (bool, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return false, err
	}
	return self.DebugConfig.internalRPCTLS().VerifyIncoming, nil
}

func (r *mqlConsul) verifyOutgoing() (bool, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return false, err
	}
	return self.DebugConfig.internalRPCTLS().VerifyOutgoing, nil
}

func (r *mqlConsul) verifyServerHostname() (bool, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return false, err
	}
	return self.DebugConfig.internalRPCTLS().VerifyServerHostname, nil
}

func (r *mqlConsul) autoEncryptTls() (bool, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return false, err
	}
	return self.DebugConfig.AutoEncryptTLS, nil
}

func (r *mqlConsul) autoEncryptAllowTls() (bool, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return false, err
	}
	return self.DebugConfig.AutoEncryptAllowTLS, nil
}

func (r *mqlConsul) tlsProfiles() ([]any, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return nil, err
	}

	scopes := self.DebugConfig.tlsScopes()
	res := make([]any, 0, len(scopes))
	for _, scoped := range scopes {
		profile, err := CreateResource(r.MqlRuntime, "consul.tlsProfile", map[string]*llx.RawData{
			"__id":                 llx.StringData(r.__id + "/tls/" + scoped.Scope),
			"scope":                llx.StringData(scoped.Scope),
			"verifyIncoming":       llx.BoolData(scoped.Config.VerifyIncoming),
			"verifyOutgoing":       llx.BoolData(scoped.Config.VerifyOutgoing),
			"verifyServerHostname": llx.BoolData(scoped.Config.VerifyServerHostname),
			"tlsMinVersion":        llx.StringData(scoped.Config.TLSMinVersion),
			"cipherSuites":         llx.ArrayData(convert.SliceAnyToInterface(scoped.Config.CipherSuites), types.String),
			"caFile":               llx.StringData(scoped.Config.CAFile),
			"caPath":               llx.StringData(scoped.Config.CAPath),
			"certFile":             llx.StringData(scoped.Config.CertFile),
			"useAutoCert":          llx.BoolData(scoped.Config.UseAutoCert),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, profile)
	}
	return res, nil
}

func (r *mqlConsul) acl() (*mqlConsulAclSystem, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return nil, err
	}

	debug := self.DebugConfig
	enabled := debug.aclEnabled()
	defaultPolicy := debug.aclDefaultPolicy()

	res, err := CreateResource(r.MqlRuntime, "consul.aclSystem", map[string]*llx.RawData{
		"__id":                llx.StringData(r.__id + "/acl"),
		"enabled":             llx.BoolData(enabled),
		"defaultPolicy":       llx.StringData(defaultPolicy),
		"downPolicy":          llx.StringData(debug.aclDownPolicy()),
		"defaultDeny":         llx.BoolData(aclDefaultDeny(enabled, defaultPolicy)),
		"tokenReplication":    llx.BoolData(debug.ACLTokenReplication),
		"enableKeyListPolicy": llx.BoolData(debug.ACLEnableKeyListPolicy),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlConsulAclSystem), nil
}

func (r *mqlConsul) mesh() (*mqlConsulServiceMesh, error) {
	self, err := r.fetchSelf()
	if err != nil {
		return nil, err
	}

	debug := self.DebugConfig
	intentionPolicy := defaultIntentionPolicy(debug.aclEnabled(), debug.aclDefaultPolicy())

	res, err := CreateResource(r.MqlRuntime, "consul.serviceMesh", map[string]*llx.RawData{
		"__id":                   llx.StringData(r.__id + "/mesh"),
		"enabled":                llx.BoolData(debug.ConnectEnabled),
		"defaultIntentionPolicy": llx.StringData(intentionPolicy),
		"defaultDeny":            llx.BoolData(intentionPolicy == intentionPolicyDeny),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlConsulServiceMesh), nil
}
