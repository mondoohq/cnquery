package lr

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

type goBuilder struct {
	data      string
	collector *Collector
}

// Go produced go code for the LR file
func Go(ast *LR, collector *Collector) (string, error) {
	o := goBuilder{collector: collector}
	o.data += goHeader
	o.goRegistryInit(ast.Resources)

	for i := range ast.Resources {
		err := o.goResource(ast.Resources[i])
		if err != nil {
			return o.data, err
		}
	}

	return o.data, nil
}

func (b *goBuilder) goRegistryInit(r []*Resource) {
	arr := []string{}
	for i := range r {
		arr = append(arr, "registry.Add((&"+r[i].structName()+"{}).initInfo())")
	}
	res := strings.Join(arr, "\n")

	b.data += `// Init all resources into the registry
func Init(registry *lumi.Registry) {
` + indent(res, 1) + `
}

`
}

func (b *goBuilder) goResource(r *Resource) error {
	if r.ListType != nil {
		t := r.ListType.Type.Type
		r.Body.Fields = append(r.Body.Fields,
			&Field{
				ID:   "list",
				Args: &FieldArgs{},
				Type: Type{ListType: &ListType{Type: Type{SimpleType: &SimpleType{t}}}},
			},
		)
	}

	b.goInterface(r)
	b.goStruct(r)
	b.goFactory(r)
	if err := b.goInitInfo(r); err != nil {
		return err
	}
	b.goRegister(r)
	b.goField(r)
	b.goCompute(r)
	return nil
}

func (b *goBuilder) goInterface(r *Resource) {
	fields := ""
	for _, f := range r.Body.Fields {
		fields += "\t" + f.methodname() + "() (" + f.Type.goType() + ", error)\n"
	}

	b.data += fmt.Sprintf(`// %s resource interface
type %s interface {
	LumiResource() (*lumi.Resource)
	Compute(string) error
	Field(string) (interface{}, error)
	Register(string) error
	Validate() error
%s}

`, r.interfaceName(), r.interfaceName(), fields)
}

func (b *goBuilder) goStruct(r *Resource) {
	b.data += fmt.Sprintf(`// %s for the %s resource
type %s struct {
	*lumi.Resource
}

`, r.structName(), r.ID, r.structName())

	b.data += fmt.Sprintf(`// LumiResource to retrieve the underlying resource info
func (s *%s) LumiResource() *lumi.Resource {
	return s.Resource
}

`, r.structName())
}

