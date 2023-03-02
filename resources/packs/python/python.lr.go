// Code generated by resources. DO NOT EDIT.
package python

import (
	"errors"
	"fmt"
	"time"

	"go.mondoo.com/cnquery/resources"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources/packs/core"
)

// Init all resources into the registry
func Init(registry *resources.Registry) {
	registry.AddFactory("python", newPython)
	registry.AddFactory("python.package", newPythonPackage)
}

// Python resource interface
type Python interface {
	MqlResource() (*resources.Resource)
	MqlCompute(string) error
	Field(string) (interface{}, error)
	Register(string) error
	Validate() error
	Path() (string, error)
	Packages() ([]interface{}, error)
	Children() ([]interface{}, error)
}

// mqlPython for the python resource
type mqlPython struct {
	*resources.Resource
}

// MqlResource to retrieve the underlying resource info
func (s *mqlPython) MqlResource() *resources.Resource {
	return s.Resource
}

// create a new instance of the python resource
func newPython(runtime *resources.Runtime, args *resources.Args) (interface{}, error) {
	// User hooks
	var err error
	res := mqlPython{runtime.NewResource("python")}
	var existing Python
	args, existing, err = res.init(args)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// assign all named fields
	var id string

	now := time.Now().Unix()
	for name, val := range *args {
		if val == nil {
			res.Cache.Store(name, &resources.CacheEntry{Data: val, Valid: true, Timestamp: now})
			continue
		}

		switch name {
		case "path":
			if _, ok := val.(string); !ok {
				return nil, errors.New("Failed to initialize \"python\", its \"path\" argument has the wrong type (expected type \"string\")")
			}
		case "packages":
			if _, ok := val.([]interface{}); !ok {
				return nil, errors.New("Failed to initialize \"python\", its \"packages\" argument has the wrong type (expected type \"[]interface{}\")")
			}
		case "children":
			if _, ok := val.([]interface{}); !ok {
				return nil, errors.New("Failed to initialize \"python\", its \"children\" argument has the wrong type (expected type \"[]interface{}\")")
			}
		case "__id":
			idVal, ok := val.(string)
			if !ok {
				return nil, errors.New("Failed to initialize \"python\", its \"__id\" argument has the wrong type (expected type \"string\")")
			}
			id = idVal
		default:
			return nil, errors.New("Initialized python with unknown argument " + name)
		}
		res.Cache.Store(name, &resources.CacheEntry{Data: val, Valid: true, Timestamp: now})
	}

	// Get the ID
	if id == "" {
		res.Resource.Id, err = res.id()
		if err != nil {
			return nil, err
		}
	} else {
		res.Resource.Id = id
	}

	return &res, nil
}

func (s *mqlPython) Validate() error {
	// required arguments
	if _, ok := s.Cache.Load("path"); !ok {
		return errors.New("Initialized \"python\" resource without a \"path\". This field is required.")
	}

	return nil
}

// Register accessor autogenerated
func (s *mqlPython) Register(name string) error {
	log.Trace().Str("field", name).Msg("[python].Register")
	switch name {
	case "path":
		return nil
	case "packages":
		return nil
	case "children":
		return nil
	default:
		return errors.New("Cannot find field '" + name + "' in \"python\" resource")
	}
}

// Field accessor autogenerated
func (s *mqlPython) Field(name string) (interface{}, error) {
	log.Trace().Str("field", name).Msg("[python].Field")
	switch name {
	case "path":
		return s.Path()
	case "packages":
		return s.Packages()
	case "children":
		return s.Children()
	default:
		return nil, fmt.Errorf("Cannot find field '" + name + "' in \"python\" resource")
	}
}

// Path accessor autogenerated
func (s *mqlPython) Path() (string, error) {
	res, ok := s.Cache.Load("path")
	if !ok || !res.Valid {
		return "", errors.New("\"python\" failed: no value provided for static field \"path\"")
	}
	if res.Error != nil {
		return "", res.Error
	}
	tres, ok := res.Data.(string)
	if !ok {
		return "", fmt.Errorf("\"python\" failed to cast field \"path\" to the right type (string): %#v", res)
	}
	return tres, nil
}

