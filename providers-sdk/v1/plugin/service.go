// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"errors"
	"os"
	"strconv"
	"strings"
	sync "sync"
	"time"

	llx "go.mondoo.com/mql/v13/llx"
	inventory "go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/memoize"
)

const DISABLE_DELAYED_DISCOVERY_OPTION = "disable-delayed-discovery"

type Service struct {
	runtimes         map[uint32]*Runtime
	lastConnectionID uint32
	runtimesLock     sync.Mutex

	lastHeartbeat int64
	heartbeatLock sync.Mutex

	memoize.Memoizer
}

var (
	cacheExpirationTime = 3 * time.Hour
	cacheCleanupTime    = 6 * time.Hour
)

func NewService() *Service {
	return &Service{
		runtimes: make(map[uint32]*Runtime),
		Memoizer: memoize.New(cacheExpirationTime, cacheCleanupTime),
	}
}

var heartbeatRes HeartbeatRes

// FIXME: once we move to v12, remove the conf parameter and remove the connId from the createRuntime function.
// The connection ID will always be set before the connection call is done, so we don't need to do anything about it here.
// The parameters are needed now, only to make sure that old clients can work with new providers.
func (s *Service) AddRuntime(conf *inventory.Config, createRuntime func(connId uint32) (*Runtime, error)) (*Runtime, error) {
	// FIXME: DEPRECATED, remove in v12.0 vv
	// This approach is used only when old clients use new providers. We will throw it away in v12
	if conf.Id == 0 {
		if conf.Options == nil {
			conf.Options = make(map[string]string)
		}

		// Disable delayed discovery for old clients since they don't know how to handle it
		conf.Options[DISABLE_DELAYED_DISCOVERY_OPTION] = "true"
		return s.deprecatedAddRuntime(createRuntime)
	}
	// ^^

	// If a runtime with this ID already exists, then return that
	if runtime, err := s.GetRuntime(conf.Id); err == nil {
		return runtime, nil
	}

	runtime, err := createRuntime(conf.Id)
	if err != nil {
		return nil, err
	}

	isChild := false
	if runtime.Connection != nil {
		if parentId := runtime.Connection.ParentID(); parentId > 0 {
			parentRuntime, err := s.GetRuntime(parentId)
			if err != nil {
				return nil, errors.New("parent connection " + strconv.FormatUint(uint64(parentId), 10) + " not found")
			}
			runtime.Resources = parentRuntime.Resources
			isChild = true
		}
	}

	// Upgrade to SQLite-backed cache only for root connections (not children
	// that share the parent's cache), avoiding throwaway temp DB files.
	if !isChild {
		InitSqliteResources(runtime)
	}

	// store the new runtime
	s.addRuntime(conf.Id, runtime)

	return runtime, nil
}

func (s *Service) addRuntime(id uint32, runtime *Runtime) {
	s.runtimesLock.Lock()
	defer s.runtimesLock.Unlock()
	s.runtimes[id] = runtime
}

// FIXME: DEPRECATED, remove in v12.0 vv
func (s *Service) deprecatedAddRuntime(createRuntime func(connId uint32) (*Runtime, error)) (*Runtime, error) {
	s.runtimesLock.Lock()
	defer s.runtimesLock.Unlock()

	s.lastConnectionID++
	runtime, err := createRuntime(s.lastConnectionID)
	if err != nil {
		// If the runtime creation fails, revert the lastConnectionID
		s.lastConnectionID--
		return nil, err
	}

	isChild := false
	if runtime.Connection != nil {
		if parentId := runtime.Connection.ParentID(); parentId > 0 {
			parentRuntime, err := s.doGetRuntime(parentId)
			if err != nil {
				return nil, errors.New("parent connection " + strconv.FormatUint(uint64(parentId), 10) + " not found")
			}
			runtime.Resources = parentRuntime.Resources
			isChild = true
		}
	}
	if !isChild {
		InitSqliteResources(runtime)
	}
	s.runtimes[s.lastConnectionID] = runtime
	return runtime, nil
}

// ^^

func (s *Service) GetRuntime(id uint32) (*Runtime, error) {
	s.runtimesLock.Lock()
	defer s.runtimesLock.Unlock()
	return s.doGetRuntime(id)
}

// doGetRuntime is a helper function to get a runtime by its ID. It MUST be called
// with a lock on s.runtimesLock.
func (s *Service) doGetRuntime(id uint32) (*Runtime, error) {
	if runtime, ok := s.runtimes[id]; ok {
		return runtime, nil
	}
	return nil, errors.New("connection " + strconv.FormatUint(uint64(id), 10) + " not found")
}

func (s *Service) Disconnect(req *DisconnectReq) (*DisconnectRes, error) {
	s.runtimesLock.Lock()
	defer s.runtimesLock.Unlock()
	s.doDisconnect(req.Connection)
	if len(s.runtimes) == 0 {
		// flush our memoizer when there are no more connected runtimes
		s.Flush()
	}
	return &DisconnectRes{}, nil
}

// doDisconnect is a helper function to disconnect a runtime by its ID. It MUST be called
// with a lock on s.runtimesLock.
func (s *Service) doDisconnect(id uint32) {
	if runtime, ok := s.runtimes[id]; ok {
		// If the runtime implements the Closer interface, we need to call the
		// Close function
		if closer, ok := runtime.Connection.(Closer); ok {
			closer.Close()
		}
		delete(s.runtimes, id)

		// Close the resource cache if it implements io.Closer (e.g. SqliteResources),
		// but only if no other runtime shares the same Resources instance.
		if closer, ok := runtime.Resources.(interface{ Close() error }); ok {
			shared := false
			for _, other := range s.runtimes {
				if other.Resources == runtime.Resources {
					shared = true
					break
				}
			}
			if !shared {
				closer.Close()
			}
		}
	}
}