func (b *goBuilder) goFactory(r *Resource) {
	args := ""
	for _, f := range r.Body.Fields {
		args += fmt.Sprintf(`		case "%s":
			if _, ok := val.(%s); !ok {
				return nil, errors.New("Failed to initialize \"%s\", its \"%s\" argument has the wrong type (expected type \"%s\")")
			}
			break
`, f.ID, f.Type.goType(), r.ID, f.ID, f.Type.goType())
	}

	required := ""
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

	hasInit := b.collector.HasInit(r.structName())
	log.Debug().Bool("init", hasInit).Msg("dynamic calls for " + r.interfaceName())

	initcall := ""
	if b.collector.HasInit(r.structName()) {
		initcall = `var existing ` + r.interfaceName() + `
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
func new` + r.interfaceName() + `(runtime *lumi.Runtime, args *lumi.Args) (interface{}, error) {
	// User hooks
	var err error
	res := ` + r.structName() + `{runtime.NewResource("` + r.ID + `")}
	` + initcall + `// assign all named fields
	for name, val := range *args {
		switch name {
` + args + `		default:
			return nil, errors.New("Initialized ` + r.ID + ` with unknown argument " + name)
		}
		res.Cache.Store(name, &lumi.CacheEntry{Data: val, Valid: true, Timestamp: time.Now().Unix()})
	}

	// Get the ID
	res.Resource.Id, err = res.id()
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *` + r.structName() + `) Validate() error {
	// required arguments
` + required + `
	return nil
}

`
}

func (b *goBuilder) goInitInfo(r *Resource) error {
	fields := ""
	fieldsMap := make(map[string]*Field)
	for _, f := range r.Body.Fields {
		fieldsMap[f.ID] = f
		refs := "nil"
		if f.Args != nil && len(f.Args.List) > 0 {
			arglist := []string{}
			for _, arg := range f.Args.List {
				arglist = append(arglist, "\""+arg.Type+"\"")
			}
			refs = "[]string{" + strings.Join(arglist, ", ") + "}"
		}
		fields += fmt.Sprintf(
			`	fields["%s"] = &lumi.Field{Name: "%s", Type: string(%s), Mandatory: %t, Refs: %s}
`, f.ID, f.ID, f.Type.mondooType(), f.isStatic(), refs)
	}

	if len(r.Body.Inits) > 1 {
		return errors.New("Resource defined more than one init function: " + r.ID)
	}

	init := "nil"
	if len(r.Body.Inits) == 1 {
		args := []string{}
		i := r.Body.Inits[0]
		for _, arg := range i.Args {
			typ := arg.Type.mondooType()
			if typ == "NO_TYPE_DETECTED" {
				return errors.New("A field in the init that isnt found in the resource must have a type assigned. FIeld \"" + arg.ID + "\"")
			}

			ref, ok := fieldsMap[arg.ID]
			if ok {
				ftype := ref.Type.mondooType()
				if typ != ftype {
					return errors.New("Init field type and resource field type are different: " + r.ID + " field " + arg.ID)
				}
			}

			args = append(args, `				&lumi.TypedArg{Name: "`+arg.ID+`", Type: string(`+typ+`)},
`)
		}
		init = `&lumi.Init{Args: []*lumi.TypedArg{
` + strings.Join(args, "\n") + `}}`
	}

	listType := "\"\""
	if r.ListType != nil {
		listType = `string(types.Resource("` + r.ListType.Type.Type + `"))`
	}

	b.data += `// initInfo contains all information needed for the resource registration
func (s *` + r.structName() + `) initInfo() *lumi.ResourceCls {
	fields := make(map[string]*lumi.Field)
` + fields + `
	info := lumi.ResourceInfo{
		Name: "` + r.ID + `",
		Init: ` + init + `,
		Fields: fields,
		ListType: ` + listType + `,
	}
	return &lumi.ResourceCls{
		Factory:      new` + r.interfaceName() + `,
		ResourceInfo: info,
	}
}

`
	return nil
}

func goRegisterField(f *Field) string {
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
			`		if err = s.Runtime.WatchAndCompute(s, "%s", s, "%s"); err != nil {
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
	for i := range r.Body.Fields {
		fields = append(fields, goRegisterField(r.Body.Fields[i]))
	}

	b.data += fmt.Sprintf(`// Register accessor autogenerated
func (s *%s) Register(name string) error {
	log.Debug().Str("field", name).Msg("[%s].Register")
	switch name {
%s	default:
		return errors.New("Cannot find field '" + name + "' in \"%s\" resource")
	}
}

`, r.structName(), r.ID, strings.Join(fields, ""), r.ID)
}

func (b *goBuilder) goField(r *Resource) {
	caseField := []string{}
	for _, f := range r.Body.Fields {
		caseField = append(caseField, fmt.Sprintf(`	case "%s":
		return s.%s()
`, f.ID, f.goName()))
	}
	b.data += fmt.Sprintf(`// Field accessor autogenerated
func (s *%s) Field(name string) (interface{}, error) {
	log.Debug().Str("field", name).Msg("[%s].Field")
	switch name {
%s	default:
		return nil, fmt.Errorf("Cannot find field '" + name + "' in \"%s\" resource")
	}
}

`, r.structName(), r.ID, strings.Join(caseField, ""), r.ID)

	for i := range r.Body.Fields {
		f := r.Body.Fields[i]
		b.goFieldAccessor(r, f)
	}
}

func (b *goBuilder) goCompute(r *Resource) {
	caseField := []string{}

	for _, f := range r.Body.Fields {
		// static fields don't have a compute call associated with them
		if f.isStatic() {
			caseField = append(caseField, fmt.Sprintf(`	case "%s":
		return nil
`, f.ID))
			continue
		}

		caseField = append(caseField, fmt.Sprintf(`	case "%s":
		return s.Compute%s()
`, f.ID, f.goName()))
	}

	b.data += fmt.Sprintf(`// Compute accessor autogenerated
func (s *%s) Compute(name string) error {
	log.Debug().Str("field", name).Msg("[%s].Compute")
	switch name {
%s	default:
		return errors.New("Cannot find field '" + name + "' in \"%s\" resource")
	}
}

`, r.structName(), r.ID, strings.Join(caseField, ""), r.ID)

	for i := range r.Body.Fields {
		f := r.Body.Fields[i]
		b.goFieldComputer(r, f)
	}
}

func (b *goBuilder) goFieldAccessor(r *Resource, f *Field) {
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
		s.Runtime.Trigger(s, "%s")`,
			f.goName(), f.Type.goZeroValue(),
			f.ID,
			f.Type.goZeroValue(), r.ID, f.ID,
			f.ID,
		)
	} else {
		notFound = "return " + f.Type.goZeroValue() + ", lumi.NotReadyError{}"
	}

	b.data += fmt.Sprintf(`// %s accessor autogenerated
func (s *%s) %s() (%s, error) {
	res, ok := s.Cache.Load("%s")
	if !ok || !res.Valid {
		%s
	}
	tres, ok := res.Data.(%s)
	if !ok {
		return %s, fmt.Errorf("\"%s\" failed to cast field \"%s\" to the right type (%s): %%#v", res)
	}
	return tres, nil
}

`, f.goName(),
		r.structName(), f.goName(), f.Type.goType(),
		f.ID, notFound,
		f.Type.goType(),
		f.Type.goZeroValue(), r.ID, f.ID, f.Type.goType())
}

