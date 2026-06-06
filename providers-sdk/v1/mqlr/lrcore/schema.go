// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package lrcore

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
	"go.mondoo.com/mql/v13/types"
)

func Schema(ast *LR) (*resources.Schema, error) {
	provider, ok := ast.Options["provider"]
	if !ok {
		return nil, errors.New("missing provider name for resources to generate schema")
	}

	res := &resources.Schema{
		Resources:    make(map[string]*resources.ResourceInfo, len(ast.Resources)),
		Dependencies: make(map[string]*resources.ProviderInfo, 0),
	}

	for dep := range ast.imports {
		if !strings.HasSuffix(provider, dep) && dep != "core" {
			res.Dependencies[dep] = &resources.ProviderInfo{
				Id:   strings.TrimSuffix(ast.packPaths[dep], "/resources"),
				Name: dep,
			}
		}
	}

	var schemaErrs []error
	for i := range ast.Resources {
		x, err := resourceSchema(ast.Resources[i], ast)
		if err != nil {
			schemaErrs = append(schemaErrs, err)
			if x == nil {
				continue
			}
		}

		res.Resources[x.Id] = x
	}

	// Aliases share the same *ResourceInfo under a second key. This is how the
	// invariant in resources.proto (Schema.resources) gets established: if a
	// map key in res.Resources differs from its value's `Id`, the entry is an
	// alias from the key to the resource named by `Id`. We deliberately reuse
	// the pointer rather than deep-copy so the two entries stay in lockstep.
	for defName, r := range ast.aliases {
		x, ok := res.Resources[r.ID]
		if !ok {
			var err error
			x, err = resourceSchema(r, ast)
			if err != nil {
				schemaErrs = append(schemaErrs, err)
				if x == nil {
					continue
				}
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

	sorted := slices.SortedFunc(maps.Keys(res.Resources), func(a, b string) int {
		aDepth := strings.Count(a, ".")
		bDepth := strings.Count(b, ".")
		return aDepth - bDepth
	})

	// In this block we finalize the schema. This means:
	// 1: create implicit resources (eg: sshd.config => create sshd)
	// 2: create implicit fields (eg: sshd.config => sshd { config: {..} })
	for _, name := range sorted {
		v := res.Resources[name]
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
					Maturity:           v.Maturity,
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

	// Validate all maturity values
	for name, ri := range res.Resources {
		if err := resources.ValidateMaturity(ri.Maturity); err != nil {
			return nil, fmt.Errorf("resource %s: %w", name, err)
		}
		for fname, f := range ri.Fields {
			if err := resources.ValidateMaturity(f.Maturity); err != nil {
				return nil, fmt.Errorf("resource %s field %s: %w", name, fname, err)
			}
		}
	}

	if len(schemaErrs) > 0 {
		return res, errors.Join(schemaErrs...)
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

	var validationErrs []error
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
		f.Comments = lastCommentGroup(f.Comments)
		if verr := validateDocCommentStructure(f.Comments, "field "+r.ID+"."+f.BasicField.ID); verr != nil {
			validationErrs = append(validationErrs, verr)
		}
		title, desc := extractTitleAndDescription(f.Comments)
		fields[f.BasicField.ID] = &resources.Field{
			Name:        f.BasicField.ID,
			Type:        string(f.BasicField.Type.Type(ast)),
			IsMandatory: f.BasicField.isStatic(),
			Title:       title,
			Desc:        desc,
			Refs:        refs,
			IsEmbedded:  f.BasicField.isEmbedded,
			Maturity:    f.BasicField.Maturity,
		}
	}

	if len(validationErrs) > 0 {
		return fields, errors.Join(validationErrs...)
	}
	return fields, nil
}

func resourceSchema(r *Resource, ast *LR) (*resources.ResourceInfo, error) {
	// Keep going even if fields had validation errors so we can collect every
	// violation in one pass and the caller can report them all at once.
	fields, fieldsErr := resourceFields(r, ast)

	init, err := resourceInit(r, fields, ast)
	if err != nil {
		return nil, errors.Join(fieldsErr, err)
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
		Maturity:    r.Maturity,
	}

	if r.ListType != nil {
		res.ListType = string(r.ListType.Type.typeItems(ast))
	}

	return res, fieldsErr
}
