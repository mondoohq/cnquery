package explorer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMqueryMerge(t *testing.T) {
	a := &Mquery{
		Mql:   "base",
		Title: "base title",
		Docs: &MqueryDocs{
			Desc:  "base desc",
			Audit: "base audit",
			Remediation: &Remediation{
				Items: []*TypedDoc{{
					Id:   "default",
					Desc: "a desciption",
				}},
			},
		},
	}
	b := &Mquery{
		Mql: "override",
		Docs: &MqueryDocs{
			Desc: "override desc",
			Remediation: &Remediation{
				Items: []*TypedDoc{{
					Id:   "default",
					Desc: "b desciption",
				}},
			},
		},
	}

	c := b.Merge(a)

	assert.NotEqual(t, a.Mql, c.Mql)
	assert.Equal(t, b.Mql, c.Mql)

	assert.Equal(t, a.Title, c.Title)
	assert.NotEqual(t, b.Title, c.Title)

	assert.NotEqual(t, a.Docs.Desc, c.Docs.Desc)
	assert.Equal(t, b.Docs.Desc, c.Docs.Desc)

	assert.Equal(t, a.Docs.Audit, c.Docs.Audit)
	assert.NotEqual(t, b.Docs.Audit, c.Docs.Audit)

	a.CodeId = "not this"
	b.CodeId = "not this either"
	assert.Equal(t, "", c.CodeId)

	// we want to make sure there are no residual shallow-copies
	cD := c.Docs.Remediation.Items[0].Desc
	a.Docs.Remediation.Items[0].Desc = "not this"
	a.Docs.Remediation.Items[0].Desc = "not this either"
	assert.Equal(t, cD, c.Docs.Remediation.Items[0].Desc)
}
