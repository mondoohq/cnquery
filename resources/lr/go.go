package lr

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/types"
)

type goBuilder struct {
	data       string
	collector  *Collector
	ast        *LR
	errors     error
	packsInUse map[string]struct{}
}

// Go produced go code for the LR file
func Go(packageName string, ast *LR, collector *Collector) (string, error) {
	o := goBuilder{
		collector:  collector,
		ast:        ast,
		packsInUse: map[string]struct{}{},
	}

	o.goRegistryInit(ast.Resources)

	for i := range ast.Resources {
		err := o.goResource(ast.Resources[i])
		if err != nil {
			return o.data, err
		}
	}

	imports := ""
	for packName := range o.packsInUse {
		importPath, ok := ast.packPaths[packName]
		if !ok {
			return "", errors.New("cannot find import path for pack: " + packName)
		}

		imports += "\n\t" + strconv.Quote(importPath)
	}

	header := fmt.Sprintf(goHeader, packageName, imports)
	return header + o.data, nil
}

func (b *goBuilder) goRegistryInit(r []*Resource) {
	arr := []string{}
	for i := range r {
		arr = append(arr, "registry.AddFactory("+strconv.Quote(r[i].ID)+", new"+r[i].interfaceName(b)+")")
	}
	res := strings.Join(arr, "\n")

	b.data += `// Init all resources into the registry
func Init(registry *resources.Registry) {
` + indent(res, 1) + `
}

`
}

func (b *goBuilder) goResource(r *Resource) error {
	if r.ListType != nil {
		t := r.ListType.Type.Type
		args := r.ListType.Args

		// args of nil tell the compiler that this field needs to be pre-populated
		// however for list we don't have this logic, it is always computed
		if args == nil {
			args = &FieldArgs{}
		}

		field := &BasicField{
			ID:   "list",
			Args: args,
			Type: Type{ListType: &ListType{Type: Type{SimpleType: &SimpleType{t}}}},
		}

		r.Body.Fields = append(r.Body.Fields, &Field{BasicField: field})
	}

	for i, f := range r.Body.Fields {
		if f.Embeddable == nil {
			continue
		}
		var name string
		if f.Embeddable.Alias != nil {
			name = *f.Embeddable.Alias
		} else {
			// use the first part of the type name as a id, i.e. os for os.any
			// this wont work if there're are multiple embedded resources without aliases that share the same package, i.e os.any and os.base
			name = strings.Split(f.Embeddable.Type, ".")[0]
		}
		newField := &Field{
			Comments: f.Comments,
			BasicField: &BasicField{
				ID:         name,
				Type:       Type{SimpleType: &SimpleType{f.Embeddable.Type}},
				Args:       &FieldArgs{},
				isEmbedded: true,
			},
		}
		r.Body.Fields[i] = newField
	}

	b.goInterface(r)
	b.goStruct(r)
	b.goFactory(r)
	b.goRegister(r)
	b.goField(r)
	b.goCompute(r)
	return nil
}

func (b *goBuilder) goInterface(r *Resource) {
	fields := ""
	for _, f := range r.Body.Fields {
		var fieldType string
		// for embedded fields we want to return a ResourceType so we can initialize a context there
		if f.BasicField != nil {
			if f.BasicField.isEmbedded {
				fieldType = "resources.ResourceType"
			} else {
				fieldType = f.BasicField.Type.goType(b)
			}
			fields += "\t" + f.BasicField.methodname() + "() (" + fieldType + ", error)\n"
		}
	}

	b.data += fmt.Sprintf(`// %s resource interface
type %s interface {
	MqlResource() (*resources.Resource)
	Compute(string) error
	Field(string) (interface{}, error)
	Register(string) error
	Validate() error
%s}

`, r.interfaceName(b), r.interfaceName(b), fields)
}

func (b *goBuilder) goStruct(r *Resource) {
	b.data += fmt.Sprintf(`// %s for the %s resource
type %s struct {
	*resources.Resource
}

`, r.structName(b), r.ID, r.structName(b))

	b.data += fmt.Sprintf(`// MqlResource to retrieve the underlying resource info
func (s *%s) MqlResource() *resources.Resource {
	return s.Resource
}

`, r.structName(b))
}

