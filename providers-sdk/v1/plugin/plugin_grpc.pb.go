// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v6.31.1
// source: plugin.proto

package plugin

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	ProviderPlugin_Heartbeat_FullMethodName   = "/cnquery.providers.v1.ProviderPlugin/Heartbeat"
	ProviderPlugin_ParseCLI_FullMethodName    = "/cnquery.providers.v1.ProviderPlugin/ParseCLI"
	ProviderPlugin_Connect_FullMethodName     = "/cnquery.providers.v1.ProviderPlugin/Connect"
	ProviderPlugin_Disconnect_FullMethodName  = "/cnquery.providers.v1.ProviderPlugin/Disconnect"
	ProviderPlugin_MockConnect_FullMethodName = "/cnquery.providers.v1.ProviderPlugin/MockConnect"
	ProviderPlugin_Shutdown_FullMethodName    = "/cnquery.providers.v1.ProviderPlugin/Shutdown"
	ProviderPlugin_GetData_FullMethodName     = "/cnquery.providers.v1.ProviderPlugin/GetData"
	ProviderPlugin_StoreData_FullMethodName   = "/cnquery.providers.v1.ProviderPlugin/StoreData"
)

// ProviderPluginClient is the client API for ProviderPlugin service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ProviderPluginClient interface {
	Heartbeat(ctx context.Context, in *HeartbeatReq, opts ...grpc.CallOption) (*HeartbeatRes, error)
	ParseCLI(ctx context.Context, in *ParseCLIReq, opts ...grpc.CallOption) (*ParseCLIRes, error)
	Connect(ctx context.Context, in *ConnectReq, opts ...grpc.CallOption) (*ConnectRes, error)
	Disconnect(ctx context.Context, in *DisconnectReq, opts ...grpc.CallOption) (*DisconnectRes, error)
	MockConnect(ctx context.Context, in *ConnectReq, opts ...grpc.CallOption) (*ConnectRes, error)
	Shutdown(ctx context.Context, in *ShutdownReq, opts ...grpc.CallOption) (*ShutdownRes, error)
	GetData(ctx context.Context, in *DataReq, opts ...grpc.CallOption) (*DataRes, error)
	StoreData(ctx context.Context, in *StoreReq, opts ...grpc.CallOption) (*StoreRes, error)
}

type providerPluginClient struct {
	cc grpc.ClientConnInterface
}

func NewProviderPluginClient(cc grpc.ClientConnInterface) ProviderPluginClient {
	return &providerPluginClient{cc}
}

