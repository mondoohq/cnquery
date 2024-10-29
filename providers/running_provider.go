// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/rs/zerolog/log"
	pp "go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/resources"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// connectionGraphNode is a node in the connection graph. It represents a connection.
type connectionGraphNode struct {
	// explicitlyConnected is true if the connection was explicitly connected
	// it is set to false when explicitly disconnected.
	// When reconnecting, disconnected connections are not set to explicitly connected,
	// even if we require the connection to connect another connection.
	explicitlyConnected bool
	// data is the connect request data for the connection
	data connectReq
}

// connectionGraph is a directed graph of connections.
// Each node represents a connection. It can have one edge to its parent connection.
//
// When a connection is first connected, addNode is called to add the connection to the graph
// and keep track of the connect request data. This is also when setEdge is called to set the
// edge to the parent connection.
//
// When a connection is disconnected, markDisconnected is called to mark the connection as disconnected.
// When a connection is marked as disconnected, it indicates that the connection is not explicitly required.
// It is still possible that the connection needs to be reconnected if another connection has it set as its
// parent.
// This is also when garbageCollect is called to remove connections from the graph is they are not explicitly
// connected and are not required by any other connection.
type connectionGraph struct {
	// nodes is a map of connection id to connectionGraphNode. We store data to
	// reestablish the connection when reconnecting. We also store if the connection
	// has been disconnected.
	nodes map[uint32]connectionGraphNode
	// edges is a map of connection id to parent connection id
	edges map[uint32]uint32
}

func newConnectionGraph() *connectionGraph {
	return &connectionGraph{
		nodes: map[uint32]connectionGraphNode{},
		edges: map[uint32]uint32{},
	}
}

// addNode adds a node to the graph with the given data.
func (c *connectionGraph) addNode(node uint32, data connectReq) {
	c.nodes[node] = connectionGraphNode{
		explicitlyConnected: true,
		data:                data,
	}
}

// getNode returns the connect request data for the given node.
func (c *connectionGraph) getNode(node uint32) (connectReq, bool) {
	n, ok := c.nodes[node]
	if !ok {
		return connectReq{}, false
	}
	return n.data, ok
}

// setEdge sets the edge from the from node to the to node.
// from is the child node and to is the parent node.
func (c *connectionGraph) setEdge(from, to uint32) {
	c.edges[from] = to
}

// markDisconnected marks the connection as disconnected. It may still be needed by other connections.
func (c *connectionGraph) markDisconnected(id uint32) {
	if node, ok := c.nodes[id]; ok {
		node.explicitlyConnected = false
		c.nodes[id] = node
	}
}

// topoSort returns a topological sorted list of the nodes in the graph. Connecting in this order
// will ensure that all connections are connected in the correct order.
func (c *connectionGraph) topoSort() []uint32 {
	var sorted []uint32
	var visit func(node uint32, visited map[uint32]bool, sorted *[]uint32)
	visit = func(node uint32, visited map[uint32]bool, sorted *[]uint32) {
		if visited[node] {
			return
		}
		visited[node] = true
		if connected, ok := c.edges[node]; ok {
			if connected != 0 {
				visit(connected, visited, sorted)
			}
		}
		*sorted = append(*sorted, node)
	}
	visited := map[uint32]bool{}
	for nodeId, node := range c.nodes {
		if !node.explicitlyConnected {
			continue
		}
		visit(nodeId, visited, &sorted)
	}
	return sorted
}

// garbageCollect removes nodes from the graph that are not explicitly connected and are not required by any other connection.
func (c *connectionGraph) garbageCollect() []uint32 {
	sorted := c.topoSort()

	keep := map[uint32]struct{}{}
	for _, node := range sorted {
		keep[node] = struct{}{}
	}

	collected := []uint32{}
	for node := range c.nodes {
		if _, ok := keep[node]; !ok {
			collected = append(collected, node)
			delete(c.nodes, node)
			delete(c.edges, node)
		}
	}

	return collected
}

type (
	ReconnectFunc func() (pp.ProviderPlugin, *plugin.Client, error)
	connectReq    struct {
		req *pp.ConnectReq
		cb  pp.ProviderCallback
	}
)

const maxRestartCount = 3

type RestartableProvider struct {
	plugin          pp.ProviderPlugin
	client          *plugin.Client
	connectionGraph *connectionGraph
	reconnectFunc   ReconnectFunc
	restartCount    int
	lock            sync.Mutex
}

func (r *RestartableProvider) Client() *plugin.Client {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.client
}

