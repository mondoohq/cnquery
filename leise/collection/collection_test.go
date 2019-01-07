package collection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectionDoc_Convert(t *testing.T) {
	data, err := parseYAML(`
collections:
- name: collection-123
  title: Collection 123
  tags:
  - 1.0.0
  - latest
  queries:
  - test1
queries:
- id: test1
  code: expect(1 == 2)
  title: Test1
`)
	if err != nil {
		assert.Nil(t, err)
		return
	}

	res, err := data.Convert()
	if err != nil {
		assert.Nil(t, err)
		return
	}

	collection := res.Collection[0]
	query := res.Queries[0]

	t.Run("metadata is read correctly", func(t *testing.T) {
		assert.Equal(t, "collection-123", collection.Name)
		assert.NotEmpty(t, collection.Id)
		assert.Equal(t, []string{"1.0.0", "latest"}, collection.Labels)
	})

	t.Run("query ID is updated", func(t *testing.T) {
		assert.Equal(t, collection.Queries[0], query.Id)
	})
}