func (s *Service) GetData(req *DataReq) (*DataRes, error) {
	runtime, err := s.GetRuntime(req.Connection)
	if err != nil {
		return nil, err
	}

	args := PrimitiveArgsToRawDataArgs(req.Args, runtime)

	if req.ResourceId == "" && req.Field == "" {
		res, err := runtime.NewResource(runtime, req.Resource, args)
		if err != nil {
			return nil, err
		}

		rd := llx.ResourceData(res, res.MqlName()).Result()
		return &DataRes{
			Data: rd.Data,
		}, nil
	}

	resource, ok := runtime.Resources.Get(req.Resource + "\x00" + req.ResourceId)
	if !ok {
		// Note: Since resources are internally always created, there are only very
		// few cases where we arrive here:
		// 1. The caller is wrong. Possibly a mixup with IDs
		// 2. The resource was loaded from a recording, but the field is not
		// in the recording. Thus the resource was never created inside the
		// plugin. We will attempt to create the resource and see if the field
		// can be computed.
		if !runtime.HasRecording {
			return nil, errors.New("resource '" + req.Resource + "' (id: " + req.ResourceId + ") doesn't exist")
		}

		args, err := runtime.ResourceFromRecording(req.Resource, req.ResourceId)
		if err != nil {
			return nil, errors.New("attempted to load resource '" + req.Resource + "' (id: " + req.ResourceId + ") from recording failed: " + err.Error())
		}

		resource, err = runtime.CreateResource(runtime, req.Resource, args)
		if err != nil {
			return nil, errors.New("attempted to create resource '" + req.Resource + "' (id: " + req.ResourceId + ") from recording failed: " + err.Error())
		}
	}

	cacheKey := req.Resource + "\x00" + req.ResourceId

	// Check field cache before expensive computation.
	if fc, ok := runtime.Resources.(ResourcesWithFieldCache); ok {
		if cached := fc.GetField(cacheKey, req.Field); cached != nil {
			return cached, nil
		}
	}

	res := runtime.GetData(resource, req.Field, args)

	// Re-insert the resource into the cache after field computation.
	// Field methods (e.g. packages.list) may create many child resources via
	// CreateResource, which can evict the parent resource from the LRU.
	// Without this, the next GetData call would reconstruct a fresh instance
	// from SQLite, losing all computed TValue fields and re-triggering
	// expensive operations like system_profiler or API calls.
	runtime.Resources.Set(cacheKey, resource)

	// Cache the field result for future reconstruction.
	// Skip empty DataRes (nil Data + empty Error) — that means NotReady.
	if res.Data != nil || res.Error != "" {
		if fc, ok := runtime.Resources.(ResourcesWithFieldCache); ok {
			fc.SetField(cacheKey, req.Field, res)
		}
	}

	return res, nil
}

func (s *Service) StoreData(req *StoreReq) (*StoreRes, error) {
	runtime, err := s.GetRuntime(req.Connection)
	if err != nil {
		return nil, err
	}

	var errs []string
	for i := range req.Resources {
		info := req.Resources[i]

		args, err := ProtoArgsToRawDataArgs(info.Fields)
		if err != nil {
			errs = append(errs, "failed to add cached "+info.Name+" (id: "+info.Id+"), failed to parse arguments")
			continue
		}

		cacheKey := info.Name + "\x00" + info.Id
		resource, ok := runtime.Resources.Get(cacheKey)
		if !ok {
			resource, err = runtime.CreateResource(runtime, info.Name, args)
			if err != nil {
				errs = append(errs, "failed to add cached "+info.Name+" (id: "+info.Id+"), creation failed: "+err.Error())
				continue
			}

			runtime.Resources.Set(cacheKey, resource)
			if rwa, ok := runtime.Resources.(ResourcesWithArgs); ok {
				rwa.SetWithArgs(cacheKey, resource, args)
			}
		}

		for k, v := range args {
			if err := runtime.SetData(resource, k, v); err != nil {
				errs = append(errs, "failed to add cached "+info.Name+" (id: "+info.Id+"), field error: "+err.Error())
			}
		}
	}

	if len(errs) != 0 {
		return nil, errors.New(strings.Join(errs, ", "))
	}
	return &StoreRes{}, nil
}

func (s *Service) Heartbeat(req *HeartbeatReq) (*HeartbeatRes, error) {
	if req.Interval == 0 {
		return nil, errors.New("heartbeat failed, requested interval is 0")
	}

	now := time.Now().UnixNano()
	s.heartbeatLock.Lock()
	s.lastHeartbeat = now
	s.heartbeatLock.Unlock()

	go func() {
		time.Sleep(time.Duration(req.Interval))

		s.heartbeatLock.Lock()
		isDead := s.lastHeartbeat == now
		s.heartbeatLock.Unlock()

		if isDead {
			// use 4 since we actually do not want to reach the point, see tetraphobia
			os.Exit(4)
		}
	}()

	return &heartbeatRes, nil
}

func (s *Service) Shutdown(req *ShutdownReq) (*ShutdownRes, error) {
	s.runtimesLock.Lock()
	defer s.runtimesLock.Unlock()

	for id := range s.runtimes {
		s.doDisconnect(id)
	}

	s.Flush() // flush our Memoizer
	return &ShutdownRes{}, nil
}