func (c *providerPluginClient) Heartbeat(ctx context.Context, in *HeartbeatReq, opts ...grpc.CallOption) (*HeartbeatRes, error) {
	out := new(HeartbeatRes)
	err := c.cc.Invoke(ctx, ProviderPlugin_Heartbeat_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *providerPluginClient) ParseCLI(ctx context.Context, in *ParseCLIReq, opts ...grpc.CallOption) (*ParseCLIRes, error) {
	out := new(ParseCLIRes)
	err := c.cc.Invoke(ctx, ProviderPlugin_ParseCLI_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *providerPluginClient) Connect(ctx context.Context, in *ConnectReq, opts ...grpc.CallOption) (*ConnectRes, error) {
	out := new(ConnectRes)
	err := c.cc.Invoke(ctx, ProviderPlugin_Connect_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *providerPluginClient) Disconnect(ctx context.Context, in *DisconnectReq, opts ...grpc.CallOption) (*DisconnectRes, error) {
	out := new(DisconnectRes)
	err := c.cc.Invoke(ctx, ProviderPlugin_Disconnect_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *providerPluginClient) MockConnect(ctx context.Context, in *ConnectReq, opts ...grpc.CallOption) (*ConnectRes, error) {
	out := new(ConnectRes)
	err := c.cc.Invoke(ctx, ProviderPlugin_MockConnect_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *providerPluginClient) Shutdown(ctx context.Context, in *ShutdownReq, opts ...grpc.CallOption) (*ShutdownRes, error) {
	out := new(ShutdownRes)
	err := c.cc.Invoke(ctx, ProviderPlugin_Shutdown_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *providerPluginClient) GetData(ctx context.Context, in *DataReq, opts ...grpc.CallOption) (*DataRes, error) {
	out := new(DataRes)
	err := c.cc.Invoke(ctx, ProviderPlugin_GetData_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *providerPluginClient) StoreData(ctx context.Context, in *StoreReq, opts ...grpc.CallOption) (*StoreRes, error) {
	out := new(StoreRes)
	err := c.cc.Invoke(ctx, ProviderPlugin_StoreData_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ProviderPluginServer is the server API for ProviderPlugin service.
// All implementations must embed UnimplementedProviderPluginServer
// for forward compatibility
type ProviderPluginServer interface {
	Heartbeat(context.Context, *HeartbeatReq) (*HeartbeatRes, error)
	ParseCLI(context.Context, *ParseCLIReq) (*ParseCLIRes, error)
	Connect(context.Context, *ConnectReq) (*ConnectRes, error)
	Disconnect(context.Context, *DisconnectReq) (*DisconnectRes, error)
	MockConnect(context.Context, *ConnectReq) (*ConnectRes, error)
	Shutdown(context.Context, *ShutdownReq) (*ShutdownRes, error)
	GetData(context.Context, *DataReq) (*DataRes, error)
	StoreData(context.Context, *StoreReq) (*StoreRes, error)
	mustEmbedUnimplementedProviderPluginServer()
}

// UnimplementedProviderPluginServer must be embedded to have forward compatible implementations.
type UnimplementedProviderPluginServer struct {
}

func (UnimplementedProviderPluginServer) Heartbeat(context.Context, *HeartbeatReq) (*HeartbeatRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Heartbeat not implemented")
}
func (UnimplementedProviderPluginServer) ParseCLI(context.Context, *ParseCLIReq) (*ParseCLIRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ParseCLI not implemented")
}
func (UnimplementedProviderPluginServer) Connect(context.Context, *ConnectReq) (*ConnectRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Connect not implemented")
}
func (UnimplementedProviderPluginServer) Disconnect(context.Context, *DisconnectReq) (*DisconnectRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Disconnect not implemented")
}
func (UnimplementedProviderPluginServer) MockConnect(context.Context, *ConnectReq) (*ConnectRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MockConnect not implemented")
}
func (UnimplementedProviderPluginServer) Shutdown(context.Context, *ShutdownReq) (*ShutdownRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Shutdown not implemented")
}
func (UnimplementedProviderPluginServer) GetData(context.Context, *DataReq) (*DataRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetData not implemented")
}
func (UnimplementedProviderPluginServer) StoreData(context.Context, *StoreReq) (*StoreRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StoreData not implemented")
}
func (UnimplementedProviderPluginServer) mustEmbedUnimplementedProviderPluginServer() {}

// UnsafeProviderPluginServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ProviderPluginServer will
// result in compilation errors.
type UnsafeProviderPluginServer interface {
	mustEmbedUnimplementedProviderPluginServer()
}

func RegisterProviderPluginServer(s grpc.ServiceRegistrar, srv ProviderPluginServer) {
	s.RegisterService(&ProviderPlugin_ServiceDesc, srv)
}

func _ProviderPlugin_Heartbeat_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HeartbeatReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProviderPluginServer).Heartbeat(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ProviderPlugin_Heartbeat_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProviderPluginServer).Heartbeat(ctx, req.(*HeartbeatReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProviderPlugin_ParseCLI_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ParseCLIReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProviderPluginServer).ParseCLI(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ProviderPlugin_ParseCLI_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProviderPluginServer).ParseCLI(ctx, req.(*ParseCLIReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProviderPlugin_Connect_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ConnectReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProviderPluginServer).Connect(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ProviderPlugin_Connect_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProviderPluginServer).Connect(ctx, req.(*ConnectReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProviderPlugin_Disconnect_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DisconnectReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProviderPluginServer).Disconnect(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ProviderPlugin_Disconnect_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProviderPluginServer).Disconnect(ctx, req.(*DisconnectReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProviderPlugin_MockConnect_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ConnectReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProviderPluginServer).MockConnect(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ProviderPlugin_MockConnect_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProviderPluginServer).MockConnect(ctx, req.(*ConnectReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProviderPlugin_Shutdown_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ShutdownReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProviderPluginServer).Shutdown(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ProviderPlugin_Shutdown_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProviderPluginServer).Shutdown(ctx, req.(*ShutdownReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProviderPlugin_GetData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DataReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProviderPluginServer).GetData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ProviderPlugin_GetData_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProviderPluginServer).GetData(ctx, req.(*DataReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProviderPlugin_StoreData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StoreReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProviderPluginServer).StoreData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ProviderPlugin_StoreData_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProviderPluginServer).StoreData(ctx, req.(*StoreReq))
	}
	return interceptor(ctx, in, info, handler)
}

// ProviderPlugin_ServiceDesc is the grpc.ServiceDesc for ProviderPlugin service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ProviderPlugin_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cnquery.providers.v1.ProviderPlugin",
	HandlerType: (*ProviderPluginServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Heartbeat",
			Handler:    _ProviderPlugin_Heartbeat_Handler,
		},
		{
			MethodName: "ParseCLI",
			Handler:    _ProviderPlugin_ParseCLI_Handler,
		},
		{
			MethodName: "Connect",
			Handler:    _ProviderPlugin_Connect_Handler,
		},
		{
			MethodName: "Disconnect",
			Handler:    _ProviderPlugin_Disconnect_Handler,
		},
		{
			MethodName: "MockConnect",
			Handler:    _ProviderPlugin_MockConnect_Handler,
		},
		{
			MethodName: "Shutdown",
			Handler:    _ProviderPlugin_Shutdown_Handler,
		},
		{
			MethodName: "GetData",
			Handler:    _ProviderPlugin_GetData_Handler,
		},
		{
			MethodName: "StoreData",
			Handler:    _ProviderPlugin_StoreData_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "plugin.proto",
}

const (
	ProviderCallback_Collect_FullMethodName      = "/cnquery.providers.v1.ProviderCallback/Collect"
	ProviderCallback_GetRecording_FullMethodName = "/cnquery.providers.v1.ProviderCallback/GetRecording"
	ProviderCallback_GetData_FullMethodName      = "/cnquery.providers.v1.ProviderCallback/GetData"
)

// ProviderCallbackClient is the client API for ProviderCallback service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ProviderCallbackClient interface {
	Collect(ctx context.Context, in *DataRes, opts ...grpc.CallOption) (*CollectRes, error)
	GetRecording(ctx context.Context, in *DataReq, opts ...grpc.CallOption) (*ResourceData, error)
	GetData(ctx context.Context, in *DataReq, opts ...grpc.CallOption) (*DataRes, error)
}

type providerCallbackClient struct {
	cc grpc.ClientConnInterface
}

func NewProviderCallbackClient(cc grpc.ClientConnInterface) ProviderCallbackClient {
	return &providerCallbackClient{cc}
}

func (c *providerCallbackClient) Collect(ctx context.Context, in *DataRes, opts ...grpc.CallOption) (*CollectRes, error) {
	out := new(CollectRes)
	err := c.cc.Invoke(ctx, ProviderCallback_Collect_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *providerCallbackClient) GetRecording(ctx context.Context, in *DataReq, opts ...grpc.CallOption) (*ResourceData, error) {
	out := new(ResourceData)
	err := c.cc.Invoke(ctx, ProviderCallback_GetRecording_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *providerCallbackClient) GetData(ctx context.Context, in *DataReq, opts ...grpc.CallOption) (*DataRes, error) {
	out := new(DataRes)
	err := c.cc.Invoke(ctx, ProviderCallback_GetData_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ProviderCallbackServer is the server API for ProviderCallback service.
// All implementations must embed UnimplementedProviderCallbackServer
// for forward compatibility
type ProviderCallbackServer interface {
	Collect(context.Context, *DataRes) (*CollectRes, error)
	GetRecording(context.Context, *DataReq) (*ResourceData, error)
	GetData(context.Context, *DataReq) (*DataRes, error)
	mustEmbedUnimplementedProviderCallbackServer()
}

// UnimplementedProviderCallbackServer must be embedded to have forward compatible implementations.
type UnimplementedProviderCallbackServer struct {
}

func (UnimplementedProviderCallbackServer) Collect(context.Context, *DataRes) (*CollectRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Collect not implemented")
}
func (UnimplementedProviderCallbackServer) GetRecording(context.Context, *DataReq) (*ResourceData, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRecording not implemented")
}
func (UnimplementedProviderCallbackServer) GetData(context.Context, *DataReq) (*DataRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetData not implemented")
}
func (UnimplementedProviderCallbackServer) mustEmbedUnimplementedProviderCallbackServer() {}

// UnsafeProviderCallbackServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ProviderCallbackServer will
// result in compilation errors.
type UnsafeProviderCallbackServer interface {
	mustEmbedUnimplementedProviderCallbackServer()
}

func RegisterProviderCallbackServer(s grpc.ServiceRegistrar, srv ProviderCallbackServer) {
	s.RegisterService(&ProviderCallback_ServiceDesc, srv)
}

func _ProviderCallback_Collect_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DataRes)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProviderCallbackServer).Collect(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ProviderCallback_Collect_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProviderCallbackServer).Collect(ctx, req.(*DataRes))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProviderCallback_GetRecording_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DataReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProviderCallbackServer).GetRecording(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ProviderCallback_GetRecording_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProviderCallbackServer).GetRecording(ctx, req.(*DataReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _ProviderCallback_GetData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DataReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProviderCallbackServer).GetData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ProviderCallback_GetData_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProviderCallbackServer).GetData(ctx, req.(*DataReq))
	}
	return interceptor(ctx, in, info, handler)
}

// ProviderCallback_ServiceDesc is the grpc.ServiceDesc for ProviderCallback service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ProviderCallback_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cnquery.providers.v1.ProviderCallback",
	HandlerType: (*ProviderCallbackServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Collect",
			Handler:    _ProviderCallback_Collect_Handler,
		},
		{
			MethodName: "GetRecording",
			Handler:    _ProviderCallback_GetRecording_Handler,
		},
		{
			MethodName: "GetData",
			Handler:    _ProviderCallback_GetData_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "plugin.proto",
}
