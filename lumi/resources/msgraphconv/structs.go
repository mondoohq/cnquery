package msgraphconv

// this package creates a copy of the msgraph object that we use for embedded struct. This is required since microsoft
// defines structs with lower case and does not attach json tags or implements the standard marshalling function

import (
	"time"

	"github.com/microsoftgraph/msgraph-beta-sdk-go/models/microsoft/graph"
)

func toString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

type AssignedPlan struct {
	AssignedDateTime *time.Time `json:"assignedDateTime"`
	CapabilityStatus string     `json:"capabilityStatus"`
	Service          string     `json:"service"`
	ServicePlanId    string     `json:"servicePlanId"`
}

func NewAssignedPlans(p []graph.AssignedPlan) []AssignedPlan {
	res := []AssignedPlan{}
	for i := range p {
		res = append(res, NewAssignedPlan(p[i]))
	}
	return res
}

func NewAssignedPlan(p graph.AssignedPlan) AssignedPlan {
	return AssignedPlan{
		AssignedDateTime: p.GetAssignedDateTime(),
		CapabilityStatus: toString(p.GetCapabilityStatus()),
		Service:          toString(p.GetService()),
		ServicePlanId:    toString(p.GetServicePlanId()),
	}
}

type VerifiedDomain struct {
	Capabilities string `json:"capabilities"`
	IsDefault    bool   `json:"isDefault"`
	IsInitial    bool   `json:"isInitial"`
	Name         string `json:"name"`
	Type         string `json:"type"`
}

func NewVerifiedDomains(p []graph.VerifiedDomain) []VerifiedDomain {
	res := []VerifiedDomain{}
	for i := range p {
		res = append(res, NewVerifiedDomain(p[i]))
	}
	return res
}

func NewVerifiedDomain(p graph.VerifiedDomain) VerifiedDomain {
	return VerifiedDomain{
		Capabilities: toString(p.GetCapabilities()),
		IsDefault:    toBool(p.GetIsDefault()),
		IsInitial:    toBool(p.GetIsInitial()),
		Name:         toString(p.GetName()),
		Type:         toString(p.GetType()),
	}
}

type UnifiedRolePermission struct {
	AllowedResourceActions  []string `json:"allowedResourceActions"`
	Condition               string   `json:"condition"`
	ExcludedResourceActions []string `json:"excludedResourceActions"`
}

func NewUnifiedRolePermissions(p []graph.UnifiedRolePermission) []UnifiedRolePermission {
	res := []UnifiedRolePermission{}
	for i := range p {
		res = append(res, NewUnifiedRolePermission(p[i]))
	}
	return res
}

func NewUnifiedRolePermission(p graph.UnifiedRolePermission) UnifiedRolePermission {
	return UnifiedRolePermission{
		AllowedResourceActions:  p.GetAllowedResourceActions(),
		Condition:               toString(p.GetCondition()),
		ExcludedResourceActions: p.GetExcludedResourceActions(),
	}
}

type DirectorySetting struct {
	DisplayName string         `json:"displayName"`
	TemplateId  string         `json:"templateId"`
	Values      []SettingValue `json:"values"`
}

type SettingValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func NewDirectorySettings(p []graph.DirectorySetting) []DirectorySetting {
	res := []DirectorySetting{}
	for i := range p {
		res = append(res, NewDirectorySetting(p[i]))
	}
	return res
}

func NewDirectorySetting(p graph.DirectorySetting) DirectorySetting {
	values := []SettingValue{}
	entries := p.GetValues()
	for i := range entries {
		values = append(values, SettingValue{
			Name:  toString(entries[i].GetName()),
			Value: toString(entries[i].GetValue()),
		})
	}

	return DirectorySetting{
		DisplayName: toString(p.GetDisplayName()),
		TemplateId:  toString(p.GetTemplateId()),
		Values:      values,
	}
}
