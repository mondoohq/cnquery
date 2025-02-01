// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package lr

import (
	"errors"
	"strings"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v11/types"
)

var CONTEXT_FIELD = "context"

func Schema(ast *LR) (*resources.Schema, error) {
	provider, ok := ast.Options["provider"]
	if !ok {
		return nil, errors.New("missing provider name for resources to generate schema")
	}

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

	for defName, r := range ast.aliases {
		x, ok := res.Resources[r.ID]
		if !ok {
			var err error
			x, err = resourceSchema(r, ast)
			if err != nil {
				return res, err
			}
		}
		res.Resources[defName] = x
	}

	// make sure every resource and field has the provider set
	for _, v := range res.Resources {
		v.Provider = provider
		for _, field := range v.Fields {
			field.Provider = provider
		}
	}

	// In this block we finalize the schema. This means:
	// 1: create implicit resources (eg: sshd.config => create sshd)
	// 2: create implicit fields (eg: sshd.config => sshd { config: {..} })
	for name, v := range res.Resources {
		if !strings.Contains(name, ".") {
			continue
		}

		rem := name
		fieldInfo := v
		isPrivate := v.Private
		for {
			last := strings.LastIndex(rem, ".")
			if last == -1 {
				break
			}

			resource := rem
			basename := rem[last+1:]
			rem = rem[:last]

			child, ok := res.Resources[rem]
			if !ok {
				child = &resources.ResourceInfo{
					Id:          rem,
					Fields:      map[string]*resources.Field{},
					IsExtension: true,
					// Resource extensions do not set the provider. They are here to
					// indicate that it bridges the resource chain, but it cannot
					// initialize this resource! This is why no provider is set.
				}
				res.Resources[rem] = child
			}

			if _, ok := child.Fields[basename]; !ok {
				child.Fields[basename] = &resources.Field{
					Name:               basename,
					Type:               string(types.Resource(resource)),
					IsMandatory:        false, // it cannot be mandatory if we create it here
					IsImplicitResource: true,
					IsPrivate:          isPrivate,
					Title:              fieldInfo.Title,
					Desc:               fieldInfo.Desc,
					Provider:           provider,
				}
			}

			// Some of the call-chain might have been created by other resources.
			// If this resource, however, is not private, then it must be accessible
			// through the callchain.
			if !isPrivate {
				child.Fields[basename].IsPrivate = false
			}

			fieldInfo = child
		}
	}

	return res, nil
}

func resourceInit(r *Resource, fields map[string]*resources.Field, ast *LR) (*resources.Init, error) {
	inits := r.GetInitFields()
	if len(inits) == 0 {
		return nil, nil
	}

	args := []*resources.TypedArg{}
	i := inits[0]
	isOptional := false
	for _, arg := range i.Args {
		typ := arg.Type.Type(ast)
		if typ == types.Unset {
			return nil, errors.New("A field in the init that isn't found in the resource must have a type assigned. Field \"" + arg.ID + "\"")
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

func resourceFields(r *Resource, ast *LR) (map[string]*resources.Field, error) {
	fields := make(map[string]*resources.Field)

	for _, f := range r.Body.Fields {
		if f.BasicField == nil {
			continue
		}
		refs := []string{}

		if f.BasicField.Args != nil && len(f.BasicField.Args.List) > 0 {
			for _, arg := range f.BasicField.Args.List {
				refs = append(refs, "\""+arg.Type+"\"")
			}
		}

		f.Comments = SanitizeComments(f.Comments)
		title, desc := extractTitleAndDescription(f.Comments)
		fields[f.BasicField.ID] = &resources.Field{
			Name:        f.BasicField.ID,
			Type:        string(f.BasicField.Type.Type(ast)),
			IsMandatory: f.BasicField.isStatic(),
			Title:       title,
			Desc:        desc,
			Refs:        refs,
			IsEmbedded:  f.BasicField.isEmbedded,
		}
	}

	if r.Context != "" {
		if _, ok := fields[CONTEXT_FIELD]; ok {
			return nil, errors.New("'" + CONTEXT_FIELD + "' field already exists on resource " + r.ID)
		}
		fields[CONTEXT_FIELD] = &resources.Field{
			Name:        CONTEXT_FIELD,
			Type:        r.Context,
			IsMandatory: true,
			Title:       "Context",
			Desc:        "Contextual info, where this resource is located and defined",
			IsEmbedded:  false,
		}
	}

	return fields, nil
}

func resourceSchema(r *Resource, ast *LR) (*resources.ResourceInfo, error) {
	fields, err := resourceFields(r, ast)
	if err != nil {
		return nil, err
	}

	init, err := resourceInit(r, fields, ast)
	if err != nil {
		return nil, err
	}

	if init != nil && r.IsExtension {
		return nil, errors.New("Resource '" + r.ID + "' as an init method AND is flagged as 'extends'. You cannot do both at the same time. Either this resource extends another or it is the root resource that gets extended.")
	}

	res := &resources.ResourceInfo{
		Id:          r.ID,
		Name:        r.ID,
		Title:       r.title,
		Desc:        r.desc,
		Init:        init,
		Private:     r.IsPrivate,
		IsExtension: r.IsExtension,
		Fields:      fields,
		Defaults:    r.Defaults,
		Context:     r.Context,
	}

	if r.ListType != nil {
		res.ListType = string(r.ListType.Type.typeItems(ast))
	}

	return res, nil
}