// Packages accessor autogenerated
func (s *mqlPython) Packages() ([]interface{}, error) {
	res, ok := s.Cache.Load("packages")
	if !ok || !res.Valid {
		if err := s.ComputePackages(); err != nil {
			return nil, err
		}
		res, ok = s.Cache.Load("packages")
		if !ok {
			return nil, errors.New("\"python\" calculated \"packages\" but didn't find its value in cache.")
		}
		s.MotorRuntime.Trigger(s, "packages")
	}
	if res.Error != nil {
		return nil, res.Error
	}
	tres, ok := res.Data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("\"python\" failed to cast field \"packages\" to the right type ([]interface{}): %#v", res)
	}
	return tres, nil
}

// Children accessor autogenerated
func (s *mqlPython) Children() ([]interface{}, error) {
	res, ok := s.Cache.Load("children")
	if !ok || !res.Valid {
		if err := s.ComputeChildren(); err != nil {
			return nil, err
		}
		res, ok = s.Cache.Load("children")
		if !ok {
			return nil, errors.New("\"python\" calculated \"children\" but didn't find its value in cache.")
		}
		s.MotorRuntime.Trigger(s, "children")
	}
	if res.Error != nil {
		return nil, res.Error
	}
	tres, ok := res.Data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("\"python\" failed to cast field \"children\" to the right type ([]interface{}): %#v", res)
	}
	return tres, nil
}

// Compute accessor autogenerated
func (s *mqlPython) MqlCompute(name string) error {
	log.Trace().Str("field", name).Msg("[python].MqlCompute")
	switch name {
	case "path":
		return nil
	case "packages":
		return s.ComputePackages()
	case "children":
		return s.ComputeChildren()
	default:
		return errors.New("Cannot find field '" + name + "' in \"python\" resource")
	}
}

// ComputePackages computer autogenerated
func (s *mqlPython) ComputePackages() error {
	var err error
	if _, ok := s.Cache.Load("packages"); ok {
		return nil
	}
	vres, err := s.GetPackages()
	if _, ok := err.(resources.NotReadyError); ok {
		return err
	}
	s.Cache.Store("packages", &resources.CacheEntry{Data: vres, Valid: true, Error: err, Timestamp: time.Now().Unix()})
	return nil
}

// ComputeChildren computer autogenerated
func (s *mqlPython) ComputeChildren() error {
	var err error
	if _, ok := s.Cache.Load("children"); ok {
		return nil
	}
	vres, err := s.GetChildren()
	if _, ok := err.(resources.NotReadyError); ok {
		return err
	}
	s.Cache.Store("children", &resources.CacheEntry{Data: vres, Valid: true, Error: err, Timestamp: time.Now().Unix()})
	return nil
}

// PythonPackage resource interface
type PythonPackage interface {
	MqlResource() (*resources.Resource)
	MqlCompute(string) error
	Field(string) (interface{}, error)
	Register(string) error
	Validate() error
	Id() (string, error)
	Name() (string, error)
	File() (core.File, error)
	Version() (string, error)
	License() (string, error)
	Author() (string, error)
	Summary() (string, error)
}

// mqlPythonPackage for the python.package resource
type mqlPythonPackage struct {
	*resources.Resource
}

// MqlResource to retrieve the underlying resource info
func (s *mqlPythonPackage) MqlResource() *resources.Resource {
	return s.Resource
}