func (b *goBuilder) goFactory(r *Resource) {
	args := ""
	for _, f := range r.Body.Fields {
		if f.BasicField == nil {
			// at this point all embeddable field should be converted to basic fields
			continue
		}
		args += fmt.Sprintf(`		case "%s":
			if _, ok := val.(%s); !ok {
				return nil, errors.New("Failed to initialize \"%s\", its \"%s\" argument has the wrong type (expected type \"%s\")")
			}
`, f.BasicField.ID, f.BasicField.Type.goType(b), r.ID, f.BasicField.ID, f.BasicField.Type.goType(b))
	}

	required := ""
	// TODO: staticFields?
	sfields := r.Body.staticFields()
	if len(sfields) == 0 {
		required += "\t// no required fields found\n"
	}
	for _, f := range sfields {
		required += fmt.Sprintf(`	if _, ok := s.Cache.Load("%s"); !ok {
		return errors.New("Initialized \"%s\" resource without a \"%s\". This field is required.")
	}
`, f.ID, r.ID, f.ID)
	}

	hasInit := b.collector.HasInit(r.structName(b))
	log.Debug().Bool("init", hasInit).Msg("dynamic calls for " + r.interfaceName(b))

	initcall := ""
	if b.collector.HasInit(r.structName(b)) {
		initcall = `var existing ` + r.interfaceName(b) + `
	args, existing, err = res.init(args)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	`
	}

	b.data += `// create a new instance of the ` + r.ID + ` resource
func new` + r.interfaceName(b) + `(runtime *resources.Runtime, args *resources.Args) (interface{}, error) {
	// User hooks
	var err error
	res := ` + r.structName(b) + `{runtime.NewResource("` + r.ID + `")}
	` + initcall + `// assign all named fields
	var id string

	now := time.Now().Unix()
	for name, val := range *args {
		if val == nil {
			res.Cache.Store(name, &resources.CacheEntry{Data: val, Valid: true, Timestamp: now})
			continue
		}

		switch name {
` + args + `		case "__id":
			idVal, ok := val.(string)
			if !ok {
				return nil, errors.New("Failed to initialize \"` + r.ID + `\", its \"__id\" argument has the wrong type (expected type \"string\")")
			}
			id = idVal
		default:
			return nil, errors.New("Initialized ` + r.ID + ` with unknown argument " + name)
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

func (s *` + r.structName(b) + `) Validate() error {
	// required arguments
` + required + `
	return nil
}

`
}

func goRegisterField(f *BasicField) string {
	// No Args means this field is static and cannot be computed
	// so there is nothing to register, but we need to trigger it
	if f.Args == nil {
		return fmt.Sprintf(`	case "%s":
		return nil
`, f.ID)
	}

	if len(f.Args.List) == 0 {
		return fmt.Sprintf(`	case "%s":
		return nil
`, f.ID)
	}

	l := []string{}
	for _, arg := range f.Args.List {
		l = append(l, fmt.Sprintf(
			`		if err = s.MotorRuntime.WatchAndCompute(s, "%s", s, "%s"); err != nil {
			return err
		}
`, arg.Type, f.ID))
	}
	return fmt.Sprintf(`	case "%s":
		var err error
%s		return nil
`, f.ID, strings.Join(l, "\n"))
}

func (b *goBuilder) goRegister(r *Resource) {
	fields := []string{}
	for _, f := range r.Body.Fields {
		if f.BasicField != nil {
			fields = append(fields, goRegisterField(f.BasicField))
		}
	}

	b.data += fmt.Sprintf(`// Register accessor autogenerated
func (s *%s) Register(name string) error {
	log.Trace().Str("field", name).Msg("[%s].Register")
	switch name {
%s	default:
		return errors.New("Cannot find field '" + name + "' in \"%s\" resource")
	}
}

`, r.structName(b), r.ID, strings.Join(fields, ""), r.ID)
}

