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

	llx "go.mondoo.com/cnquery/v10/llx"
)

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

func (s *Service) AddRuntime(runtime *Runtime) uint32 {
	s.runtimesLock.Lock()
	defer s.runtimesLock.Unlock()
	s.lastConnectionID++
	runtime.Connection.SetID(s.lastConnectionID)
	s.runtimes[s.lastConnectionID] = runtime
	return s.lastConnectionID
}

func (s *Service) GetRuntime(id uint32) (*Runtime, error) {
	s.runtimesLock.Lock()
	defer s.runtimesLock.Unlock()
	if runtime, ok := s.runtimes[id]; ok {
		return runtime, nil
	}
	return nil, errors.New("connection " + strconv.FormatUint(uint64(id), 10) + " not found")
}

func (s *Service) Disconnect(req *DisconnectReq) (*DisconnectRes, error) {
	s.runtimesLock.Lock()
	defer s.runtimesLock.Unlock()
	if runtime, ok := s.runtimes[req.Connection]; ok {
		// If the runtime implements the Closer interface, we need to call the
		// Close function
		if closer, ok := runtime.Connection.(Closer); ok {
			closer.Close()
		}
		delete(s.runtimes, req.Connection)
	}
	return &DisconnectRes{}, nil
}

func (s *Service) GetData(req *DataReq) (*DataRes, error) {
	runtime, ok := s.runtimes[req.Connection]
	if !ok {
		return nil, errors.New("connection " + strconv.FormatUint(uint64(req.Connection), 10) + " not found")
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
	runtime, ok := s.runtimes[req.Connection]
	if !ok {
		return nil, errors.New("connection " + strconv.FormatUint(uint64(req.Connection), 10) + " not found")
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
