package resources

type ResourceNotFound struct{}

func (e *ResourceNotFound) Error() string {
	return "could not find resource"
}

// ResourceFactory for creating a new resource instance
type ResourceFactory func(*Runtime, *Args) (interface{}, error)

// ResourceCls contains the resource factory and all metadata
type ResourceCls struct {
	Factory ResourceFactory
	ResourceInfo
}

func newResourceCls(name string) *ResourceCls {
	return &ResourceCls{ResourceInfo: ResourceInfo{
		Name:   name,
		Fields: make(map[string]*Field),
	}}
}

// Resource instance, tied to a motor runtime
type Resource struct {
	Cache        Cache    `json:"data"`
	MotorRuntime *Runtime `json:"-"`
	ResourceID
}

// UID combines the resources name and instance ID to a unique ID
func (r *Resource) UID() string {
	return r.Name + "\x00" + r.Id
}

// FieldUID provides a unique ID for this resource's given field
func (r *Resource) FieldUID(field string) string {
	return r.Name + "\x00" + r.Id + "\x00" + field
}

// ResourceType is a helper for all auto-generated resources via lr.
// They have a number of methods we need to access, and this is the
// helper to get all those methods
type ResourceType interface {
	// Retrieve the current (cached) value of a field
	Field(name string) (interface{}, error)
	// Register a field by (1) registering all callbacks to other resources
	// and fields and (2) triggering its initial run if it has no other dependencies
	Register(field string) error
	// Compute a field of the resource
	Compute(field string) error
	MqlResource() *Resource
	Validate() error
}