// Connect implements plugin.ProviderPlugin.
func (r *RestartableProvider) Connect(req *pp.ConnectReq, cb pp.ProviderCallback) (*pp.ConnectRes, error) {
	if len(req.Asset.GetConnections()) > 0 {
		reqClone := proto.Clone(req).(*pp.ConnectReq)
		r.lock.Lock()
		connectionId := req.Asset.Connections[0].Id
		if _, ok := r.connectionGraph.getNode(connectionId); !ok {
			r.connectionGraph.addNode(connectionId, connectReq{
				req: reqClone,
				cb:  cb,
			})
			r.connectionGraph.setEdge(connectionId, req.Asset.Connections[0].ParentConnectionId)
		}

		r.lock.Unlock()
	}

	resp, err := r.plugin.Connect(req, cb)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (r *RestartableProvider) Reconnect() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.restartCount >= maxRestartCount {
		return errors.New("reached maximum provider restart count")
	}
	r.restartCount++

	p, c, err := r.reconnectFunc()
	if err != nil {
		return fmt.Errorf("failed to reconnect: %w", err)
	}
	r.plugin = p
	r.client = c

	connectRequestOrder := r.connectionGraph.topoSort()

	for _, connect := range connectRequestOrder {
		cr, ok := r.connectionGraph.getNode(connect)
		if !ok {
			continue
		}

		if _, err := r.plugin.Connect(cr.req, cr.cb); err != nil {
			return fmt.Errorf("failed to reconnect connection %d: %w", connect, err)
		}
	}

	return nil
}

// Disconnect implements plugin.ProviderPlugin.
func (r *RestartableProvider) Disconnect(req *pp.DisconnectReq) (*pp.DisconnectRes, error) {
	r.lock.Lock()
	r.connectionGraph.markDisconnected(req.Connection)
	collected := r.connectionGraph.garbageCollect()
	r.lock.Unlock()

	resp, err := r.plugin.Disconnect(req)

	for _, c := range collected {
		if c == req.Connection {
			continue
		}
		_, err := r.plugin.Disconnect(&pp.DisconnectReq{
			Connection: c,
		})
		if err != nil {
			log.Warn().Err(err).Uint32("connection", c).Msg("failed to disconnect garbage collected connection")
		}
	}

	return resp, err
}

// GetData implements plugin.ProviderPlugin.
func (r *RestartableProvider) GetData(req *pp.DataReq) (*pp.DataRes, error) {
	return r.plugin.GetData(req)
}

// Heartbeat implements plugin.ProviderPlugin.
func (r *RestartableProvider) Heartbeat(req *pp.HeartbeatReq) (*pp.HeartbeatRes, error) {
	return r.plugin.Heartbeat(req)
}

// MockConnect implements plugin.ProviderPlugin.
func (r *RestartableProvider) MockConnect(req *pp.ConnectReq, callback pp.ProviderCallback) (*pp.ConnectRes, error) {
	return r.plugin.MockConnect(req, callback)
}

// ParseCLI implements plugin.ProviderPlugin.
func (r *RestartableProvider) ParseCLI(req *pp.ParseCLIReq) (*pp.ParseCLIRes, error) {
	return r.plugin.ParseCLI(req)
}

// Shutdown implements plugin.ProviderPlugin.
func (r *RestartableProvider) Shutdown(req *pp.ShutdownReq) (*pp.ShutdownRes, error) {
	return r.plugin.Shutdown(req)
}

// StoreData implements plugin.ProviderPlugin.
func (r *RestartableProvider) StoreData(req *pp.StoreReq) (*pp.StoreRes, error) {
	return r.plugin.StoreData(req)
}

var _ pp.ProviderPlugin = &RestartableProvider{}

type RunningProvider struct {
	Name   string
	ID     string
	Plugin pp.ProviderPlugin
	Schema resources.ResourcesSchema

	// isClosed is true for any provider that is not running anymore,
	// either via shutdown or via crash
	isClosed bool
	// isShutdown is only used once during provider shutdown
	isShutdown bool
	// provider errors which are evaluated and printed during shutdown of the provider
	err          error
	lock         sync.Mutex
	shutdownLock sync.Mutex
	interval     time.Duration
	gracePeriod  time.Duration
	hbCancelFunc context.CancelFunc
}

