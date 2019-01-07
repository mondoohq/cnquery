package collection

import (
	"encoding/json"
	"errors"

	"github.com/ghodss/yaml"
	"go.mondoo.io/mondoo/db"
	"go.mondoo.io/mondoo/leise"
	"go.mondoo.io/mondoo/llx"
)

type collectionsDoc struct {
	Collections []*collectionDoc `json:"collections"`
	Queries     []*queryDoc      `json:"queries"`
}

type collectionDoc struct {
	Name        string   `json:"name"`
	Labels      []string `json:"tags,omitempty"`
	Queries     []string `json:"queries,omitempty"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
}

type queryDoc struct {
	ID          string            `json:"id,omitempty"`
	Code        string            `json:"code,omitempty"`
	Queries     []string          `json:"queries,omitempty"`
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// FromYAML and turn into a collections bundle
func FromYAML(yaml string) (*db.CollectionsBundle, error) {
	doc, err := parseYAML(yaml)
	if err != nil {
		return nil, err
	}
	return doc.Convert()
}

// parseYAML collection and return internal collections doc
func parseYAML(data string) (*collectionsDoc, error) {
	res := collectionsDoc{}
	err := yaml.Unmarshal([]byte(data), &res)
	if err != nil {
		return nil, errors.New("Failed to parse collection yaml: " + err.Error())
	}
	return &res, res.validate()
}

// FromJSON and turn into a collections bundle
func FromJSON(json string) (*db.CollectionsBundle, error) {
	doc, err := parseJSON(json)
	if err != nil {
		return nil, err
	}
	return doc.Convert()
}

func parseJSON(data string) (*collectionsDoc, error) {
	res := collectionsDoc{}
	err := json.Unmarshal([]byte(data), &res)
	if err != nil {
		return nil, errors.New("Failed to parse collection json: " + err.Error())
	}
	return &res, res.validate()
}

func (c *collectionsDoc) validate() error {
	if len(c.Collections) == 0 {
		return errors.New("No collections found")
	}
	return nil
}

// Convert extracts and parses the collections and queries
func (c *collectionsDoc) Convert() (*db.CollectionsBundle, error) {
	queryIDs := map[string]string{}
	resQuery := make([]*db.Query, len(c.Queries))
	var resCode []*llx.CodeBundle

	for i := range c.Queries {
		yamlOrg := c.Queries[i]

		// compile leise code
		code, err := leise.Compile(yamlOrg.Code, llx.DefaultRegistry.Schema())
		if err != nil {
			return nil, err
		}
		resCode = append(resCode, code)

		// create query and link the codes to it
		query := &db.Query{
			Code:        yamlOrg.Code,
			CodeRefs:    []string{code.Code.Id},
			Queries:     yamlOrg.Queries,
			Title:       yamlOrg.Title,
			Description: yamlOrg.Description,
			Attributes:  yamlOrg.Attributes,
		}
		query.UpdateID()

		queryIDs[yamlOrg.ID] = query.Id
		resQuery[i] = query
	}

	resCollection := make([]*db.Collection, len(c.Collections))
	for i := range c.Collections {
		cur := c.Collections[i]

		// check that all referened queries are available for the collection
		queries := make([]string, len(cur.Queries))
		for j := range cur.Queries {
			oldID := cur.Queries[j]
			nuID, found := queryIDs[oldID]
			if !found {
				return nil, errors.New("Cannot find query ID '" + oldID + "' in uploaded collection.")
			}
			queries[j] = nuID
		}

		col := &db.Collection{
			Name:        cur.Name,
			Labels:      cur.Labels,
			Queries:     queries,
			Title:       cur.Title,
			Description: cur.Description,
		}
		col.UpdateID()
		resCollection[i] = col
	}

	// TODO: validate queries and collections
	res := db.CollectionsBundle{
		Collection: resCollection,
		Queries:    resQuery,
		Code:       resCode,
	}
	return &res, nil
}

// CollectionsDoc generates a collections structure for the given ID
// that has all the queries included. It is great for saving it to a file.
func bundle2doc(bundle *db.CollectionsBundle) (*collectionsDoc, error) {
	res := collectionsDoc{}

	res.Collections = make([]*collectionDoc, len(bundle.Collection))
	for i := range bundle.Collection {
		collection := bundle.Collection[i]
		res.Collections[i] = &collectionDoc{
			Name:        collection.Name,
			Labels:      collection.Labels,
			Queries:     collection.Queries,
			Title:       collection.Title,
			Description: collection.Description,
		}
	}

	res.Queries = make([]*queryDoc, len(bundle.Queries))
	for i := range bundle.Queries {
		query := bundle.Queries[i]
		res.Queries[i] = &queryDoc{
			ID:          query.Id,
			Queries:     query.Queries,
			Title:       query.Title,
			Description: query.Description,
			Attributes:  query.Attributes,
			Tags:        query.Tags,
			Code:        query.Code,
		}
	}

	return &res, nil
}

// ToYAML turns a bundle into YAML
func ToYAML(bundle *db.CollectionsBundle) ([]byte, error) {
	doc, err := bundle2doc(bundle)
	if err != nil {
		return nil, err
	}

	return yaml.Marshal(doc)
}

// // ResolveQueries turns IDs into all query objects
// func (c *Service) ResolveQueries(collection *db.Collection) (map[string]*db.Query, error) {
// 	queries := map[string]*db.Query{}
// 	remaining := collection.Queries
// 	for i := 0; i < len(remaining); i++ {
// 		qid := remaining[i]
// 		query, err := c.collections.GetQuery(qid)
// 		if err != nil {
// 			return nil, err
// 		}

// 		queries[qid] = query

// 		for _, rid := range query.Queries {
// 			if _, ok := queries[rid]; !ok {
// 				remaining = append(remaining, rid)
// 			}
// 		}
// 	}
// 	return queries, nil
// }