// create a new instance of the python.package resource
func newPythonPackage(runtime *resources.Runtime, args *resources.Args) (interface{}, error) {
	// User hooks
	var err error
	res := mqlPythonPackage{runtime.NewResource("python.package")}
	var existing PythonPackage
	args, existing, err = res.init(args)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// assign all named fields
	var id string

	now := time.Now().Unix()
	for name, val := range *args {
		if val == nil {
			res.Cache.Store(name, &resources.CacheEntry{Data: val, Valid: true, Timestamp: now})
			continue
		}

		switch name {
		case "id":
			if _, ok := val.(string); !ok {
				return nil, errors.New("Failed to initialize \"python.package\", its \"id\" argument has the wrong type (expected type \"string\")")
			}
		case "name":
			if _, ok := val.(string); !ok {
				return nil, errors.New("Failed to initialize \"python.package\", its \"name\" argument has the wrong type (expected type \"string\")")
			}
		case "file":
			if _, ok := val.(core.File); !ok {
				return nil, errors.New("Failed to initialize \"python.package\", its \"file\" argument has the wrong type (expected type \"core.File\")")
			}
		case "version":
			if _, ok := val.(string); !ok {
				return nil, errors.New("Failed to initialize \"python.package\", its \"version\" argument has the wrong type (expected type \"string\")")
			}
		case "license":
			if _, ok := val.(string); !ok {
				return nil, errors.New("Failed to initialize \"python.package\", its \"license\" argument has the wrong type (expected type \"string\")")
			}
		case "author":
			if _, ok := val.(string); !ok {
				return nil, errors.New("Failed to initialize \"python.package\", its \"author\" argument has the wrong type (expected type \"string\")")
			}
		case "summary":
			if _, ok := val.(string); !ok {
				return nil, errors.New("Failed to initialize \"python.package\", its \"summary\" argument has the wrong type (expected type \"string\")")
			}
		case "__id":
			idVal, ok := val.(string)
			if !ok {
				return nil, errors.New("Failed to initialize \"python.package\", its \"__id\" argument has the wrong type (expected type \"string\")")
			}
			id = idVal
		default:
			return nil, errors.New("Initialized python.package with unknown argument " + name)
		}
		res.Cache.Store(name, &resources.CacheEntry{Data: val, Valid: true, Timestamp: now})
	}

	// Get the ID
	if id == "" {
		res.Resource.Id, err = res.id()
		if err != nil {
			return nil, err
		}
	} else {
		res.Resource.Id = id
	}

	return &res, nil
}

func (s *mqlPythonPackage) Validate() error {
	// required arguments
	if _, ok := s.Cache.Load("id"); !ok {
		return errors.New("Initialized \"python.package\" resource without a \"id\". This field is required.")
	}
	if _, ok := s.Cache.Load("name"); !ok {
		return errors.New("Initialized \"python.package\" resource without a \"name\". This field is required.")
	}
	if _, ok := s.Cache.Load("file"); !ok {
		return errors.New("Initialized \"python.package\" resource without a \"file\". This field is required.")
	}
	if _, ok := s.Cache.Load("version"); !ok {
		return errors.New("Initialized \"python.package\" resource without a \"version\". This field is required.")
	}
	if _, ok := s.Cache.Load("license"); !ok {
		return errors.New("Initialized \"python.package\" resource without a \"license\". This field is required.")
	}
	if _, ok := s.Cache.Load("author"); !ok {
		return errors.New("Initialized \"python.package\" resource without a \"author\". This field is required.")
	}
	if _, ok := s.Cache.Load("summary"); !ok {
		return errors.New("Initialized \"python.package\" resource without a \"summary\". This field is required.")
	}

	return nil
}

// Register accessor autogenerated
func (s *mqlPythonPackage) Register(name string) error {
	log.Trace().Str("field", name).Msg("[python.package].Register")
	switch name {
	case "id":
		return nil
	case "name":
		return nil
	case "file":
		return nil
	case "version":
		return nil
	case "license":
		return nil
	case "author":
		return nil
	case "summary":
		return nil
	default:
		return errors.New("Cannot find field '" + name + "' in \"python.package\" resource")
	}
}

// Field accessor autogenerated
func (s *mqlPythonPackage) Field(name string) (interface{}, error) {
	log.Trace().Str("field", name).Msg("[python.package].Field")
	switch name {
	case "id":
		return s.Id()
	case "name":
		return s.Name()
	case "file":
		return s.File()
	case "version":
		return s.Version()
	case "license":
		return s.License()
	case "author":
		return s.Author()
	case "summary":
		return s.Summary()
	default:
		return nil, fmt.Errorf("Cannot find field '" + name + "' in \"python.package\" resource")
	}
}

