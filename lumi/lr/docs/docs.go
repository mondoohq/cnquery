package docs

import (
	"fmt"
	"strconv"
	"strings"
)

type LrDocs struct {
	Resources map[string]*LrDocsEntry `json:"resources,omitempty"`
}

func (d LrDocs) MarshalGo() string {
	var sb strings.Builder

	for k := range d.Resources {
		if d.Resources[k] == nil {
			continue
		}
		sb.WriteString(fmt.Sprintf(`		"%s": {
			%s
		},
`, k, d.Resources[k].MarshalGo()))
	}

	return fmt.Sprintf(`var ResourceDocs = docs.LrDocs{
	Resources: map[string]*docs.LrDocsEntry{
       %s
	},
}
`, sb.String())
}

type LrDocsEntry struct {
	// Maturity of the resource: experimental, preview, public, deprecated
	// default maturity is public if nothing is provided
	Maturity string `json:"maturity,omitempty"`
	// this is just an indicator, we may want to replace this with native lumi resource platform information
	Platform  *LrDocsPlatform      `json:"platform,omitempty"`
	Docs      *LrDocsDocumentation `json:"docs,omitempty"`
	Resources []LrDocsRefs         `json:"resources ,omitempty"`
	Refs      []LrDocsRefs         `json:"refs,omitempty"`
	Snippets  []LrDocsSnippet      `json:"snippets,omitempty"`
}

func (d LrDocsEntry) MarshalGo() string {
	var sb strings.Builder

	if d.Platform != nil {
		sb.WriteString(fmt.Sprintf(`Platform: &docs.LrDocsPlatform{
			%s
		},`, d.Platform.MarshalGo()))
	}

	if d.Docs != nil {
		sb.WriteString(fmt.Sprintf(`Docs: &docs.LrDocsDocumentation{
			%s
		},`, d.Docs.MarshalGo()))
	}

	if len(d.Resources) > 0 {
		sb.WriteString("Resources: []docs.LrDocsRefs{\n")
		for i := range d.Resources {
			sb.WriteString("{\n")
			sb.WriteString(d.Resources[i].MarshalGo())
			sb.WriteString("},\n")
		}
		sb.WriteString("},")
	}

	if len(d.Refs) > 0 {
		sb.WriteString("Refs: []docs.LrDocsRefs{\n")
		for i := range d.Refs {
			sb.WriteString("{\n")
			sb.WriteString(d.Refs[i].MarshalGo())
			sb.WriteString("},\n")
		}
		sb.WriteString("},")
	}

	if len(d.Snippets) > 0 {
		sb.WriteString("Snippets: []docs.LrDocsSnippet{\n")
		for i := range d.Snippets {
			sb.WriteString("{\n")
			sb.WriteString(d.Snippets[i].MarshalGo())
			sb.WriteString("},\n")
		}
		sb.WriteString("},")
	}

	return sb.String()
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
	return fmt.Sprintf("Description: " + strconv.Quote(d.Description) + ",\n")
}

type LrDocsRefs struct {
	Title string `json:"title,omitempty"`
	Url   string `json:"url,omitempty"`
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

func (d LrDocsSnippet) MarshalGo() string {
	var sb strings.Builder
	sb.WriteString("Title: " + strconv.Quote(d.Title) + ",\n")
	sb.WriteString("Query: " + strconv.Quote(d.Query) + ",\n")
	return sb.String()
}
