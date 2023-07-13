package docs

import (
	"fmt"
	"strconv"
	"strings"
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
	Fields           map[string]*LrDocsField `json:"fields,omitEmpty"`
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
	return fmt.Sprintf("Description: " + strconv.Quote(d.Description) + ",\n")
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