// Id accessor autogenerated
func (s *mqlPythonPackage) Id() (string, error) {
	res, ok := s.Cache.Load("id")
	if !ok || !res.Valid {
		return "", errors.New("\"python.package\" failed: no value provided for static field \"id\"")
	}
	if res.Error != nil {
		return "", res.Error
	}
	tres, ok := res.Data.(string)
	if !ok {
		return "", fmt.Errorf("\"python.package\" failed to cast field \"id\" to the right type (string): %#v", res)
	}
	return tres, nil
}

// Name accessor autogenerated
func (s *mqlPythonPackage) Name() (string, error) {
	res, ok := s.Cache.Load("name")
	if !ok || !res.Valid {
		return "", errors.New("\"python.package\" failed: no value provided for static field \"name\"")
	}
	if res.Error != nil {
		return "", res.Error
	}
	tres, ok := res.Data.(string)
	if !ok {
		return "", fmt.Errorf("\"python.package\" failed to cast field \"name\" to the right type (string): %#v", res)
	}
	return tres, nil
}

// File accessor autogenerated
func (s *mqlPythonPackage) File() (core.File, error) {
	res, ok := s.Cache.Load("file")
	if !ok || !res.Valid {
		return nil, errors.New("\"python.package\" failed: no value provided for static field \"file\"")
	}
	if res.Error != nil {
		return nil, res.Error
	}
	tres, ok := res.Data.(core.File)
	if !ok {
		return nil, fmt.Errorf("\"python.package\" failed to cast field \"file\" to the right type (core.File): %#v", res)
	}
	return tres, nil
}

// Version accessor autogenerated
func (s *mqlPythonPackage) Version() (string, error) {
	res, ok := s.Cache.Load("version")
	if !ok || !res.Valid {
		return "", errors.New("\"python.package\" failed: no value provided for static field \"version\"")
	}
	if res.Error != nil {
		return "", res.Error
	}
	tres, ok := res.Data.(string)
	if !ok {
		return "", fmt.Errorf("\"python.package\" failed to cast field \"version\" to the right type (string): %#v", res)
	}
	return tres, nil
}

// License accessor autogenerated
func (s *mqlPythonPackage) License() (string, error) {
	res, ok := s.Cache.Load("license")
	if !ok || !res.Valid {
		return "", errors.New("\"python.package\" failed: no value provided for static field \"license\"")
	}
	if res.Error != nil {
		return "", res.Error
	}
	tres, ok := res.Data.(string)
	if !ok {
		return "", fmt.Errorf("\"python.package\" failed to cast field \"license\" to the right type (string): %#v", res)
	}
	return tres, nil
}

// Author accessor autogenerated
func (s *mqlPythonPackage) Author() (string, error) {
	res, ok := s.Cache.Load("author")
	if !ok || !res.Valid {
		return "", errors.New("\"python.package\" failed: no value provided for static field \"author\"")
	}
	if res.Error != nil {
		return "", res.Error
	}
	tres, ok := res.Data.(string)
	if !ok {
		return "", fmt.Errorf("\"python.package\" failed to cast field \"author\" to the right type (string): %#v", res)
	}
	return tres, nil
}

// Summary accessor autogenerated
func (s *mqlPythonPackage) Summary() (string, error) {
	res, ok := s.Cache.Load("summary")
	if !ok || !res.Valid {
		return "", errors.New("\"python.package\" failed: no value provided for static field \"summary\"")
	}
	if res.Error != nil {
		return "", res.Error
	}
	tres, ok := res.Data.(string)
	if !ok {
		return "", fmt.Errorf("\"python.package\" failed to cast field \"summary\" to the right type (string): %#v", res)
	}
	return tres, nil
}

// Compute accessor autogenerated
func (s *mqlPythonPackage) MqlCompute(name string) error {
	log.Trace().Str("field", name).Msg("[python.package].MqlCompute")
	switch name {
	case "id":
		return nil
	case "name":
		return nil
	case "file":
		return nil
	case "version":
		return nil
	case "license":
		return nil
	case "author":
		return nil
	case "summary":
		return nil
	default:
		return errors.New("Cannot find field '" + name + "' in \"python.package\" resource")
	}
}

