// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strconv"

	"github.com/cockroachdb/errors"
	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/atlassian/connection/jira"
)

// ---------- Custom fields ----------

func (a *mqlAtlassianJira) customFields() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("current connection does not allow jira access")
	}
	jiraClient := conn.Client()

	fields, _, err := jiraClient.Issue.Field.Gets(context.Background())
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(fields))
	for _, f := range fields {
		if f == nil || !f.Custom {
			continue
		}
		schemaType, schemaItems, schemaSystem, schemaCustom := "", "", "", ""
		var schemaCustomId int64
		if f.Schema != nil {
			schemaType = f.Schema.Type
			schemaItems = f.Schema.Items
			schemaSystem = f.Schema.System
			schemaCustom = f.Schema.Custom
			schemaCustomId = int64(f.Schema.CustomID)
		}
		mqlField, err := CreateResource(a.MqlRuntime, "atlassian.jira.customField",
			map[string]*llx.RawData{
				"id":             llx.StringData(f.ID),
				"key":            llx.StringData(f.Key),
				"name":           llx.StringData(f.Name),
				"description":    llx.StringData(f.Description),
				"searcherKey":    llx.StringData(f.SearcherKey),
				"searchable":     llx.BoolData(f.Searchable),
				"navigable":      llx.BoolData(f.Navigable),
				"orderable":      llx.BoolData(f.Orderable),
				"isLocked":       llx.BoolData(f.IsLocked),
				"schemaType":     llx.StringData(schemaType),
				"schemaItems":    llx.StringData(schemaItems),
				"schemaSystem":   llx.StringData(schemaSystem),
				"schemaCustom":   llx.StringData(schemaCustom),
				"schemaCustomId": llx.IntData(schemaCustomId),
				"screensCount":   llx.IntData(int64(f.ScreensCount)),
				"contextsCount":  llx.IntData(int64(f.ContextsCount)),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlField)
	}
	return res, nil
}

func (c *mqlAtlassianJiraCustomField) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return "atlassian.jira.customField/" + c.Id.Data, nil
}

// ---------- Workflows ----------

func (a *mqlAtlassianJira) workflows() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("current connection does not allow jira access")
	}
	jiraClient := conn.Client()

	res := []any{}
	startAt := 0
	const pageSize = 50
	for {
		page, _, err := jiraClient.Workflow.Gets(context.Background(), nil, startAt, pageSize)
		if err != nil {
			return nil, err
		}
		if page == nil {
			break
		}
		for _, wf := range page.Values {
			if wf == nil {
				continue
			}
			mqlWf, err := workflowToMql(a, wf)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlWf)
		}
		if page.IsLast || len(page.Values) == 0 {
			break
		}
		startAt += len(page.Values)
	}
	return res, nil
}

func workflowToMql(a *mqlAtlassianJira, wf *models.WorkflowScheme) (any, error) {
	id, name := "", ""
	if wf.ID != nil {
		id = wf.ID.EntityID
		name = wf.ID.Name
	}
	if id == "" {
		id = name
	}

	statuses := make([]any, 0, len(wf.Statuses))
	for _, s := range wf.Statuses {
		if s == nil {
			continue
		}
		statuses = append(statuses, s.Name)
	}

	transitions := make([]any, 0, len(wf.Transitions))
	for _, t := range wf.Transitions {
		if t == nil {
			continue
		}
		from := []any{}
		for _, f := range t.From {
			if f == nil {
				continue
			}
			from = append(from, f.StatusReference)
		}
		to := ""
		if t.To != nil {
			to = t.To.StatusReference
		}
		entry := map[string]any{
			"id":          t.ID,
			"name":        t.Name,
			"type":        t.Type,
			"description": t.Description,
			"from":        from,
			"to":          to,
			"conditions":  ruleConfigurationsToAny(t.Conditions),
			"validators":  ruleListToAny(t.Validators),
			"actions":     ruleListToAny(t.Actions),
			"triggers":    ruleListToAny(t.Triggers),
		}
		transitions = append(transitions, entry)
	}

	return CreateResource(a.MqlRuntime, "atlassian.jira.workflow",
		map[string]*llx.RawData{
			"id":          llx.StringData(id),
			"name":        llx.StringData(name),
			"description": llx.StringData(wf.Description),
			"isDefault":   llx.BoolData(wf.IsDefault),
			"statuses":    llx.ArrayData(statuses, "string"),
			"transitions": llx.ArrayData(transitions, "dict"),
		})
}