func (b *goBuilder) goField(r *Resource) {
	caseField := []string{}
	for _, f := range r.Body.Fields {
		if f.BasicField == nil {
			continue
		}
		caseField = append(caseField, fmt.Sprintf(`	case "%s":
		return s.%s()
`, f.BasicField.ID, f.BasicField.goName()))
	}
	b.data += fmt.Sprintf(`// Field accessor autogenerated
func (s *%s) Field(name string) (interface{}, error) {
	log.Trace().Str("field", name).Msg("[%s].Field")
	switch name {
%s	default:
		return nil, fmt.Errorf("Cannot find field '" + name + "' in \"%s\" resource")
	}
}

`, r.structName(b), r.ID, strings.Join(caseField, ""), r.ID)

	for _, f := range r.Body.Fields {
		if f.BasicField == nil {
			continue
		}
		b.goFieldAccessor(r, f.BasicField)
	}
}

func (b *goBuilder) goCompute(r *Resource) {
	caseField := []string{}

	for _, f := range r.Body.Fields {
		if f.BasicField == nil {
			continue
		}
		// static fields don't have a compute call associated with them
		if f.BasicField.isStatic() {
			caseField = append(caseField, fmt.Sprintf(`	case "%s":
		return nil
`, f.BasicField.ID))
			continue
		}

		caseField = append(caseField, fmt.Sprintf(`	case "%s":
		return s.Compute%s()
`, f.BasicField.ID, f.BasicField.goName()))
	}

	b.data += fmt.Sprintf(`// Compute accessor autogenerated
func (s *%s) Compute(name string) error {
	log.Trace().Str("field", name).Msg("[%s].Compute")
	switch name {
%s	default:
		return errors.New("Cannot find field '" + name + "' in \"%s\" resource")
	}
}

`, r.structName(b), r.ID, strings.Join(caseField, ""), r.ID)

	for _, f := range r.Body.Fields {
		if f.BasicField == nil {
			continue
		}
		b.goFieldComputer(r, f.BasicField)
	}
}

func (b *goBuilder) goFieldAccessor(r *Resource, f *BasicField) {
	var notFound string
	if f.Args == nil {
		notFound = fmt.Sprintf(
			`return %s, errors.New("\"%s\" failed: no value provided for static field \"%s\"")`,
			f.Type.goZeroValue(), r.ID, f.ID)
	} else if len(f.Args.List) == 0 {
		// the case where we should easily retrieve the value from a dependency-free computation
		notFound = fmt.Sprintf(
			`if err := s.Compute%s(); err != nil {
			return %s, err
		}
		res, ok = s.Cache.Load("%s")
		if !ok {
			return %s, errors.New("\"%s\" calculated \"%s\" but didnt find its value in cache.")
		}
		s.MotorRuntime.Trigger(s, "%s")`,
			f.goName(), f.Type.goZeroValue(),
			f.ID,
			f.Type.goZeroValue(), r.ID, f.ID,
			f.ID,
		)
	} else {
		notFound = "return " + f.Type.goZeroValue() + ", resources.NotReadyError{}"
	}

	returnType := f.Type.goType(b)
	if f.isEmbedded {
		// for embedded resources we want to return a ResourceType to be able to initialize a context
		returnType = "resources.ResourceType"
	}
	b.data += fmt.Sprintf(`// %s accessor autogenerated
func (s *%s) %s() (%s, error) {
	res, ok := s.Cache.Load("%s")
	if !ok || !res.Valid {
		%s
	}
	if res.Error != nil {
		return %s, res.Error
	}
	tres, ok := res.Data.(%s)
	if !ok {
		return %s, fmt.Errorf("\"%s\" failed to cast field \"%s\" to the right type (%s): %%#v", res)
	}
	return tres, nil
}

`, f.goName(),
		r.structName(b), f.goName(), returnType,
		f.ID, notFound,
		f.Type.goZeroValue(),
		f.Type.goType(b),
		f.Type.goZeroValue(), r.ID, f.ID, f.Type.goType(b))
}

