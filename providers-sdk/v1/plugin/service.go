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

	llx "go.mondoo.com/cnquery/v11/llx"
	inventory "go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
)

const DISABLE_DELAYED_DISCOVERY_OPTION = "disable-delayed-discovery"

type Service struct {
	runtimes         map[uint32]*Runtime
	lastConnectionID uint32
	runtimesLock     sync.Mutex

	lastHeartbeat int64
	heartbeatLock sync.Mutex
}

func NewService() *Service {
	return &Service{
		runtimes: make(map[uint32]*Runtime),
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

	s.runtimesLock.Lock()
	defer s.runtimesLock.Unlock()

	// If a runtime with this ID already exists, then return that
	if runtime, ok := s.runtimes[conf.Id]; ok {
		return runtime, nil
	}

	runtime, err := createRuntime(conf.Id)
	if err != nil {
		return nil, err
	}

	if runtime.Connection != nil {
		if parentId := runtime.Connection.ParentID(); parentId > 0 {
			parentRuntime, err := s.doGetRuntime(parentId)
			if err != nil {
				return nil, errors.New("parent connection " + strconv.FormatUint(uint64(parentId), 10) + " not found")
			}
			runtime.Resources = parentRuntime.Resources

		}
	}
	s.runtimes[conf.Id] = runtime
	return runtime, nil
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
				return nil, errors.New("parent connection " + strconv.FormatUint(uint64(parentId), 10) + " not found")
			}
			runtime.Resources = parentRuntime.Resources

		}
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

	return runtime.GetData(resource, req.Field, args), nil
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
			os.Exit(1)
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
	return &ShutdownRes{}, nil
}
