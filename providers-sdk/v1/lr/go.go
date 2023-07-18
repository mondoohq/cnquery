package lr

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
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

	o.goCreateResource(ast.Resources)
	o.goGetData(ast.Resources)
	o.goSetData(ast.Resources)

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

	header := fmt.Sprintf(goHeader, imports)
	return header + o.data, nil
}

const goHeader = `// Code generated by resources. DO NOT EDIT.
package resources

import (
	"errors"

	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/types"%s
)
`

func (b *goBuilder) goCreateResource(r []*Resource) {
	newCmds := make([]string, len(r))
	for i := range r {
		resource := r[i]
		newCmds[i] = "\"" + resource.ID + "\": New" + resource.interfaceName(b) + ","
	}

	b.data += `
var newResource map[string]func(runtime *plugin.Runtime, args map[string]interface{}) (plugin.Resource, error)

func init() {
	newResource = map[string]func(runtime *plugin.Runtime, args map[string]interface{}) (plugin.Resource, error) {
		` + strings.Join(newCmds, "\n\t\t") + `
	}
}

// CreateResource is used by the runtime of this plugin
func CreateResource(runtime *plugin.Runtime, name string, args map[string]interface{}) (plugin.Resource, error) {
	f, ok := newResource[name]
	if !ok {
		return nil, errors.New("cannot find resource " + name + " in os provider")
	}

	res, err := f(runtime, args)
	if err != nil {
		return nil, err
	}

	id := res.MqlID()
	if x, ok := runtime.Resources[name+"\x00"+id]; ok {
		res = x
	} else {
		runtime.Resources[name+"\x00"+id] = res
	}

	return res, nil
}
`
}

func (b *goBuilder) goGetData(r []*Resource) {
	fields := []string{}
	for i := range r {
		resource := r[i]
		for j := range resource.Body.Fields {
			field := resource.Body.Fields[j]
			if field.Init != nil {
				continue
			}

			x := fmt.Sprintf(`"%s.%s": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*%s).Get%s()).ToDataRes(%s)
	},`,
				resource.ID, field.BasicField.ID,
				resource.structName(b), field.BasicField.methodname(),
				field.BasicField.Type.mondooType(),
			)
			fields = append(fields, x)
		}
	}

	b.data += `
var getDataFields = map[string]func(r plugin.Resource) *plugin.DataRes{
	` + strings.Join(fields, "\n\t") + `
}

func GetData(resource plugin.Resource, field string, args map[string]interface{}) *plugin.DataRes {
	f, ok := getDataFields[resource.MqlName()+"."+field]
	if !ok {
		return &plugin.DataRes{Error: "cannot find '" + field + "' in resource '" + resource.MqlName() + "'"}
	}

	return f(resource)
}
`
}

func (b *goBuilder) goSetData(r []*Resource) {
	fields := []string{}
	for i := range r {
		resource := r[i]

		x := fmt.Sprintf(`"%s.%s": func(r plugin.Resource, v interface{}) bool {
			var ok bool
			r.(*%s).__id, ok = v.(string)
			return ok
		},`,
			resource.ID, "__id",
			resource.structName(b),
		)
		fields = append(fields, x)

		for j := range resource.Body.Fields {
			field := resource.Body.Fields[j]
			if field.Init != nil {
				continue
			}

			x := fmt.Sprintf(`"%s.%s": func(r plugin.Resource, v interface{}) bool {
		var ok bool
		r.(*%s).%s, ok = plugin.RawToTValue[%s](v)
		return ok
	},`,
				resource.ID, field.BasicField.ID,
				resource.structName(b), field.BasicField.methodname(),
				field.BasicField.Type.goType(b),
			)
			fields = append(fields, x)
		}
	}

	b.data += `
var setDataFields = map[string]func(r plugin.Resource, v interface{}) bool {
	` + strings.Join(fields, "\n\t") + `
}

func SetData(resource plugin.Resource, field string, val interface{}) error {
	f, ok := setDataFields[resource.MqlName() + "." + field]
	if !ok {
		return errors.New("cannot set '"+field+"' in resource '"+resource.MqlName()+"', field not found")
	}

	if ok := f(resource, val); !ok {
		return errors.New("cannot set '"+field+"' in resource '"+resource.MqlName()+"', type does not match")
	}
	return nil
}
`
}