func (b *goBuilder) goFieldComputer(r *Resource, f *BasicField) {
	if f.Args == nil {
		return
	}

	argGetters := ""
	args := make([]string, len(f.Args.List))

	for i, arg := range f.Args.List {
		args[i] = "varg" + arg.goType(b)
		argGetters += fmt.Sprintf(`	varg%s, err := s.%s()
	if err != nil {
		if _, ok := err.(resources.NotReadyError); ok {
			return err
		}
		s.Cache.Store("%s", &resources.CacheEntry{Valid: true, Error: err, Timestamp: time.Now().Unix()})
		return nil
	}
`, arg.goType(b), arg.goType(b), f.ID)
	}

	// for fields that only compute a default value, only do this once
	if len(f.Args.List) == 0 {
		argGetters = `	if _, ok := s.Cache.Load("` + f.ID + `"); ok {
		return nil
	}
`
	}

	b.data += fmt.Sprintf(`// Compute%s computer autogenerated
func (s *%s) Compute%s() error {
	var err error
%s	vres, err := s.Get%s(%s)
	if _, ok := err.(resources.NotReadyError); ok {
		return err
	}
	s.Cache.Store("%s", &resources.CacheEntry{Data: vres, Valid: true, Error: err, Timestamp: time.Now().Unix()})
	return nil
}

`, f.goName(), r.structName(b), f.goName(), argGetters, f.goName(), strings.Join(args, ", "), f.ID)
}

const goHeader = `// Code generated by resources. DO NOT EDIT.
package %s

import (
	"errors"
	"fmt"
	"time"

	"go.mondoo.com/cnquery/resources"
	"github.com/rs/zerolog/log"%s
)

`

// GO METHODS FOR AST

func indent(s string, depth int) string {
	space := ""
	for i := 0; i < depth; i++ {
		space += "\t"
	}
	return space + strings.Replace(s, "\n", "\n"+space, -1)
}

func (r *Resource) structName(b *goBuilder) string {
	return "mql" + r.interfaceName(b)
}

var reMethodName = regexp.MustCompile("\\.[a-z]")

func capitalizeDot(in []byte) []byte {
	return bytes.ToUpper([]byte{in[1]})
}

func (r *Resource) interfaceName(b *goBuilder) string {
	return resource2goname(r.ID, b)
}

func resource2goname(s string, b *goBuilder) string {
	pack := strings.SplitN(s, ".", 2)
	var name string
	if pack[0] != s {
		resources, ok := b.ast.imports[pack[0]]
		if ok {
			if _, ok := resources[pack[1]]; !ok {
				b.errors = multierror.Append(b.errors,
					errors.New("cannot find resource "+pack[1]+" in imported resource pack "+pack[0]))
			}
			name = pack[0] + "." + strings.Title(string(
				reMethodName.ReplaceAllFunc([]byte(pack[1]), capitalizeDot),
			))
			b.packsInUse[pack[0]] = struct{}{}
		}
	}
	if name == "" {
		name = strings.Title(string(
			reMethodName.ReplaceAllFunc([]byte(s), capitalizeDot),
		))
	}

	return name
}

func (b *ResourceDef) staticFields() []*BasicField {
	res := []*BasicField{}
	for _, f := range b.Fields {
		if f.BasicField != nil {
			if f.BasicField.isStatic() {
				res = append(res, f.BasicField)
			}
		}
	}
	return res
}

func (f *BasicField) goName() string {
	return strings.Title(f.ID)
}

func (f *BasicField) isStatic() bool {
	return f.Args == nil
}

func (f *BasicField) methodname() string {
	return strings.Title(f.ID)
}

// Retrieve the raw mondoo equivalent type, which can be looked up
// as a resource.
func (t *Type) Type(ast *LR) types.Type {
	if t.SimpleType != nil {
		return t.SimpleType.typeItems(ast)
	}
	if t.ListType != nil {
		return t.ListType.typeItems(ast)
	}
	if t.MapType != nil {
		return t.MapType.typeItems(ast)
	}
	return types.Any
}

func (t *MapType) typeItems(ast *LR) types.Type {
	return types.Map(t.Key.typeItems(ast), t.Value.Type(ast))
}

func (t *ListType) typeItems(ast *LR) types.Type {
	return types.Array(t.Type.Type(ast))
}

