package service

//go:generate protoc --proto_path=../../resources:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. service.proto

import (
	context "context"
	json "encoding/json"

	"go.mondoo.com/cnquery/resources"
)

type Server struct {
	Registry *resources.Registry
	Runtime  *resources.Runtime
}

// List out all resources
func (server *Server) ListResources(context.Context, *Empty) (*ResourceList, error) {
	return &ResourceList{
		Resources: server.Registry.Names(),
	}, nil
}

// GetSchema of resources
func (server *Server) GetSchema(context.Context, *Empty) (*resources.Schema, error) {
	return server.Registry.Schema(), nil
}

// ListFields TODO: probably not necessary anymore
func (server *Server) ListFields(ctx context.Context, q *FieldsQuery) (*Fields, error) {
	res, err := server.Registry.Fields(q.Name)
	if err != nil {
		return nil, err
	}
	return &Fields{Fields: res}, nil
}

// CreateResource from args
func (server *Server) CreateResource(ctx context.Context, q *ResourceArguments) (*resources.ResourceID, error) {
	args := []interface{}{}
	for k, v := range q.Named {
		args = append(args, k, v)
	}

	res, err := server.Runtime.CreateResource(q.Name, args)
	if err != nil {
		// TODO return unavailable return
		return nil, err
	}
	return &res.MqlResource().ResourceID, nil
}

// GetField essentially returns the result of a field
// this would return either a resource or raw data
func (server *Server) GetField(ctx context.Context, q *FieldArguments) (*FieldReturn, error) {
	r, err := server.Runtime.GetResource(q.Name, q.Id)
	if err != nil {
		// TODO: return unavailable error
		return nil, err
	}

	res, err := r.Field(q.Field)
	if err != nil {
		// TODO: return unavailable error with message "Failed to get field "+q.Field+" from resource "+q.Name+" with id "+q.Id+"
		return nil, err
	}

	bytes, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}
	return &FieldReturn{Data: bytes}, nil
}
