// Code generated by resources. DO NOT EDIT.
package gitlab

import (
	"errors"
	"fmt"
	"time"

	"go.mondoo.com/cnquery/resources"
	"github.com/rs/zerolog/log"
)

// Init all resources into the registry
func Init(registry *resources.Registry) {
	registry.AddFactory("gitlab.group", newGitlabGroup)
	registry.AddFactory("gitlab.project", newGitlabProject)
}

// GitlabGroup resource interface
type GitlabGroup interface {
	MqlResource() (*resources.Resource)
	MqlCompute(string) error
	Field(string) (interface{}, error)
	Register(string) error
	Validate() error
	Id() (int64, error)
	Name() (string, error)
	Path() (string, error)
	Description() (string, error)
	Visibility() (string, error)
	RequireTwoFactorAuthentication() (bool, error)
	Projects() ([]interface{}, error)
}

// mqlGitlabGroup for the gitlab.group resource
type mqlGitlabGroup struct {
	*resources.Resource
}

// MqlResource to retrieve the underlying resource info
func (s *mqlGitlabGroup) MqlResource() *resources.Resource {
	return s.Resource
}

// create a new instance of the gitlab.group resource
func newGitlabGroup(runtime *resources.Runtime, args *resources.Args) (interface{}, error) {
	// User hooks
	var err error
	res := mqlGitlabGroup{runtime.NewResource("gitlab.group")}
	var existing GitlabGroup
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
			if _, ok := val.(int64); !ok {
				return nil, errors.New("Failed to initialize \"gitlab.group\", its \"id\" argument has the wrong type (expected type \"int64\")")
			}
		case "name":
			if _, ok := val.(string); !ok {
				return nil, errors.New("Failed to initialize \"gitlab.group\", its \"name\" argument has the wrong type (expected type \"string\")")
			}
		case "path":
			if _, ok := val.(string); !ok {
				return nil, errors.New("Failed to initialize \"gitlab.group\", its \"path\" argument has the wrong type (expected type \"string\")")
			}
		case "description":
			if _, ok := val.(string); !ok {
				return nil, errors.New("Failed to initialize \"gitlab.group\", its \"description\" argument has the wrong type (expected type \"string\")")
			}
		case "visibility":
			if _, ok := val.(string); !ok {
				return nil, errors.New("Failed to initialize \"gitlab.group\", its \"visibility\" argument has the wrong type (expected type \"string\")")
			}
		case "requireTwoFactorAuthentication":
			if _, ok := val.(bool); !ok {
				return nil, errors.New("Failed to initialize \"gitlab.group\", its \"requireTwoFactorAuthentication\" argument has the wrong type (expected type \"bool\")")
			}
		case "projects":
			if _, ok := val.([]interface{}); !ok {
				return nil, errors.New("Failed to initialize \"gitlab.group\", its \"projects\" argument has the wrong type (expected type \"[]interface{}\")")
			}
		case "__id":
			idVal, ok := val.(string)
			if !ok {
				return nil, errors.New("Failed to initialize \"gitlab.group\", its \"__id\" argument has the wrong type (expected type \"string\")")
			}
			id = idVal
		default:
			return nil, errors.New("Initialized gitlab.group with unknown argument " + name)
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

func (s *mqlGitlabGroup) Validate() error {
	// required arguments
	if _, ok := s.Cache.Load("id"); !ok {
		return errors.New("Initialized \"gitlab.group\" resource without a \"id\". This field is required.")
	}
	if _, ok := s.Cache.Load("name"); !ok {
		return errors.New("Initialized \"gitlab.group\" resource without a \"name\". This field is required.")
	}
	if _, ok := s.Cache.Load("path"); !ok {
		return errors.New("Initialized \"gitlab.group\" resource without a \"path\". This field is required.")
	}
	if _, ok := s.Cache.Load("description"); !ok {
		return errors.New("Initialized \"gitlab.group\" resource without a \"description\". This field is required.")
	}
	if _, ok := s.Cache.Load("visibility"); !ok {
		return errors.New("Initialized \"gitlab.group\" resource without a \"visibility\". This field is required.")
	}
	if _, ok := s.Cache.Load("requireTwoFactorAuthentication"); !ok {
		return errors.New("Initialized \"gitlab.group\" resource without a \"requireTwoFactorAuthentication\". This field is required.")
	}

	return nil
}

// Register accessor autogenerated
func (s *mqlGitlabGroup) Register(name string) error {
	log.Trace().Str("field", name).Msg("[gitlab.group].Register")
	switch name {
	case "id":
		return nil
	case "name":
		return nil
	case "path":
		return nil
	case "description":
		return nil
	case "visibility":
		return nil
	case "requireTwoFactorAuthentication":
		return nil
	case "projects":
		return nil
	default:
		return errors.New("Cannot find field '" + name + "' in \"gitlab.group\" resource")
	}
}

// Field accessor autogenerated
func (s *mqlGitlabGroup) Field(name string) (interface{}, error) {
	log.Trace().Str("field", name).Msg("[gitlab.group].Field")
	switch name {
	case "id":
		return s.Id()
	case "name":
		return s.Name()
	case "path":
		return s.Path()
	case "description":
		return s.Description()
	case "visibility":
		return s.Visibility()
	case "requireTwoFactorAuthentication":
		return s.RequireTwoFactorAuthentication()
	case "projects":
		return s.Projects()
	default:
		return nil, fmt.Errorf("Cannot find field '" + name + "' in \"gitlab.group\" resource")
	}
}

// Id accessor autogenerated
func (s *mqlGitlabGroup) Id() (int64, error) {
	res, ok := s.Cache.Load("id")
	if !ok || !res.Valid {
		return 0, errors.New("\"gitlab.group\" failed: no value provided for static field \"id\"")
	}
	if res.Error != nil {
		return 0, res.Error
	}
	tres, ok := res.Data.(int64)
	if !ok {
		return 0, fmt.Errorf("\"gitlab.group\" failed to cast field \"id\" to the right type (int64): %#v", res)
	}
	return tres, nil
}

// Name accessor autogenerated
func (s *mqlGitlabGroup) Name() (string, error) {
	res, ok := s.Cache.Load("name")
	if !ok || !res.Valid {
		return "", errors.New("\"gitlab.group\" failed: no value provided for static field \"name\"")
	}
	if res.Error != nil {
		return "", res.Error
	}
	tres, ok := res.Data.(string)
	if !ok {
		return "", fmt.Errorf("\"gitlab.group\" failed to cast field \"name\" to the right type (string): %#v", res)
	}
	return tres, nil
}

// Path accessor autogenerated
func (s *mqlGitlabGroup) Path() (string, error) {
	res, ok := s.Cache.Load("path")
	if !ok || !res.Valid {
		return "", errors.New("\"gitlab.group\" failed: no value provided for static field \"path\"")
	}
	if res.Error != nil {
		return "", res.Error
	}
	tres, ok := res.Data.(string)
	if !ok {
		return "", fmt.Errorf("\"gitlab.group\" failed to cast field \"path\" to the right type (string): %#v", res)
	}
	return tres, nil
}

// Description accessor autogenerated
func (s *mqlGitlabGroup) Description() (string, error) {
	res, ok := s.Cache.Load("description")
	if !ok || !res.Valid {
		return "", errors.New("\"gitlab.group\" failed: no value provided for static field \"description\"")
	}
	if res.Error != nil {
		return "", res.Error
	}
	tres, ok := res.Data.(string)
	if !ok {
		return "", fmt.Errorf("\"gitlab.group\" failed to cast field \"description\" to the right type (string): %#v", res)
	}
	return tres, nil
}

// Visibility accessor autogenerated
func (s *mqlGitlabGroup) Visibility() (string, error) {
	res, ok := s.Cache.Load("visibility")
	if !ok || !res.Valid {
		return "", errors.New("\"gitlab.group\" failed: no value provided for static field \"visibility\"")
	}
	if res.Error != nil {
		return "", res.Error
	}
	tres, ok := res.Data.(string)
	if !ok {
		return "", fmt.Errorf("\"gitlab.group\" failed to cast field \"visibility\" to the right type (string): %#v", res)
	}
	return tres, nil
}

// RequireTwoFactorAuthentication accessor autogenerated
func (s *mqlGitlabGroup) RequireTwoFactorAuthentication() (bool, error) {
	res, ok := s.Cache.Load("requireTwoFactorAuthentication")
	if !ok || !res.Valid {
		return false, errors.New("\"gitlab.group\" failed: no value provided for static field \"requireTwoFactorAuthentication\"")
	}
	if res.Error != nil {
		return false, res.Error
	}
	tres, ok := res.Data.(bool)
	if !ok {
		return false, fmt.Errorf("\"gitlab.group\" failed to cast field \"requireTwoFactorAuthentication\" to the right type (bool): %#v", res)
	}
	return tres, nil
}

// Projects accessor autogenerated
func (s *mqlGitlabGroup) Projects() ([]interface{}, error) {
	res, ok := s.Cache.Load("projects")
	if !ok || !res.Valid {
		if err := s.ComputeProjects(); err != nil {
			return nil, err
		}
		res, ok = s.Cache.Load("projects")
		if !ok {
			return nil, errors.New("\"gitlab.group\" calculated \"projects\" but didn't find its value in cache.")
		}
		s.MotorRuntime.Trigger(s, "projects")
	}
	if res.Error != nil {
		return nil, res.Error
	}
	tres, ok := res.Data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("\"gitlab.group\" failed to cast field \"projects\" to the right type ([]interface{}): %#v", res)
	}
	return tres, nil
}

