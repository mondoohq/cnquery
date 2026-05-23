// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/aws-cloudformation/rain/cft"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/cloudformation/connection"
	"go.mondoo.com/mql/v13/types"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	"gopkg.in/yaml.v3"
)

func initCloudformationTemplate(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.CloudformationConnection)
	template := conn.CftTemplate()

	args["version"] = llx.StringData("")
	args["description"] = llx.StringData("")
	args["transform"] = llx.NilData

	// cft.Template.GetSection dereferences Node.Content[0] without a guard, so
	// short-circuit on a degenerate template (empty file, comments only).
	if template.Node == nil || len(template.Node.Content) == 0 {
		return args, nil, nil
	}

	version, err := template.GetSection(cft.AWSTemplateFormatVersion)
	if err == nil {
		args["version"] = llx.StringData(version.Value)
	}

	desc, err := template.GetSection(cft.Description)
	if err == nil {
		args["description"] = llx.StringData(desc.Value)
	}

	transform, err := template.GetSection(cft.Transform)
	if err == nil && len(transform.Content) > 0 {
		var entries []string
		for _, entry := range transform.Content {
			entries = append(entries, entry.Value)
		}
		args["transform"] = llx.ArrayData(convert.SliceAnyToInterface(entries), types.String)
	}

	return args, nil, nil
}

func (r *mqlCloudformationTemplate) id() (string, error) {
	return "cloudformation", nil
}

func (r *mqlCloudformationTemplate) extractDict(section cft.Section) (map[string]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.CloudformationConnection)
	template := conn.CftTemplate()

	if template.Node == nil || len(template.Node.Content) == 0 {
		return nil, nil
	}

	_, parameters, err := gatherMapValue(template.Node.Content[0], string(section))
	if err != nil && status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	result := make(map[string]any)
	for i := 0; i < len(parameters.Content); i += 2 {
		keyNode := parameters.Content[i]
		valueNode := parameters.Content[i+1]

		dict, err := convertYamlToDict(valueNode)
		if err != nil {
			return nil, err
		}

		result[keyNode.Value] = dict
	}

	return result, nil
}

func (r *mqlCloudformationTemplate) mappings() (map[string]any, error) {
	return r.extractDict(cft.Mappings)
}

var Globals cft.Section = "Globals"

// Reads the Globals section of the SAM template.
// see https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/sam-specification-template-anatomy.html
func (r *mqlCloudformationTemplate) globals() (map[string]any, error) {
	return r.extractDict(Globals)
}

func (r *mqlCloudformationTemplate) parameters() (map[string]any, error) {
	return r.extractDict(cft.Parameters)
}

func (r *mqlCloudformationTemplate) metadata() (map[string]any, error) {
	return r.extractDict(cft.Metadata)
}

func (r *mqlCloudformationTemplate) conditions() (map[string]any, error) {
	return r.extractDict(cft.Conditions)
}

func (r *mqlCloudformationTemplate) rules() (map[string]any, error) {
	return r.extractDict(cft.Rules)
}

func (x *mqlCloudformationResource) id() (string, error) {
	return x.Name.Data, nil
}