func ruleListToAny(rules []*models.WorkflowRuleConfigurationScheme) []any {
	out := make([]any, 0, len(rules))
	for _, r := range rules {
		if r == nil {
			continue
		}
		out = append(out, map[string]any{
			"id":         r.ID,
			"ruleKey":    r.RuleKey,
			"parameters": r.Parameters,
		})
	}
	return out
}

// maxConditionGroupDepth caps recursion through nested workflow condition
// groups. The Jira UI doesn't allow trees this deep in practice, so anything
// beyond this is almost certainly a malformed payload — truncate rather than
// risk an unbounded recursion.
const maxConditionGroupDepth = 20

func ruleConfigurationsToAny(cg *models.ConditionGroupConfigurationScheme) any {
	return ruleConfigurationsToAnyDepth(cg, 0)
}

func ruleConfigurationsToAnyDepth(cg *models.ConditionGroupConfigurationScheme, depth int) any {
	if cg == nil || depth >= maxConditionGroupDepth {
		return nil
	}
	groups := make([]any, 0, len(cg.ConditionGroups))
	for _, g := range cg.ConditionGroups {
		groups = append(groups, ruleConfigurationsToAnyDepth(g, depth+1))
	}
	return map[string]any{
		"operation":       cg.Operation,
		"conditions":      ruleListToAny(cg.Conditions),
		"conditionGroups": groups,
	}
}

func (w *mqlAtlassianJiraWorkflow) id() (string, error) {
	if w.Id.Error != nil {
		return "", w.Id.Error
	}
	return "atlassian.jira.workflow/" + w.Id.Data, nil
}

// ---------- Project roles ----------

func (a *mqlAtlassianJiraProject) roles() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("current connection does not allow jira access")
	}
	jiraClient := conn.Client()

	if a.Key.Error != nil {
		return nil, a.Key.Error
	}
	if a.Id.Error != nil {
		return nil, a.Id.Error
	}
	projectKey := a.Key.Data
	if projectKey == "" {
		projectKey = a.Id.Data
	}
	if projectKey == "" {
		return []any{}, nil
	}

	details, _, err := jiraClient.Project.Role.Details(context.Background(), projectKey)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(details))
	for _, d := range details {
		if d == nil {
			continue
		}
		// Fetch the full role to get actor assignments. A single role
		// failing (rate-limit, transient 5xx, permission edge case) shouldn't
		// fail the whole project — log and skip.
		role, _, err := jiraClient.Project.Role.Get(context.Background(), projectKey, d.ID)
		if err != nil {
			log.Warn().Err(err).Str("project", projectKey).Int("roleId", d.ID).Msg("failed to fetch project role details")
			continue
		}
		actors := []any{}
		if role != nil {
			for _, a := range role.Actors {
				if a == nil {
					continue
				}
				entry := map[string]any{
					"id":          a.ID,
					"type":        a.Type,
					"name":        a.Name,
					"displayName": a.DisplayName,
					"avatarUrl":   a.AvatarURL,
				}
				if a.ActorUser != nil {
					entry["accountId"] = a.ActorUser.AccountID
				}
				actors = append(actors, entry)
			}
		}
		mqlRole, err := CreateResource(a.MqlRuntime, "atlassian.jira.project.role",
			map[string]*llx.RawData{
				"id":               llx.StringData(projectKey + "/" + strconv.Itoa(d.ID)),
				"projectKey":       llx.StringData(projectKey),
				"roleId":           llx.IntData(int64(d.ID)),
				"name":             llx.StringData(d.Name),
				"description":      llx.StringData(d.Description),
				"admin":            llx.BoolData(d.Admin),
				"default":          llx.BoolData(d.Default),
				"roleConfigurable": llx.BoolData(d.RoleConfigurable),
				"actors":           llx.ArrayData(actors, "dict"),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRole)
	}
	return res, nil
}

func (r *mqlAtlassianJiraProjectRole) id() (string, error) {
	if r.Id.Error != nil {
		return "", r.Id.Error
	}
	return "atlassian.jira.project.role/" + r.Id.Data, nil
}