// Compute accessor autogenerated
func (s *mqlGitlabGroup) MqlCompute(name string) error {
	log.Trace().Str("field", name).Msg("[gitlab.group].MqlCompute")
	switch name {
	case "id":
		return nil
	case "name":
		return nil
	case "path":
		return nil
	case "description":
		return nil
	case "visibility":
		return nil
	case "requireTwoFactorAuthentication":
		return nil
	case "projects":
		return s.ComputeProjects()
	default:
		return errors.New("Cannot find field '" + name + "' in \"gitlab.group\" resource")
	}
}

// ComputeProjects computer autogenerated
func (s *mqlGitlabGroup) ComputeProjects() error {
	var err error
	if _, ok := s.Cache.Load("projects"); ok {
		return nil
	}
	vres, err := s.GetProjects()
	if _, ok := err.(resources.NotReadyError); ok {
		return err
	}
	s.Cache.Store("projects", &resources.CacheEntry{Data: vres, Valid: true, Error: err, Timestamp: time.Now().Unix()})
	return nil
}

// GitlabProject resource interface
type GitlabProject interface {
	MqlResource() (*resources.Resource)
	MqlCompute(string) error
	Field(string) (interface{}, error)
	Register(string) error
	Validate() error
	Id() (int64, error)
	Name() (string, error)
	Path() (string, error)
	Description() (string, error)
	Visibility() (string, error)
}