func (b *goBuilder) goFieldComputer(r *Resource, f *Field) {
	if f.Args == nil {
		return
	}

	argGetters := ""
	args := make([]string, len(f.Args.List))

	for i, arg := range f.Args.List {
		args[i] = "varg" + arg.goType()
		argGetters += fmt.Sprintf(`	varg%s, err := s.%s()
	if err != nil {
		return err
	}
`, arg.goType(), arg.goType())
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
	if err != nil {
		return err
	}
	s.Cache.Store("%s", &lumi.CacheEntry{Data: vres, Valid: true, Timestamp: time.Now().Unix()})
	return nil
}

`, f.goName(), r.structName(), f.goName(), argGetters, f.goName(), strings.Join(args, ", "), f.ID)
}

const goHeader = `// Code generated by lumi. DO NOT EDIT.
package resources

import (
	"errors"
	"fmt"
	"time"

	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
	"github.com/rs/zerolog/log"
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

func (r *Resource) structName() string {
	return "lumi" + r.interfaceName()
}

var reMethodName = regexp.MustCompile("\\.[a-z]")

func capitalizeDot(in []byte) []byte {
	return bytes.ToUpper([]byte{in[1]})
}

func (r *Resource) interfaceName() string {
	return resource2goname(r.ID)
}

func resource2goname(s string) string {
	cleaned := reMethodName.ReplaceAllFunc([]byte(s), capitalizeDot)
	return strings.Title(string(cleaned))
}

func (b *ResourceDef) staticFields() []*Field {
	res := []*Field{}
	for i := range b.Fields {
		f := b.Fields[i]
		if f.isStatic() {
			res = append(res, f)
		}
	}
	return res
}

func (f *Field) goName() string {
	return strings.Title(f.ID)
}

func (f *Field) isStatic() bool {
	return f.Args == nil
}

func (f *Field) methodname() string {
	return strings.Title(f.ID)
}

// retrieve the mondoo equivalent of the type
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
	default:
		return "types.Resource(\"" + t.Type + "\")"
	}

	// TODO: check that this type if a proper resource
	// panic("Cannot convert type '" + t.Type + "' to mondoo type")
}

func (t *Type) goType() string {
	if t.SimpleType != nil {
		return t.SimpleType.goType()
	}
	if t.ListType != nil {
		return t.ListType.goType()
	}
	if t.MapType != nil {
		return t.MapType.goType()
	}
	return "NO_TYPE_DETECTED"
}

func (t *MapType) goType() string {
	// limited to interface{} because we cannot cast as universally
	// between types yet
	return "map[" + t.Key.goType() + "]interface{}"
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
	"any":    "interface{}",
}

func (t *SimpleType) goType() string {
	pt, ok := primitiveTypes[t.Type]
	if ok {
		return pt
	}

	// TODO: check if the resource exists
	return resource2goname(t.Type)
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