func (t *SimpleType) typeItems(ast *LR) types.Type {
	switch t.Type {
	case "bool":
		return types.Bool
	case "int":
		return types.Int
	case "float":
		return types.Float
	case "string":
		return types.String
	case "regex":
		return types.Regex
	case "time":
		return types.Time
	case "dict":
		return types.Dict
	default:
		return resourceType(t.Type, ast)
	}
}

// Try to build an MQL resource from the given name. It may or may not exist in
// a pack. If it doesn't exist at all
func resourceType(name string, ast *LR) types.Type {
	pack := strings.SplitN(name, ".", 2)
	if pack[0] != name {
		resources, ok := ast.imports[pack[0]]
		if ok {
			if _, ok := resources[pack[1]]; ok {
				return types.Resource(pack[1])
			}
		}
	}

	// TODO: look up resources in the current registry and notify if they are not found

	return types.Resource(name)
}

// Retrieve the mondoo equivalent of the type. This is a stringified type
// i.e. it can be compiled with the MQL imports
func (t *Type) mondooType() string {
	i := t.mondooTypeItems()
	if i == "" {
		return "NO_TYPE_DETECTED"
	}
	return i
}

func (t *Type) mondooTypeItems() string {
	if t.SimpleType != nil {
		return t.SimpleType.mondooTypeItems()
	}
	if t.ListType != nil {
		return t.ListType.mondooTypeItems()
	}
	if t.MapType != nil {
		return t.MapType.mondooTypeItems()
	}
	return ""
}

func (t *MapType) mondooTypeItems() string {
	return "types.Map(" + t.Key.mondooTypeItems() + ", " + t.Value.mondooTypeItems() + ")"
}

func (t *ListType) mondooTypeItems() string {
	return "types.Array(" + t.Type.mondooTypeItems() + ")"
}

func (t *SimpleType) mondooTypeItems() string {
	switch t.Type {
	case "bool":
		return "types.Bool"
	case "int":
		return "types.Int"
	case "float":
		return "types.Float"
	case "string":
		return "types.String"
	case "regex":
		return "types.Regex"
	case "time":
		return "types.Time"
	case "dict":
		return "types.Dict"
	default:
		return "types.Resource(\"" + t.Type + "\")"
	}

	// TODO: check that this type if a proper resource
	// panic("Cannot convert type '" + t.Type + "' to mondoo type")
}

// The go type is the golang-equivalent code type, i.e. the type of the
// actual objects that are being moved around.
func (t *Type) goType(b *goBuilder) string {
	if t.SimpleType != nil {
		return t.SimpleType.goType(b)
	}
	if t.ListType != nil {
		return t.ListType.goType()
	}
	if t.MapType != nil {
		return t.MapType.goType(b)
	}
	return "NO_TYPE_DETECTED"
}

func (t *MapType) goType(b *goBuilder) string {
	// limited to interface{} because we cannot cast as universally
	// between types yet
	return "map[" + t.Key.goType(b) + "]interface{}"
}

func (t *ListType) goType() string {
	// limited to []interface{} because we cannot cast as universally
	// between types yet
	return "[]interface{}"
}

var primitiveTypes = map[string]string{
	"string": "string",
	"bool":   "bool",
	"int":    "int64",
	"float":  "float64",
	"time":   "*time.Time",
	"dict":   "interface{}",
	"any":    "interface{}",
}

func (t *SimpleType) goType(b *goBuilder) string {
	pt, ok := primitiveTypes[t.Type]
	if ok {
		return pt
	}

	// TODO: check if the resource exists
	return resource2goname(t.Type, b)
}

func (t *Type) goZeroValue() string {
	if t.SimpleType != nil {
		return t.SimpleType.goZeroValue()
	}
	return "nil"
}

var primitiveZeros = map[string]string{
	"string": "\"\"",
	"bool":   "false",
	"int":    "0",
	"float":  "0.0",
	"time":   "nil",
	"dict":   "nil",
	"any":    "nil",
}

func (t *SimpleType) goZeroValue() string {
	pt, ok := primitiveZeros[t.Type]
	if ok {
		return pt
	}

	// TODO: check if the resource exists
	return "nil"
}