func (r *mqlCloudformationTemplate) resources() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.CloudformationConnection)
	template := conn.CftTemplate()
	if template.Node == nil || len(template.Node.Content) == 0 {
		return nil, nil
	}
	_, resources, err := gatherMapValue(template.Node.Content[0], string(cft.Resources))
	if err != nil && status.Code(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	result := make([]any, 0)
	for i := 0; i < len(resources.Content); i += 2 {
		keyNode := resources.Content[i]
		valueNode := resources.Content[i+1]

		resourceType := ""
		resourceCondition := ""
		resourceDocumentation := ""

		_, val, err := gatherMapValue(valueNode, "Type")
		if err == nil {
			resourceType = val.Value
		}
		_, val, err = gatherMapValue(valueNode, "Condition")
		if err == nil {
			resourceCondition = val.Value
		}
		_, val, err = gatherMapValue(valueNode, "Documentation")
		if err == nil {
			resourceDocumentation = val.Value
		}

		attrs := make(map[string](any))
		_, val, err = gatherMapValue(valueNode, "Attributes")
		if err == nil {
			attrs, err = convertYamlToDict(val)
			if err != nil {
				return nil, err
			}
		}

		props := make(map[string](any))
		_, val, err = gatherMapValue(valueNode, "Properties")
		if err == nil {
			props, err = convertYamlToDict(val)
			if err != nil {
				return nil, err
			}
		}

		var dependsOn []any
		_, val, err = gatherMapValue(valueNode, "DependsOn")
		if err == nil {
			switch val.Kind {
			case yaml.ScalarNode:
				dependsOn = []any{val.Value}
			case yaml.SequenceNode:
				for _, item := range val.Content {
					dependsOn = append(dependsOn, item.Value)
				}
			}
		}

		deletionPolicy := ""
		_, val, err = gatherMapValue(valueNode, "DeletionPolicy")
		if err == nil {
			deletionPolicy = val.Value
		}

		updateReplacePolicy := ""
		_, val, err = gatherMapValue(valueNode, "UpdateReplacePolicy")
		if err == nil {
			updateReplacePolicy = val.Value
		}

		creationPolicy, err := nodeToDict(valueNode, "CreationPolicy")
		if err != nil {
			return nil, err
		}
		updatePolicy, err := nodeToDict(valueNode, "UpdatePolicy")
		if err != nil {
			return nil, err
		}
		resourceMetadata, err := nodeToDict(valueNode, "Metadata")
		if err != nil {
			return nil, err
		}

		pkg, err := CreateResource(r.MqlRuntime, "cloudformation.resource", map[string]*llx.RawData{
			"name":                llx.StringData(keyNode.Value),
			"type":                llx.StringData(resourceType),
			"condition":           llx.StringData(resourceCondition),
			"documentation":       llx.StringData(resourceDocumentation),
			"attributes":          llx.MapData(attrs, types.Dict),
			"properties":          llx.MapData(props, types.Dict),
			"dependsOn":           llx.ArrayData(dependsOn, types.String),
			"deletionPolicy":      llx.StringData(deletionPolicy),
			"updateReplacePolicy": llx.StringData(updateReplacePolicy),
			"creationPolicy":      llx.DictData(creationPolicy),
			"updatePolicy":        llx.DictData(updatePolicy),
			"resourceMetadata":    llx.DictData(resourceMetadata),
		})
		if err != nil {
			return nil, err
		}

		s := pkg.(*mqlCloudformationResource)
		result = append(result, s)
	}

	return result, nil
}

func (x *mqlCloudformationOutput) id() (string, error) {
	return x.Name.Data, nil
}

func (x *mqlCloudformationParameter) id() (string, error) {
	return x.Name.Data, nil
}

func (r *mqlCloudformationTemplate) outputs() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.CloudformationConnection)
	template := conn.CftTemplate()
	if template.Node == nil || len(template.Node.Content) == 0 {
		return nil, nil
	}

	_, outputs, err := gatherMapValue(template.Node.Content[0], string(cft.Outputs))
	if err != nil && status.Code(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	result := make([]any, 0)
	for i := 0; i < len(outputs.Content); i += 2 {
		keyNode := outputs.Content[i]
		valueNode := outputs.Content[i+1]

		dict, err := convertYamlToDict(valueNode)
		if err != nil {
			return nil, err
		}

		value, err := nodeToDict(valueNode, "Value")
		if err != nil {
			return nil, err
		}

		description := ""
		if _, n, err := gatherMapValue(valueNode, "Description"); err == nil {
			description = n.Value
		}

		condition := ""
		if _, n, err := gatherMapValue(valueNode, "Condition"); err == nil {
			condition = n.Value
		}

		exportName := ""
		if _, exportNode, err := gatherMapValue(valueNode, "Export"); err == nil {
			if _, n, err := gatherMapValue(exportNode, "Name"); err == nil {
				exportName = n.Value
			}
		}

		pkg, err := CreateResource(r.MqlRuntime, "cloudformation.output", map[string]*llx.RawData{
			"name":        llx.StringData(keyNode.Value),
			"properties":  llx.DictData(dict),
			"value":       llx.DictData(value),
			"description": llx.StringData(description),
			"exportName":  llx.StringData(exportName),
			"condition":   llx.StringData(condition),
		})
		if err != nil {
			return nil, err
		}

		s := pkg.(*mqlCloudformationOutput)
		result = append(result, s)
	}

	return result, nil
}

