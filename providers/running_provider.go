// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-plugin"
	"github.com/rs/zerolog/log"
	pp "go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/resources"
	"google.golang.org/grpc/status"
)


type RunningProvider struct {
	Name   string
	ID     string
	Plugin pp.ProviderPlugin
	Client *plugin.Client
	Schema *resources.Schema

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
}

// initialize the heartbeat with the provider
func (p *RunningProvider) heartbeat() error {
	if err := p.doOneHeartbeat(p.interval + p.gracePeriod); err != nil {
		p.Shutdown()
		return err
	}

	go func() {
		for !p.isCloseOrShutdown() {
			if err := p.doOneHeartbeat(p.interval + p.gracePeriod); err != nil {
				p.Shutdown()
				break
			}

			time.Sleep(p.interval)
		}
	}()

	return nil
}

func (p *RunningProvider) doOneHeartbeat(t time.Duration) error {
	_, err := p.Plugin.Heartbeat(&pp.HeartbeatReq{
		Interval: uint64(t),
	})
	if err != nil {
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
		if p.Client != nil {
			p.Client.Kill()
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
