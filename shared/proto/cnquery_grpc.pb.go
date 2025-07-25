// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v6.31.1
// source: cnquery.proto

package proto

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
	CNQuery_RunQuery_FullMethodName = "/proto.CNQuery/RunQuery"
)

// CNQueryClient is the client API for CNQuery service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type CNQueryClient interface {
	RunQuery(ctx context.Context, in *RunQueryConfig, opts ...grpc.CallOption) (*Empty, error)
}

type cNQueryClient struct {
	cc grpc.ClientConnInterface
}

func NewCNQueryClient(cc grpc.ClientConnInterface) CNQueryClient {
	return &cNQueryClient{cc}
}

func (c *cNQueryClient) RunQuery(ctx context.Context, in *RunQueryConfig, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, CNQuery_RunQuery_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CNQueryServer is the server API for CNQuery service.
// All implementations must embed UnimplementedCNQueryServer
// for forward compatibility
type CNQueryServer interface {
	RunQuery(context.Context, *RunQueryConfig) (*Empty, error)
	mustEmbedUnimplementedCNQueryServer()
}

// UnimplementedCNQueryServer must be embedded to have forward compatible implementations.
type UnimplementedCNQueryServer struct {
}

func (UnimplementedCNQueryServer) RunQuery(context.Context, *RunQueryConfig) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RunQuery not implemented")
}
func (UnimplementedCNQueryServer) mustEmbedUnimplementedCNQueryServer() {}

// UnsafeCNQueryServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to CNQueryServer will
// result in compilation errors.
type UnsafeCNQueryServer interface {
	mustEmbedUnimplementedCNQueryServer()
}

func RegisterCNQueryServer(s grpc.ServiceRegistrar, srv CNQueryServer) {
	s.RegisterService(&CNQuery_ServiceDesc, srv)
}

func _CNQuery_RunQuery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RunQueryConfig)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CNQueryServer).RunQuery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: CNQuery_RunQuery_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CNQueryServer).RunQuery(ctx, req.(*RunQueryConfig))
	}
	return interceptor(ctx, in, info, handler)
}

// CNQuery_ServiceDesc is the grpc.ServiceDesc for CNQuery service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var CNQuery_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.CNQuery",
	HandlerType: (*CNQueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RunQuery",
			Handler:    _CNQuery_RunQuery_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "cnquery.proto",
}

const (
	OutputHelper_Write_FullMethodName = "/proto.OutputHelper/Write"
)

// OutputHelperClient is the client API for OutputHelper service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type OutputHelperClient interface {
	Write(ctx context.Context, in *String, opts ...grpc.CallOption) (*Empty, error)
}

type outputHelperClient struct {
	cc grpc.ClientConnInterface
}

func NewOutputHelperClient(cc grpc.ClientConnInterface) OutputHelperClient {
	return &outputHelperClient{cc}
}

func (c *outputHelperClient) Write(ctx context.Context, in *String, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, OutputHelper_Write_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// OutputHelperServer is the server API for OutputHelper service.
// All implementations must embed UnimplementedOutputHelperServer
// for forward compatibility
type OutputHelperServer interface {
	Write(context.Context, *String) (*Empty, error)
	mustEmbedUnimplementedOutputHelperServer()
}

// UnimplementedOutputHelperServer must be embedded to have forward compatible implementations.
type UnimplementedOutputHelperServer struct {
}

func (UnimplementedOutputHelperServer) Write(context.Context, *String) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Write not implemented")
}
func (UnimplementedOutputHelperServer) mustEmbedUnimplementedOutputHelperServer() {}

// UnsafeOutputHelperServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to OutputHelperServer will
// result in compilation errors.
type UnsafeOutputHelperServer interface {
	mustEmbedUnimplementedOutputHelperServer()
}

func RegisterOutputHelperServer(s grpc.ServiceRegistrar, srv OutputHelperServer) {
	s.RegisterService(&OutputHelper_ServiceDesc, srv)
}

func _OutputHelper_Write_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(String)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OutputHelperServer).Write(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: OutputHelper_Write_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OutputHelperServer).Write(ctx, req.(*String))
	}
	return interceptor(ctx, in, info, handler)
}

// OutputHelper_ServiceDesc is the grpc.ServiceDesc for OutputHelper service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var OutputHelper_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.OutputHelper",
	HandlerType: (*OutputHelperServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Write",
			Handler:    _OutputHelper_Write_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "cnquery.proto",
}