func (r *mqlCloudformationTemplate) parameterList() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.CloudformationConnection)
	template := conn.CftTemplate()

	if template.Node == nil || len(template.Node.Content) == 0 {
		return nil, nil
	}
	_, params, err := gatherMapValue(template.Node.Content[0], string(cft.Parameters))
	if err != nil && status.Code(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	result := make([]any, 0)
	for i := 0; i < len(params.Content); i += 2 {
		keyNode := params.Content[i]
		valueNode := params.Content[i+1]

		paramType := ""
		if _, n, err := gatherMapValue(valueNode, "Type"); err == nil {
			paramType = n.Value
		}

		description := ""
		if _, n, err := gatherMapValue(valueNode, "Description"); err == nil {
			description = n.Value
		}

		allowedPattern := ""
		if _, n, err := gatherMapValue(valueNode, "AllowedPattern"); err == nil {
			allowedPattern = n.Value
		}

		constraintDescription := ""
		if _, n, err := gatherMapValue(valueNode, "ConstraintDescription"); err == nil {
			constraintDescription = n.Value
		}

		noEcho := false
		if _, n, err := gatherMapValue(valueNode, "NoEcho"); err == nil {
			noEcho = n.Value == "true" || n.Value == "True" || n.Value == "TRUE"
		}

		minLength, err := nodeToInt(valueNode, "MinLength")
		if err != nil {
			return nil, err
		}
		maxLength, err := nodeToInt(valueNode, "MaxLength")
		if err != nil {
			return nil, err
		}
		minValue, err := nodeToInt(valueNode, "MinValue")
		if err != nil {
			return nil, err
		}
		maxValue, err := nodeToInt(valueNode, "MaxValue")
		if err != nil {
			return nil, err
		}

		defaultDict, err := nodeToDict(valueNode, "Default")
		if err != nil {
			return nil, err
		}

		allowedValues, err := nodeToDictList(valueNode, "AllowedValues")
		if err != nil {
			return nil, err
		}

		pkg, err := CreateResource(r.MqlRuntime, "cloudformation.parameter", map[string]*llx.RawData{
			"__id":                  llx.StringData(keyNode.Value),
			"name":                  llx.StringData(keyNode.Value),
			"type":                  llx.StringData(paramType),
			"default":               llx.DictData(defaultDict),
			"description":           llx.StringData(description),
			"allowedValues":         llx.ArrayData(allowedValues, types.Dict),
			"allowedPattern":        llx.StringData(allowedPattern),
			"noEcho":                llx.BoolData(noEcho),
			"minLength":             llx.IntData(minLength),
			"maxLength":             llx.IntData(maxLength),
			"minValue":              llx.IntData(minValue),
			"maxValue":              llx.IntData(maxValue),
			"constraintDescription": llx.StringData(constraintDescription),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, pkg)
	}

	return result, nil
}

func (r *mqlCloudformationTemplate) types() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.CloudformationConnection)
	template := conn.CftTemplate()

	list, err := template.GetTypes()
	if err != nil {
		return nil, err
	}

	return convert.SliceAnyToInterface(list), nil
}
