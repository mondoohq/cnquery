// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
)

// The dependency graph turns the one-hop reference index into transitive
// reachability: a resource's full set of dependencies and the full blast
// radius of a change to it, plus whole-configuration queries (orphans, edges).

type mqlTerraformGraphInternal struct {
	tf *mqlTerraform
}

func (t *mqlTerraform) graph() (*mqlTerraformGraph, error) {
	r, err := CreateResource(t.MqlRuntime, "terraform.graph", map[string]*llx.RawData{
		"__id": llx.StringData("terraform.graph"),
	})
	if err != nil {
		return nil, err
	}
	g := r.(*mqlTerraformGraph)
	g.tf = t
	return g, nil
}

// terraform resolves the parent terraform resource. When the graph is queried
// directly (terraform.graph.*) it is constructed bare, so we fetch it here.
func (g *mqlTerraformGraph) terraform() (*mqlTerraform, error) {
	if g.tf != nil {
		return g.tf, nil
	}
	o, err := CreateResource(g.MqlRuntime, "terraform", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	tf := o.(*mqlTerraform)
	if err := tf.refreshCache(nil); err != nil {
		return nil, err
	}
	g.tf = tf
	return tf, nil
}

// --- transitive traversal on resources / data sources ---

func (c *mqlTerraformResource) dependencies() ([]any, error) {
	if c.tf == nil || c.tfBlock == nil {
		return []any{}, nil
	}
	return c.tf.transitiveDeps(c.tfBlock, true)
}

func (c *mqlTerraformResource) dependents() ([]any, error) {
	if c.tf == nil || c.tfBlock == nil {
		return []any{}, nil
	}
	return c.tf.transitiveDeps(c.tfBlock, false)
}

func (c *mqlTerraformDatasource) dependencies() ([]any, error) {
	if c.tf == nil || c.tfBlock == nil {
		return []any{}, nil
	}
	return c.tf.transitiveDeps(c.tfBlock, true)
}

func (c *mqlTerraformDatasource) dependents() ([]any, error) {
	if c.tf == nil || c.tfBlock == nil {
		return []any{}, nil
	}
	return c.tf.transitiveDeps(c.tfBlock, false)
}

// transitiveDeps walks the dependency graph from start, forward (dependencies)
// or backward (dependents), returning every reachable block exactly once.
func (t *mqlTerraform) transitiveDeps(start *mqlTerraformBlock, forward bool) ([]any, error) {
	if _, err := t.reverseRefIndex(); err != nil {
		return nil, err
	}

	startID, _ := start.id()
	visited := map[string]bool{startID: true}
	queue := []string{startID}
	out := []any{}

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		var neighbors []*mqlTerraformBlock
		if forward {
			neighbors = t.fwdIndex[id]
		} else {
			for _, raw := range t.refIndex[id] {
				neighbors = append(neighbors, raw.(*mqlTerraformBlock))
			}
		}

		for _, nb := range neighbors {
			nid, _ := nb.id()
			if visited[nid] {
				continue
			}
			visited[nid] = true
			out = append(out, nb)
			queue = append(queue, nid)
		}
	}
	return out, nil
}

// --- terraform.graph ---

// graphNodes returns the managed-resource and data-source blocks (the graph's
// real nodes).
func (g *mqlTerraformGraph) graphNodes() ([]*mqlTerraformBlock, error) {
	tf, err := g.terraform()
	if err != nil {
		return nil, err
	}
	if err := tf.refreshCache(nil); err != nil {
		return nil, err
	}
	nodes := append([]*mqlTerraformBlock{}, tf.mqlTerraformInternal.resources...)
	for i := range tf.Datasources.Data {
		nodes = append(nodes, tf.Datasources.Data[i].(*mqlTerraformBlock))
	}
	return nodes, nil
}

func (g *mqlTerraformGraph) nodes() ([]any, error) {
	nodes, err := g.graphNodes()
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n)
	}
	return out, nil
}

func (g *mqlTerraformGraph) orphans() ([]any, error) {
	nodes, err := g.graphNodes()
	if err != nil {
		return nil, err
	}
	tf, err := g.terraform()
	if err != nil {
		return nil, err
	}
	rev, err := tf.reverseRefIndex()
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, n := range nodes {
		id, _ := n.id()
		if len(rev[id]) == 0 {
			out = append(out, n)
		}
	}
	return out, nil
}

func (g *mqlTerraformGraph) edges() ([]any, error) {
	nodes, err := g.graphNodes()
	if err != nil {
		return nil, err
	}

	tf, err := g.terraform()
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, from := range nodes {
		fromAddr := blockAddress(from)
		refs, err := terraformReferences(g.MqlRuntime, tf, from)
		if err != nil {
			return nil, err
		}
		for i := range refs {
			ref := refs[i].(*mqlTerraformReference)
			to := ref.GetBlock()
			if to.Error != nil || to.Data == nil {
				continue
			}
			e, err := CreateResource(g.MqlRuntime, "terraform.graph.edge", map[string]*llx.RawData{
				"from":     llx.StringData(fromAddr),
				"to":       llx.StringData(blockAddress(to.Data)),
				"argument": llx.StringData(ref.Argument.Data),
			})
			if err != nil {
				return nil, err
			}
			out = append(out, e)
		}
	}
	return out, nil
}

// blockAddress renders a block's "type.name" address, prefixed with "data."
// for data sources.
func blockAddress(b *mqlTerraformBlock) string {
	switch b.Type.Data {
	case "data":
		return "data." + labelAt(b, 0) + "." + labelAt(b, 1)
	case "resource":
		return labelAt(b, 0) + "." + labelAt(b, 1)
	case "variable":
		return "var." + labelAt(b, 0)
	case "output":
		return "output." + labelAt(b, 0)
	case "module":
		return "module." + labelAt(b, 0)
	case "locals":
		return "local"
	default:
		return labelAt(b, 0)
	}
}

func (e *mqlTerraformGraphEdge) id() (string, error) {
	return "terraform.graph.edge/" + e.From.Data + "->" + e.To.Data + "/" + e.Argument.Data, nil
}
