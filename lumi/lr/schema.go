package lr

import (
	"errors"

	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
)

func Schema(ast *LR, collector *Collector) (*lumi.Schema, error) {
	res := &lumi.Schema{
		Resources: map[string]*lumi.ResourceInfo{},
	}

	for i := range ast.Resources {
		x, err := resourceSchema(ast.Resources[i], collector)
		if err != nil {
			return res, err
		}

		res.Resources[x.Id] = x
	}

	return res, nil
}

func resourceInit(r *Resource, fields map[string]*lumi.Field) (*lumi.Init, error) {
	if len(r.Body.Inits) == 0 {
		return nil, nil
	}

	args := []*lumi.TypedArg{}
	i := r.Body.Inits[0]
	isOptional := false
	for _, arg := range i.Args {
		typ := arg.Type.Type()
		if typ == types.Unset {
			return nil, errors.New("A field in the init that isnt found in the resource must have a type assigned. Field \"" + arg.ID + "\"")
		}

		ref, ok := fields[arg.ID]
		if ok {
			ftype := ref.Type
			if string(typ) != ftype {
				return nil, errors.New("Init field type and resource field type are different: " + r.ID + " field " + arg.ID)
			}
		}

		if arg.Optional {
			isOptional = true
		} else if isOptional {
			return nil, errors.New("A required argument cannot follow an optional argument. Found in init function of " + r.ID)
		}

		args = append(args, &lumi.TypedArg{
			Name:     arg.ID,
			Type:     string(typ),
			Optional: arg.Optional,
		})
	}

	return &lumi.Init{Args: args}, nil
}

func resourceFields(r *Resource) map[string]*lumi.Field {
	fields := make(map[string]*lumi.Field)

	for _, f := range r.Body.Fields {
		refs := []string{}

		if f.Args != nil && len(f.Args.List) > 0 {
			for _, arg := range f.Args.List {
				refs = append(refs, "\""+arg.Type+"\"")
			}
		}

		fields[f.ID] = &lumi.Field{
			Name:        f.ID,
			Type:        string(f.Type.Type()),
			IsMandatory: f.isStatic(),
			Title:       r.title,
			Desc:        r.desc,
			Refs:        refs,
		}
	}

	return fields
}

func resourceSchema(r *Resource, collector *Collector) (*lumi.ResourceInfo, error) {
	fields := resourceFields(r)
	init, err := resourceInit(r, fields)
	if err != nil {
		return nil, err
	}

	res := &lumi.ResourceInfo{
		Id:      r.ID,
		Name:    r.ID,
		Title:   r.title,
		Desc:    r.desc,
		Init:    init,
		Private: r.IsPrivate,
		Fields:  fields,
	}

	if r.ListType != nil {
		res.ListType = string(types.Resource(r.ListType.Type.Type))
	}

	return res, nil
}