func SupervisedRunningProivder(name string, id string, plugin pp.ProviderPlugin, client *plugin.Client, schema resources.ResourcesSchema, reconnectFunc ReconnectFunc) (*RunningProvider, error) {
	hbCtx, hbCancelFunc := context.WithCancel(context.Background())

	rp := &RunningProvider{
		Name:     name,
		ID:       id,
		Schema:   schema,
		isClosed: false,
		Plugin: &RestartableProvider{
			plugin:          plugin,
			client:          client,
			connectionGraph: newConnectionGraph(),
			reconnectFunc:   reconnectFunc,
		},
		hbCancelFunc: hbCancelFunc,
		interval:     2 * time.Second,
		gracePeriod:  3 * time.Second,
	}

	if err := rp.heartbeat(hbCtx, hbCancelFunc); err != nil {
		return nil, err
	}

	return rp, nil
}

// initialize the heartbeat with the provider
func (p *RunningProvider) heartbeat(ctx context.Context, cancelFunc context.CancelFunc) error {
	if err := p.doOneHeartbeat(p.interval + p.gracePeriod); err != nil {
		log.Error().Err(err).Str("plugin", p.Name).Msg("error in plugin heartbeat")
		if err := p.Shutdown(); err != nil {
			log.Error().Err(err).Str("plugin", p.Name).Msg("error in plugin shutdown")
		}
		return err
	}

	go func() {
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()
		for !p.isCloseOrShutdown() {
			if err := p.doOneHeartbeat(p.interval + p.gracePeriod); err != nil {
				log.Error().Err(err).Str("plugin", p.Name).Msg("error in plugin heartbeat")
				if err := p.Shutdown(); err != nil {
					log.Error().Err(err).Str("plugin", p.Name).Msg("error in plugin shutdown")
				}
				break
			}

			select {
			case <-ctx.Done():
				cancelFunc()
				return
			case <-ticker.C:

			}
		}
	}()

	return nil
}

func (p *RunningProvider) doOneHeartbeat(t time.Duration) error {
	_, err := p.Plugin.Heartbeat(&pp.HeartbeatReq{
		Interval: uint64(t),
	})
	if err != nil {
		log.Err(err).Str("plugin", p.Name).Msg("error in plugin heartbeat")
		if status, ok := status.FromError(err); ok {
			if status.Code() == 12 {
				return errors.New("please update the provider plugin for " + p.Name)
			}
		}
		return errors.New("cannot establish heartbeat with the provider plugin for " + p.Name)
	}
	return nil
}

func (p *RunningProvider) isCloseOrShutdown() bool {
	p.shutdownLock.Lock()
	defer p.shutdownLock.Unlock()
	return p.isClosed || p.isShutdown
}

func (p *RunningProvider) Reconnect() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.shutdownLock.Lock()
	defer p.shutdownLock.Unlock()
	if !(p.isClosed || p.isShutdown) {
		return nil
	}

	// we can only restart if it is a restartable provider
	if rp, ok := p.Plugin.(*RestartableProvider); ok {
		log.Warn().Str("plugin", p.Name).Msg("reconnecting provider")
		if err := rp.Reconnect(); err != nil {
			log.Error().Err(err).Str("plugin", p.Name).Msg("error in plugin reconnect")
			return err
		}
		p.isClosed = false
		p.isShutdown = false
		hbCtx, hbCancelFunc := context.WithCancel(context.Background())
		if p.hbCancelFunc != nil {
			p.hbCancelFunc()
		}
		p.hbCancelFunc = hbCancelFunc
		return p.heartbeat(hbCtx, hbCancelFunc)
	}

	return errors.New("provider is not restartable")
}

func (p *RunningProvider) Shutdown() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.isShutdown {
		return nil
	}

	// This is an error that happened earlier, so we print it directly.
	// The error this function returns is about failing to shutdown.
	if p.err != nil {
		log.Error().Msg(p.err.Error())
	}

	var err error
	if !p.isClosed {
		_, err = p.Plugin.Shutdown(&pp.ShutdownReq{})
		if err != nil {
			log.Debug().Err(err).Str("plugin", p.Name).Msg("error in plugin shutdown")
		}

		// If the plugin was not in active use, we may not have a client at this
		// point. Since all of this is run within a sync-lock, we can check the
		// client and if it exists use it to send the kill signal.
		if rp, ok := p.Plugin.(*RestartableProvider); ok {
			c := rp.Client()
			if c != nil {
				c.Kill()
			}
		}
		p.shutdownLock.Lock()
		p.isClosed = true
		p.isShutdown = true
		p.shutdownLock.Unlock()
	} else {
		p.shutdownLock.Lock()
		p.isShutdown = true
		p.shutdownLock.Unlock()
	}

	return err
}

func (p *RunningProvider) KillClient() {
	if rp, ok := p.Plugin.(*RestartableProvider); ok {
		c := rp.Client()
		if c != nil {
			c.Kill()
		}
	}
}
