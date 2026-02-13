// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package lrcore

import (
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
)

type LrDocs struct {
	Resources map[string]*LrDocsEntry `json:"resources,omitempty"`
}

type LrDocsEntry struct {
	// Maturity of the resource: experimental, preview, public, deprecated
	// default maturity is public if nothing is provided
	Maturity string `json:"maturity,omitempty"`
	// this is just an indicator, we may want to replace this with native MQL resource platform information
	Platform         *LrDocsPlatform         `json:"platform,omitempty"`
	Docs             *LrDocsDocumentation    `json:"docs,omitempty"`
	Resources        []LrDocsRefs            `json:"resources,omitempty"`
	Fields           map[string]*LrDocsField `json:"fields,omitempty"`
	Refs             []LrDocsRefs            `json:"refs,omitempty"`
	Snippets         []LrDocsSnippet         `json:"snippets,omitempty"`
	IsPrivate        bool                    `json:"is_private,omitempty"`
	MinMondooVersion string                  `json:"min_mondoo_version,omitempty"`
}

type LrDocsPlatform struct {
	Name   []string `json:"name,omitempty"`
	Family []string `json:"family,omitempty"`
}

func (d LrDocsPlatform) MarshalGo() string {
	var sb strings.Builder
	if len(d.Name) > 0 {
		sb.WriteString("Name: []string{\"" + strings.Join(d.Name, "\",\"") + "\"},")
	}

	if len(d.Family) > 0 {
		sb.WriteString("Family: []string{\"" + strings.Join(d.Family, "\",\"") + "\"},")
	}

	return sb.String()
}

type LrDocsDocumentation struct {
	Description string `json:"desc,omitempty"`
}

func (d LrDocsDocumentation) MarshalGo() string {
	return fmt.Sprint("Description: " + strconv.Quote(d.Description) + ",\n")
}

type LrDocsRefs struct {
	Title string `json:"title,omitempty"`
	Url   string `json:"url,omitempty"`
}

type LrDocsField struct {
	MinMondooVersion string `json:"min_mondoo_version,omitempty"`
}

func (d LrDocsRefs) MarshalGo() string {
	var sb strings.Builder
	sb.WriteString("Title: " + strconv.Quote(d.Title) + ",\n")
	sb.WriteString("Url: " + strconv.Quote(d.Url) + ",\n")
	return sb.String()
}

type LrDocsSnippet struct {
	Title string `json:"title,omitempty"`
	Query string `json:"query,omitempty"`
}

func InjectMetadata(schema *resources.Schema, docs *LrDocs) {
	for resource, rdoc := range docs.Resources {
		info, ok := schema.Resources[resource]
		if !ok {
			continue
		}

		info.MinMondooVersion = rdoc.MinMondooVersion

		for field, fdoc := range rdoc.Fields {
			finfo, ok := info.Fields[field]
			if !ok {
				continue
			}

			finfo.MinMondooVersion = fdoc.MinMondooVersion
		}
	}
}

func (lr *LR) GenerateDocs(currentVersion, defaultVersion string, existingDocs LrDocs) (LrDocs, error) {
	// to ensure we generate the same markdown, we sort the resources first
	sort.SliceStable(lr.Resources, func(i, j int) bool {
		return lr.Resources[i].ID < lr.Resources[j].ID
	})

	docs := LrDocs{Resources: map[string]*LrDocsEntry{}}

	fields := map[string][]*BasicField{}
	isPrivate := map[string]bool{}
	for i := range lr.Resources {
		id := lr.Resources[i].ID
		isPrivate[id] = lr.Resources[i].IsPrivate
		docs.Resources[id] = nil
		if lr.Resources[i].Body != nil {
			basicFields := []*BasicField{}
			for _, f := range lr.Resources[i].Body.Fields {
				if f.BasicField != nil {
					basicFields = append(basicFields, f.BasicField)
				}
			}
			fields[id] = basicFields
		}
	}

	// if we have docs from existing manifest, merge them in
	if existingDocs.Resources != nil {
		maps.Copy(docs.Resources, existingDocs.Resources)
	}
	// ensure default values and fields are set
	for k := range docs.Resources {
		docs.Resources[k] = ensureDefaults(k, docs.Resources[k], currentVersion, defaultVersion)
		mergeFields(docs.Resources[k], fields[k], currentVersion, defaultVersion)
		// Merge in other doc fields from core.lr
		docs.Resources[k].IsPrivate = isPrivate[k]
	}

	return docs, nil
}

func mergeFields(entry *LrDocsEntry, fields []*BasicField, currentVersion, defaultVersion string) {
	if entry == nil && len(fields) > 0 {
		entry = &LrDocsEntry{}
		entry.Fields = map[string]*LrDocsField{}
	} else if entry == nil {
		return
	} else if entry.Fields == nil {
		entry.Fields = map[string]*LrDocsField{}
	}
	docFields := entry.Fields
	for _, f := range fields {
		if docFields[f.ID] == nil {
			fDoc := &LrDocsField{
				MinMondooVersion: currentVersion,
			}
			entry.Fields[f.ID] = fDoc
		} else if entry.Fields[f.ID].MinMondooVersion == defaultVersion && currentVersion != defaultVersion {
			entry.Fields[f.ID].MinMondooVersion = currentVersion
		}
		// Scrub field version if same as resource
		if entry.Fields[f.ID].MinMondooVersion == entry.MinMondooVersion {
			entry.Fields[f.ID].MinMondooVersion = ""
		}
	}
}

func ensureDefaults(id string, entry *LrDocsEntry, currentVersion, defaultVersion string) *LrDocsEntry {
	for _, k := range platformMappingKeys {
		if entry == nil {
			entry = &LrDocsEntry{}
		}
		if entry.MinMondooVersion == "" {
			entry.MinMondooVersion = currentVersion
		} else if entry.MinMondooVersion == defaultVersion && currentVersion != defaultVersion {
			// Update to specified version if previously set to default
			entry.MinMondooVersion = currentVersion
		}
		if strings.HasPrefix(id, k) {
			entry.Platform = &LrDocsPlatform{
				Name: platformMapping[k],
			}
		}
	}
	return entry
}

// required to be before more detail platform to ensure the right mapping
var platformMappingKeys = []string{
	"aws", "gcp", "k8s", "azure", "azurerm", "arista", "equinix", "ms365", "msgraph", "vsphere", "esxi", "terraform", "terraform.state", "terraform.plan",
}

var platformMapping = map[string][]string{
	"aws":             {"aws"},
	"gcp":             {"gcp"},
	"k8s":             {"kubernetes"},
	"azure":           {"azure"},
	"azurerm":         {"azure"},
	"arista":          {"arista-eos"},
	"equinix":         {"equinix"},
	"ms365":           {"microsoft365"},
	"msgraph":         {"microsoft365"},
	"vsphere":         {"vmware-esxi", "vmware-vsphere"},
	"esxi":            {"vmware-esxi", "vmware-vsphere"},
	"terraform":       {"terraform-hcl"},
	"terraform.state": {"terraform-state"},
	"terraform.plan":  {"terraform-plan"},
}