func (b *goBuilder) goResource(r *Resource) error {
	b.goStruct(r)
	b.goFactory(r)
	b.goFields(r)
	return nil
}

func (b *goBuilder) goStruct(r *Resource) {
	internalStruct := r.structName(b) + "Internal"
	if !b.collector.HasStruct(internalStruct) {
		internalStruct = "// optional: if you define " + internalStruct + " it will be used here"
	}

	fields := []string{}
	for i := range r.Body.Fields {
		field := r.Body.Fields[i]
		if field.Init != nil {
			continue
		}
		fields = append(fields, field.BasicField.goName()+" plugin.TValue["+field.BasicField.Type.goType(b)+"]")
	}

	b.data += fmt.Sprintf(`
// %s for the %s resource
type %s struct {
	MqlRuntime *plugin.Runtime
	__id string
	%s

	%s
}
`,
		r.structName(b), r.ID, r.structName(b),
		internalStruct,
		strings.Join(fields, "\n\t"),
	)
}

func (b *goBuilder) goFactory(r *Resource) {
	newName := "New" + r.interfaceName(b)
	structName := r.structName(b)

	var initCode string
	if b.collector.HasInit(structName) {
		initCode = `var err error
	var existing *` + structName + `
	args, existing, err = res.init(args)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}`
	} else {
		initCode = `var err error
	// to override args, implement: init(args map[string]interface{}) (map[string]interface{}, *` + structName + `, error)`
	}

	var idCode string
	if b.collector.HasID(structName) {
		idCode = "res.__id, err = res.id()"
	} else {
		idCode = `// to override __id implement: id() (string, error)`
	}

	b.data += fmt.Sprintf(`
// %s creates a new instance of this resource
func %s(runtime *plugin.Runtime, args map[string]interface{}) (plugin.Resource, error) {
	res := &%s{
		MqlRuntime: runtime,
	}

	%s

	for k, v := range args {
		if err = SetData(res, k, v); err != nil {
			return res, err
		}
	}

	%s
	return res, err
}

func (c *%s) MqlName() string {
	return "%s"
}

func (c *%s) MqlID() string {
	return c.__id
}
`,
		newName, newName, structName,
		initCode,
		idCode,
		structName, r.ID,
		structName,
	)
}

func (b *goBuilder) goFields(r *Resource) {
	for i := range r.Body.Fields {
		field := r.Body.Fields[i]
		if field.Init != nil {
			continue
		}

		b.goField(r, field)
	}
}

func (b *goBuilder) goStaticField(r *Resource, field *Field) {
	goName := field.BasicField.goName()
	b.data += fmt.Sprintf(`
func (c *%s) Get%s() *plugin.TValue[%s] {
	return &c.%s
}
`,
		r.structName(b), goName, field.BasicField.Type.goType(b),
		goName,
	)
}

func (b *goBuilder) goField(r *Resource, field *Field) {
	if field.BasicField.isStatic() {
		b.goStaticField(r, field)
		return
	}

	goName := field.BasicField.goName()
	goType := field.BasicField.Type.goType(b)
	goZero := field.BasicField.Type.goZeroValue()

	argDefs := []string{}
	argCall := []string{}
	if field.BasicField.Args != nil {
		args := field.BasicField.Args.List
		for i := range args {
			arg := args[i]
			name := resource2goname(arg.Type, b)
			argDefs = append(argDefs, fmt.Sprintf(`varg%s := c.Get%s()
		if varg%s.Error != nil {
			return %s, varg%s.Error
		}
		`, name, name, name, goZero, name))
			argCall = append(argCall, "varg"+name+".Data")
		}
	}

	b.data += fmt.Sprintf(`
func (c *%s) Get%s() *plugin.TValue[%s] {
	return plugin.GetOrCompute[%s](&c.%s, func() (%s, error) {
		%sreturn c.%s(%s)
	})
}
`,
		r.structName(b), goName, goType,
		goType, goName, goType,
		strings.Join(argDefs, "\n\t\t"),
		field.BasicField.ID, strings.Join(argCall, ", "),
	)
}

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

	return "*mql" + resource2goname(t.Type, b)
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
