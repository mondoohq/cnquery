// Code generated by protoc-gen-rangerrpc version DO NOT EDIT.
// source: upstream.proto

package upstream

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	ranger "go.mondoo.com/ranger-rpc"
	"go.mondoo.com/ranger-rpc/metadata"
	jsonpb "google.golang.org/protobuf/encoding/protojson"
	pb "google.golang.org/protobuf/proto"
)

// service interface definition

type AgentManager interface {
	RegisterAgent(context.Context, *AgentRegistrationRequest) (*AgentRegistrationConfirmation, error)
	UnRegisterAgent(context.Context, *Mrn) (*Confirmation, error)
	PingPong(context.Context, *Ping) (*Pong, error)
	HealthCheck(context.Context, *AgentInfo) (*AgentCheckinResponse, error)
}

// client implementation

type AgentManagerClient struct {
	ranger.Client
	httpclient ranger.HTTPClient
	prefix     string
}

func NewAgentManagerClient(addr string, client ranger.HTTPClient, plugins ...ranger.ClientPlugin) (*AgentManagerClient, error) {
	base, err := url.Parse(ranger.SanitizeUrl(addr))
	if err != nil {
		return nil, err
	}

	u, err := url.Parse("./AgentManager")
	if err != nil {
		return nil, err
	}

	serviceClient := &AgentManagerClient{
		httpclient: client,
		prefix:     base.ResolveReference(u).String(),
	}
	serviceClient.AddPlugins(plugins...)
	return serviceClient, nil
}
func (c *AgentManagerClient) RegisterAgent(ctx context.Context, in *AgentRegistrationRequest) (*AgentRegistrationConfirmation, error) {
	out := new(AgentRegistrationConfirmation)
	err := c.DoClientRequest(ctx, c.httpclient, strings.Join([]string{c.prefix, "/RegisterAgent"}, ""), in, out)
	return out, err
}
func (c *AgentManagerClient) UnRegisterAgent(ctx context.Context, in *Mrn) (*Confirmation, error) {
	out := new(Confirmation)
	err := c.DoClientRequest(ctx, c.httpclient, strings.Join([]string{c.prefix, "/UnRegisterAgent"}, ""), in, out)
	return out, err
}
func (c *AgentManagerClient) PingPong(ctx context.Context, in *Ping) (*Pong, error) {
	out := new(Pong)
	err := c.DoClientRequest(ctx, c.httpclient, strings.Join([]string{c.prefix, "/PingPong"}, ""), in, out)
	return out, err
}
func (c *AgentManagerClient) HealthCheck(ctx context.Context, in *AgentInfo) (*AgentCheckinResponse, error) {
	out := new(AgentCheckinResponse)
	err := c.DoClientRequest(ctx, c.httpclient, strings.Join([]string{c.prefix, "/HealthCheck"}, ""), in, out)
	return out, err
}

// server implementation

type AgentManagerServerOption func(s *AgentManagerServer)

func WithUnknownFieldsForAgentManagerServer() AgentManagerServerOption {
	return func(s *AgentManagerServer) {
		s.allowUnknownFields = true
	}
}

func NewAgentManagerServer(handler AgentManager, opts ...AgentManagerServerOption) http.Handler {
	srv := &AgentManagerServer{
		handler: handler,
	}

	for i := range opts {
		opts[i](srv)
	}

	service := ranger.Service{
		Name: "AgentManager",
		Methods: map[string]ranger.Method{
			"RegisterAgent":   srv.RegisterAgent,
			"UnRegisterAgent": srv.UnRegisterAgent,
			"PingPong":        srv.PingPong,
			"HealthCheck":     srv.HealthCheck,
		},
	}
	return ranger.NewRPCServer(&service)
}

type AgentManagerServer struct {
	handler            AgentManager
	allowUnknownFields bool
}

func (p *AgentManagerServer) RegisterAgent(ctx context.Context, reqBytes *[]byte) (pb.Message, error) {
	var req AgentRegistrationRequest
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
	return p.handler.RegisterAgent(ctx, &req)
}
func (p *AgentManagerServer) UnRegisterAgent(ctx context.Context, reqBytes *[]byte) (pb.Message, error) {
	var req Mrn
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
	return p.handler.UnRegisterAgent(ctx, &req)
}
func (p *AgentManagerServer) PingPong(ctx context.Context, reqBytes *[]byte) (pb.Message, error) {
	var req Ping
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
	return p.handler.PingPong(ctx, &req)
}
func (p *AgentManagerServer) HealthCheck(ctx context.Context, reqBytes *[]byte) (pb.Message, error) {
	var req AgentInfo
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
	return p.handler.HealthCheck(ctx, &req)
}

// service interface definition

type SecureTokenService interface {
	ExchangeSSH(context.Context, *ExchangeSSHKeyRequest) (*ExchangeSSHKeyResponse, error)
}

// client implementation

type SecureTokenServiceClient struct {
	ranger.Client
	httpclient ranger.HTTPClient
	prefix     string
}

func NewSecureTokenServiceClient(addr string, client ranger.HTTPClient, plugins ...ranger.ClientPlugin) (*SecureTokenServiceClient, error) {
	base, err := url.Parse(ranger.SanitizeUrl(addr))
	if err != nil {
		return nil, err
	}

	u, err := url.Parse("./SecureTokenService")
	if err != nil {
		return nil, err
	}

	serviceClient := &SecureTokenServiceClient{
		httpclient: client,
		prefix:     base.ResolveReference(u).String(),
	}
	serviceClient.AddPlugins(plugins...)
	return serviceClient, nil
}
func (c *SecureTokenServiceClient) ExchangeSSH(ctx context.Context, in *ExchangeSSHKeyRequest) (*ExchangeSSHKeyResponse, error) {
	out := new(ExchangeSSHKeyResponse)
	err := c.DoClientRequest(ctx, c.httpclient, strings.Join([]string{c.prefix, "/ExchangeSSH"}, ""), in, out)
	return out, err
}

// server implementation

type SecureTokenServiceServerOption func(s *SecureTokenServiceServer)

func WithUnknownFieldsForSecureTokenServiceServer() SecureTokenServiceServerOption {
	return func(s *SecureTokenServiceServer) {
		s.allowUnknownFields = true
	}
}

func NewSecureTokenServiceServer(handler SecureTokenService, opts ...SecureTokenServiceServerOption) http.Handler {
	srv := &SecureTokenServiceServer{
		handler: handler,
	}

	for i := range opts {
		opts[i](srv)
	}

	service := ranger.Service{
		Name: "SecureTokenService",
		Methods: map[string]ranger.Method{
			"ExchangeSSH": srv.ExchangeSSH,
		},
	}
	return ranger.NewRPCServer(&service)
}

type SecureTokenServiceServer struct {
	handler            SecureTokenService
	allowUnknownFields bool
}

func (p *SecureTokenServiceServer) ExchangeSSH(ctx context.Context, reqBytes *[]byte) (pb.Message, error) {
	var req ExchangeSSHKeyRequest
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
	return p.handler.ExchangeSSH(ctx, &req)
}
