package lr

import (
	"errors"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/types"
)

func Schema(ast *LR) (*resources.Schema, error) {
	res := &resources.Schema{
		Resources: make(map[string]*resources.ResourceInfo, len(ast.Resources)),
	}

	for i := range ast.Resources {
		x, err := resourceSchema(ast.Resources[i], ast)
		if err != nil {
			return res, err
		}

		res.Resources[x.Id] = x
	}

	return res, nil
}

func resourceInit(r *Resource, fields map[string]*resources.Field, ast *LR) (*resources.Init, error) {
	if len(r.Body.Inits) == 0 {
		return nil, nil
	}

	args := []*resources.TypedArg{}
	i := r.Body.Inits[0]
	isOptional := false
	for _, arg := range i.Args {
		typ := arg.Type.Type(ast)
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

		args = append(args, &resources.TypedArg{
			Name:     arg.ID,
			Type:     string(typ),
			Optional: arg.Optional,
		})
	}

	return &resources.Init{Args: args}, nil
}

func resourceFields(r *Resource, ast *LR) map[string]*resources.Field {
	fields := make(map[string]*resources.Field)

	for _, f := range r.Body.Fields {
		refs := []string{}

		if f.Args != nil && len(f.Args.List) > 0 {
			for _, arg := range f.Args.List {
				refs = append(refs, "\""+arg.Type+"\"")
			}
		}

		fields[f.ID] = &resources.Field{
			Name:        f.ID,
			Type:        string(f.Type.Type(ast)),
			IsMandatory: f.isStatic(),
			Title:       r.title,
			Desc:        r.desc,
			Refs:        refs,
		}
	}

	return fields
}

func resourceSchema(r *Resource, ast *LR) (*resources.ResourceInfo, error) {
	fields := resourceFields(r, ast)
	init, err := resourceInit(r, fields, ast)
	if err != nil {
		return nil, err
	}

	res := &resources.ResourceInfo{
		Id:      r.ID,
		Name:    r.ID,
		Title:   r.title,
		Desc:    r.desc,
		Init:    init,
		Private: r.IsPrivate,
		Fields:  fields,
	}

	if r.ListType != nil {
		res.ListType = string(r.ListType.Type.typeItems(ast))
	}

	return res, nil
}
