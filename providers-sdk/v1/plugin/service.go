// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"errors"
	"fmt"
	"os"
	goruntime "runtime"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"strings"
	sync "sync"
	"time"

	"github.com/rs/zerolog/log"
	llx "go.mondoo.com/mql/v13/llx"
	inventory "go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/memoize"
)

const DISABLE_DELAYED_DISCOVERY_OPTION = "disable-delayed-discovery"

type Service struct {
	runtimes         map[uint32]*Runtime
	lastConnectionID uint32
	runtimesLock     sync.RWMutex

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

	if runtime.Connection != nil {
		if parentId := runtime.Connection.ParentID(); parentId > 0 {
			parentRuntime, err := s.GetRuntime(parentId)
			if err != nil {
				log.Warn().Uint32("parent", parentId).Uint32("child", conf.Id).
					Msg("parent connection not found, proceeding without shared resource cache")
			} else {
				runtime.Resources = parentRuntime.Resources
			}
		}
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

	if runtime.Connection != nil {
		if parentId := runtime.Connection.ParentID(); parentId > 0 {
			parentRuntime, err := s.doGetRuntime(parentId)
			if err != nil {
				log.Warn().Uint32("parent", parentId).Uint32("child", s.lastConnectionID).
					Msg("parent connection not found, proceeding without shared resource cache")
			} else {
				runtime.Resources = parentRuntime.Resources
			}
		}
	}
	s.runtimes[s.lastConnectionID] = runtime
	return runtime, nil
}

// ^^

func (s *Service) GetRuntime(id uint32) (*Runtime, error) {
	s.runtimesLock.RLock()
	defer s.runtimesLock.RUnlock()
	return s.doGetRuntime(id)
}

// doGetRuntime is a helper function to get a runtime by its ID. The caller MUST
// hold s.runtimesLock (read or write).
func (s *Service) doGetRuntime(id uint32) (*Runtime, error) {
	if runtime, ok := s.runtimes[id]; ok {
		return runtime, nil
	}
	return nil, errors.New("connection " + strconv.FormatUint(uint64(id), 10) + " not found")
}

func (s *Service) disconnectRuntime(id uint32) int {
	s.runtimesLock.Lock()
	defer s.runtimesLock.Unlock()
	s.doDisconnect(id)
	if len(s.runtimes) == 0 {
		s.Flush()
	}
	return len(s.runtimes)
}

func (s *Service) Disconnect(req *DisconnectReq) (*DisconnectRes, error) {
	remaining := s.disconnectRuntime(req.Connection)

	debug.FreeOSMemory()

	if os.Getenv("DEBUG_PROVIDER_MEMORY") != "" {
		var m goruntime.MemStats
		goruntime.ReadMemStats(&m)
		log.Info().
			Int("remaining_runtimes", remaining).
			Uint64("heap_alloc_mb", m.Alloc/1024/1024).
			Uint64("heap_sys_mb", m.HeapSys/1024/1024).
			Uint64("heap_inuse_mb", m.HeapInuse/1024/1024).
			Uint64("total_alloc_mb", m.TotalAlloc/1024/1024).
			Msg("provider memory stats after disconnect")

		if remaining == 0 {
			path := fmt.Sprintf("/tmp/provider_heap_%d_conn%d.pprof", os.Getpid(), req.Connection)
			f, err := os.Create(path)
			if err != nil {
				log.Error().Err(err).Str("path", path).Msg("failed to create heap profile file")
			} else {
				if err := pprof.WriteHeapProfile(f); err != nil {
					log.Error().Err(err).Msg("failed to write heap profile")
				} else {
					log.Info().Str("path", path).Msg("wrote heap profile after disconnect")
				}
				f.Close()
			}
		}
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

	resourceKey := req.Resource + "\x00" + req.ResourceId
	resource, ok := runtime.Resources.Get(resourceKey)
	if !ok {
		// The resource doesn't exist in this runtime. This can happen when:
		// 1. The resource was loaded from a recording but the field is not in the
		//    recording. We attempt to create the resource from the recording.
		// 2. The resource was created via a different connection's runtime (e.g.,
		//    cross-provider shared resources in K8s). We search sibling runtimes.
		// 3. The caller is wrong (possibly a mixup with IDs).
		if runtime.HasRecording {
			args, err := runtime.ResourceFromRecording(req.Resource, req.ResourceId)
			if err != nil {
				return nil, errors.New("attempted to load resource '" + req.Resource + "' (id: " + req.ResourceId + ") from recording failed: " + err.Error())
			}

			resource, err = runtime.CreateResource(runtime, req.Resource, args)
			if err != nil {
				return nil, errors.New("attempted to create resource '" + req.Resource + "' (id: " + req.ResourceId + ") from recording failed: " + err.Error())
			}
		} else if resource, ok = s.findResourceInRuntimes(req.Connection, resourceKey); ok {
			// Safe: shared resources that hit this path are fully resolved at creation
			// time and don't lazy-load fields through MqlRuntime.
			runtime.Resources.Set(resourceKey, resource)
		} else {
			return nil, errors.New("resource '" + req.Resource + "' (id: " + req.ResourceId + ") doesn't exist")
		}
	}

	return runtime.GetData(resource, req.Field, args), nil
}

// findResourceInRuntimes searches all runtimes for a resource by key. This
// handles the case where a resource was created via one connection but is being
// looked up from a different connection in the same provider (e.g., cross-provider
// shared resources created via CreateSharedResource in K8s scans).
//
// The search is intentionally global (not scoped to parent/child lineage) because
// the affected resources (container.image, certificates, cpe, tls, etc.) are
// stateless and deterministic — same __id guarantees identical data regardless of
// which connection created them.
func (s *Service) findResourceInRuntimes(connID uint32, resourceKey string) (Resource, bool) {
	s.runtimesLock.RLock()
	defer s.runtimesLock.RUnlock()
	for id, rt := range s.runtimes {
		if id == connID {
			continue
		}
		if res, ok := rt.Resources.Get(resourceKey); ok {
			return res, true
		}
	}
	return nil, false
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

		resource, ok := runtime.Resources.Get(info.Name + "\x00" + info.Id)
		if !ok {
			resource, err = runtime.CreateResource(runtime, info.Name, args)
			if err != nil {
				errs = append(errs, "failed to add cached "+info.Name+" (id: "+info.Id+"), creation failed: "+err.Error())
				continue
			}

			runtime.Resources.Set(info.Name+"\x00"+info.Id, resource)
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
