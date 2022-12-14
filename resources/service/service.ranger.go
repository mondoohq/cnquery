// Code generated by protoc-gen-rangerrpc version DO NOT EDIT.
// source: service.proto

package service

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"go.mondoo.com/cnquery/resources"
	ranger "go.mondoo.com/ranger-rpc"
	"go.mondoo.com/ranger-rpc/metadata"
	jsonpb "google.golang.org/protobuf/encoding/protojson"
	pb "google.golang.org/protobuf/proto"
)

// service interface definition

type MQL interface {
	ListResources(context.Context, *Empty) (*ResourceList, error)
	GetSchema(context.Context, *Empty) (*resources.Schema, error)
	ListFields(context.Context, *FieldsQuery) (*Fields, error)
	CreateResource(context.Context, *ResourceArguments) (*resources.ResourceID, error)
	GetField(context.Context, *FieldArguments) (*FieldReturn, error)
}

// client implementation

type MQLClient struct {
	ranger.Client
	httpclient ranger.HTTPClient
	prefix     string
}

func NewMQLClient(addr string, client ranger.HTTPClient, plugins ...ranger.ClientPlugin) (*MQLClient, error) {
	base, err := url.Parse(ranger.SanitizeUrl(addr))
	if err != nil {
		return nil, err
	}

	u, err := url.Parse("./MQL")
	if err != nil {
		return nil, err
	}

	serviceClient := &MQLClient{
		httpclient: client,
		prefix:     base.ResolveReference(u).String(),
	}
	serviceClient.AddPlugins(plugins...)
	return serviceClient, nil
}
func (c *MQLClient) ListResources(ctx context.Context, in *Empty) (*ResourceList, error) {
	out := new(ResourceList)
	err := c.DoClientRequest(ctx, c.httpclient, strings.Join([]string{c.prefix, "/ListResources"}, ""), in, out)
	return out, err
}
func (c *MQLClient) GetSchema(ctx context.Context, in *Empty) (*resources.Schema, error) {
	out := new(resources.Schema)
	err := c.DoClientRequest(ctx, c.httpclient, strings.Join([]string{c.prefix, "/GetSchema"}, ""), in, out)
	return out, err
}
func (c *MQLClient) ListFields(ctx context.Context, in *FieldsQuery) (*Fields, error) {
	out := new(Fields)
	err := c.DoClientRequest(ctx, c.httpclient, strings.Join([]string{c.prefix, "/ListFields"}, ""), in, out)
	return out, err
}
func (c *MQLClient) CreateResource(ctx context.Context, in *ResourceArguments) (*resources.ResourceID, error) {
	out := new(resources.ResourceID)
	err := c.DoClientRequest(ctx, c.httpclient, strings.Join([]string{c.prefix, "/CreateResource"}, ""), in, out)
	return out, err
}
func (c *MQLClient) GetField(ctx context.Context, in *FieldArguments) (*FieldReturn, error) {
	out := new(FieldReturn)
	err := c.DoClientRequest(ctx, c.httpclient, strings.Join([]string{c.prefix, "/GetField"}, ""), in, out)
	return out, err
}

// server implementation

type MQLServerOption func(s *MQLServer)

func WithUnknownFieldsForMQLServer() MQLServerOption {
	return func(s *MQLServer) {
		s.allowUnknownFields = true
	}
}

func NewMQLServer(handler MQL, opts ...MQLServerOption) http.Handler {
	srv := &MQLServer{
		handler: handler,
	}

	for i := range opts {
		opts[i](srv)
	}

	service := ranger.Service{
		Name: "MQL",
		Methods: map[string]ranger.Method{
			"ListResources":  srv.ListResources,
			"GetSchema":      srv.GetSchema,
			"ListFields":     srv.ListFields,
			"CreateResource": srv.CreateResource,
			"GetField":       srv.GetField,
		},
	}
	return ranger.NewRPCServer(&service)
}

type MQLServer struct {
	handler            MQL
	allowUnknownFields bool
}

func (p *MQLServer) ListResources(ctx context.Context, reqBytes *[]byte) (pb.Message, error) {
	var req Empty
	var err error

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not access header")
	}

	switch md.First("Content-Type") {
	case "application/protobuf", "application/octet-stream", "application/grpc+proto":
		err = pb.Unmarshal(*reqBytes, &req)
	default:
		// handle case of empty object
		if len(*reqBytes) > 0 {
			err = jsonpb.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(*reqBytes, &req)
		}
	}

	if err != nil {
		return nil, err
	}
	return p.handler.ListResources(ctx, &req)
}
func (p *MQLServer) GetSchema(ctx context.Context, reqBytes *[]byte) (pb.Message, error) {
	var req Empty
	var err error

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not access header")
	}

	switch md.First("Content-Type") {
	case "application/protobuf", "application/octet-stream", "application/grpc+proto":
		err = pb.Unmarshal(*reqBytes, &req)
	default:
		// handle case of empty object
		if len(*reqBytes) > 0 {
			err = jsonpb.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(*reqBytes, &req)
		}
	}

	if err != nil {
		return nil, err
	}
	return p.handler.GetSchema(ctx, &req)
}
func (p *MQLServer) ListFields(ctx context.Context, reqBytes *[]byte) (pb.Message, error) {
	var req FieldsQuery
	var err error

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not access header")
	}

	switch md.First("Content-Type") {
	case "application/protobuf", "application/octet-stream", "application/grpc+proto":
		err = pb.Unmarshal(*reqBytes, &req)
	default:
		// handle case of empty object
		if len(*reqBytes) > 0 {
			err = jsonpb.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(*reqBytes, &req)
		}
	}

	if err != nil {
		return nil, err
	}
	return p.handler.ListFields(ctx, &req)
}
func (p *MQLServer) CreateResource(ctx context.Context, reqBytes *[]byte) (pb.Message, error) {
	var req ResourceArguments
	var err error

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not access header")
	}

	switch md.First("Content-Type") {
	case "application/protobuf", "application/octet-stream", "application/grpc+proto":
		err = pb.Unmarshal(*reqBytes, &req)
	default:
		// handle case of empty object
		if len(*reqBytes) > 0 {
			err = jsonpb.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(*reqBytes, &req)
		}
	}

	if err != nil {
		return nil, err
	}
	return p.handler.CreateResource(ctx, &req)
}
func (p *MQLServer) GetField(ctx context.Context, reqBytes *[]byte) (pb.Message, error) {
	var req FieldArguments
	var err error

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not access header")
	}

	switch md.First("Content-Type") {
	case "application/protobuf", "application/octet-stream", "application/grpc+proto":
		err = pb.Unmarshal(*reqBytes, &req)
	default:
		// handle case of empty object
		if len(*reqBytes) > 0 {
			err = jsonpb.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(*reqBytes, &req)
		}
	}

	if err != nil {
		return nil, err
	}
	return p.handler.GetField(ctx, &req)
}