// mqlGitlabProject for the gitlab.project resource
type mqlGitlabProject struct {
	*resources.Resource
}

// MqlResource to retrieve the underlying resource info
func (s *mqlGitlabProject) MqlResource() *resources.Resource {
	return s.Resource
}

// create a new instance of the gitlab.project resource
func newGitlabProject(runtime *resources.Runtime, args *resources.Args) (interface{}, error) {
	// User hooks
	var err error
	res := mqlGitlabProject{runtime.NewResource("gitlab.project")}
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
			if _, ok := val.(int64); !ok {
				return nil, errors.New("Failed to initialize \"gitlab.project\", its \"id\" argument has the wrong type (expected type \"int64\")")
			}
		case "name":
			if _, ok := val.(string); !ok {
				return nil, errors.New("Failed to initialize \"gitlab.project\", its \"name\" argument has the wrong type (expected type \"string\")")
			}
		case "path":
			if _, ok := val.(string); !ok {
				return nil, errors.New("Failed to initialize \"gitlab.project\", its \"path\" argument has the wrong type (expected type \"string\")")
			}
		case "description":
			if _, ok := val.(string); !ok {
				return nil, errors.New("Failed to initialize \"gitlab.project\", its \"description\" argument has the wrong type (expected type \"string\")")
			}
		case "visibility":
			if _, ok := val.(string); !ok {
				return nil, errors.New("Failed to initialize \"gitlab.project\", its \"visibility\" argument has the wrong type (expected type \"string\")")
			}
		case "__id":
			idVal, ok := val.(string)
			if !ok {
				return nil, errors.New("Failed to initialize \"gitlab.project\", its \"__id\" argument has the wrong type (expected type \"string\")")
			}
			id = idVal
		default:
			return nil, errors.New("Initialized gitlab.project with unknown argument " + name)
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

func (s *mqlGitlabProject) Validate() error {
	// required arguments
	if _, ok := s.Cache.Load("id"); !ok {
		return errors.New("Initialized \"gitlab.project\" resource without a \"id\". This field is required.")
	}
	if _, ok := s.Cache.Load("name"); !ok {
		return errors.New("Initialized \"gitlab.project\" resource without a \"name\". This field is required.")
	}
	if _, ok := s.Cache.Load("path"); !ok {
		return errors.New("Initialized \"gitlab.project\" resource without a \"path\". This field is required.")
	}
	if _, ok := s.Cache.Load("description"); !ok {
		return errors.New("Initialized \"gitlab.project\" resource without a \"description\". This field is required.")
	}
	if _, ok := s.Cache.Load("visibility"); !ok {
		return errors.New("Initialized \"gitlab.project\" resource without a \"visibility\". This field is required.")
	}

	return nil
}

// Register accessor autogenerated
func (s *mqlGitlabProject) Register(name string) error {
	log.Trace().Str("field", name).Msg("[gitlab.project].Register")
	switch name {
	case "id":
		return nil
	case "name":
		return nil
	case "path":
		return nil
	case "description":
		return nil
	case "visibility":
		return nil
	default:
		return errors.New("Cannot find field '" + name + "' in \"gitlab.project\" resource")
	}
}

// Field accessor autogenerated
func (s *mqlGitlabProject) Field(name string) (interface{}, error) {
	log.Trace().Str("field", name).Msg("[gitlab.project].Field")
	switch name {
	case "id":
		return s.Id()
	case "name":
		return s.Name()
	case "path":
		return s.Path()
	case "description":
		return s.Description()
	case "visibility":
		return s.Visibility()
	default:
		return nil, fmt.Errorf("Cannot find field '" + name + "' in \"gitlab.project\" resource")
	}
}

// Id accessor autogenerated
func (s *mqlGitlabProject) Id() (int64, error) {
	res, ok := s.Cache.Load("id")
	if !ok || !res.Valid {
		return 0, errors.New("\"gitlab.project\" failed: no value provided for static field \"id\"")
	}
	if res.Error != nil {
		return 0, res.Error
	}
	tres, ok := res.Data.(int64)
	if !ok {
		return 0, fmt.Errorf("\"gitlab.project\" failed to cast field \"id\" to the right type (int64): %#v", res)
	}
	return tres, nil
}

// Name accessor autogenerated
func (s *mqlGitlabProject) Name() (string, error) {
	res, ok := s.Cache.Load("name")
	if !ok || !res.Valid {
		return "", errors.New("\"gitlab.project\" failed: no value provided for static field \"name\"")
	}
	if res.Error != nil {
		return "", res.Error
	}
	tres, ok := res.Data.(string)
	if !ok {
		return "", fmt.Errorf("\"gitlab.project\" failed to cast field \"name\" to the right type (string): %#v", res)
	}
	return tres, nil
}

// Path accessor autogenerated
func (s *mqlGitlabProject) Path() (string, error) {
	res, ok := s.Cache.Load("path")
	if !ok || !res.Valid {
		return "", errors.New("\"gitlab.project\" failed: no value provided for static field \"path\"")
	}
	if res.Error != nil {
		return "", res.Error
	}
	tres, ok := res.Data.(string)
	if !ok {
		return "", fmt.Errorf("\"gitlab.project\" failed to cast field \"path\" to the right type (string): %#v", res)
	}
	return tres, nil
}

// Description accessor autogenerated
func (s *mqlGitlabProject) Description() (string, error) {
	res, ok := s.Cache.Load("description")
	if !ok || !res.Valid {
		return "", errors.New("\"gitlab.project\" failed: no value provided for static field \"description\"")
	}
	if res.Error != nil {
		return "", res.Error
	}
	tres, ok := res.Data.(string)
	if !ok {
		return "", fmt.Errorf("\"gitlab.project\" failed to cast field \"description\" to the right type (string): %#v", res)
	}
	return tres, nil
}

// Visibility accessor autogenerated
func (s *mqlGitlabProject) Visibility() (string, error) {
	res, ok := s.Cache.Load("visibility")
	if !ok || !res.Valid {
		return "", errors.New("\"gitlab.project\" failed: no value provided for static field \"visibility\"")
	}
	if res.Error != nil {
		return "", res.Error
	}
	tres, ok := res.Data.(string)
	if !ok {
		return "", fmt.Errorf("\"gitlab.project\" failed to cast field \"visibility\" to the right type (string): %#v", res)
	}
	return tres, nil
}

// Compute accessor autogenerated
func (s *mqlGitlabProject) MqlCompute(name string) error {
	log.Trace().Str("field", name).Msg("[gitlab.project].MqlCompute")
	switch name {
	case "id":
		return nil
	case "name":
		return nil
	case "path":
		return nil
	case "description":
		return nil
	case "visibility":
		return nil
	default:
		return errors.New("Cannot find field '" + name + "' in \"gitlab.project\" resource")
	}
}

