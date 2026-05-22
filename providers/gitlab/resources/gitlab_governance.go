// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/gitlab/connection"
	"go.mondoo.com/mql/v13/types"
)

// -----------------------------------------------------------------------------
// id() methods
// -----------------------------------------------------------------------------

func (r *mqlGitlabProjectApprovalRule) id() (string, error) {
	return "gitlab.project.approvalRule/" + strconv.FormatInt(r.Id.Data, 10), nil
}

func (r *mqlGitlabProjectCodeowners) id() (string, error) {
	return "gitlab.project.codeowners/" + r.Path.Data, nil
}

func (r *mqlGitlabProjectCodeownersRule) id() (string, error) {
	return "gitlab.project.codeowners.rule/" +
		strconv.FormatInt(r.LineNumber.Data, 10) + "/" +
		r.Pattern.Data, nil
}

// -----------------------------------------------------------------------------
// Runner — fetch full details on demand via GetRunnerDetails. Stored in an
// Internal struct so a single API call satisfies every detail accessor.
// -----------------------------------------------------------------------------

type mqlGitlabProjectRunnerInternal struct {
	detailsOnce sync.Once
	details     *gitlab.RunnerDetails
	detailsErr  error
}

func (r *mqlGitlabProjectRunner) loadDetails() (*gitlab.RunnerDetails, error) {
	r.detailsOnce.Do(func() {
		conn := r.MqlRuntime.Connection.(*connection.GitLabConnection)
		details, _, err := conn.Client().Runners.GetRunnerDetails(int(r.Id.Data))
		r.details = details
		r.detailsErr = err
	})
	return r.details, r.detailsErr
}

func (r *mqlGitlabProjectRunner) tagList() ([]any, error) {
	d, err := r.loadDetails()
	if err != nil || d == nil {
		return []any{}, err
	}
	out := make([]any, 0, len(d.TagList))
	for _, t := range d.TagList {
		out = append(out, t)
	}
	return out, nil
}

func (r *mqlGitlabProjectRunner) runUntagged() (bool, error) {
	d, err := r.loadDetails()
	if err != nil || d == nil {
		return false, err
	}
	return d.RunUntagged, nil
}

func (r *mqlGitlabProjectRunner) lockedToProject() (bool, error) {
	d, err := r.loadDetails()
	if err != nil || d == nil {
		return false, err
	}
	return d.Locked, nil
}

func (r *mqlGitlabProjectRunner) accessLevel() (string, error) {
	d, err := r.loadDetails()
	if err != nil || d == nil {
		return "", err
	}
	return d.AccessLevel, nil
}

func (r *mqlGitlabProjectRunner) maximumTimeout() (int64, error) {
	d, err := r.loadDetails()
	if err != nil || d == nil {
		return 0, err
	}
	return d.MaximumTimeout, nil
}

func (r *mqlGitlabProjectRunner) contactedAt() (*time.Time, error) {
	d, err := r.loadDetails()
	if err != nil || d == nil {
		return nil, err
	}
	return d.ContactedAt, nil
}

func (r *mqlGitlabProjectRunner) maintenanceNote() (string, error) {
	d, err := r.loadDetails()
	if err != nil || d == nil {
		return "", err
	}
	return d.MaintenanceNote, nil
}

func (r *mqlGitlabProjectRunner) projects() ([]any, error) {
	d, err := r.loadDetails()
	if err != nil || d == nil {
		return []any{}, err
	}
	out := make([]any, 0, len(d.Projects))
	for _, p := range d.Projects {
		out = append(out, p.PathWithNamespace)
	}
	return out, nil
}

func (r *mqlGitlabProjectRunner) groups() ([]any, error) {
	d, err := r.loadDetails()
	if err != nil || d == nil {
		return []any{}, err
	}
	out := make([]any, 0, len(d.Groups))
	for _, g := range d.Groups {
		out = append(out, g.WebURL)
	}
	return out, nil
}

// -----------------------------------------------------------------------------
// CODEOWNERS
// -----------------------------------------------------------------------------

// codeownersCandidatePaths lists the locations GitLab checks for a
// CODEOWNERS file, in priority order. Source:
// https://docs.gitlab.com/user/project/codeowners/reference/#codeowners-file-location
var codeownersCandidatePaths = []string{"CODEOWNERS", ".gitlab/CODEOWNERS", "docs/CODEOWNERS"}

func (p *mqlGitlabProject) codeowners() (*mqlGitlabProjectCodeowners, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)
	projectID := int(p.Id.Data)
	defaultBranch := p.DefaultBranch.Data
	if defaultBranch == "" {
		// fall back to "main" — most modern GitLab projects use it;
		// when this still 404s we surface a present=false resource.
		defaultBranch = "main"
	}

	var content []byte
	var foundPath string
	for _, path := range codeownersCandidatePaths {
		body, resp, err := conn.Client().RepositoryFiles.GetRawFile(projectID, path, &gitlab.GetRawFileOptions{
			Ref: gitlab.Ptr(defaultBranch),
		})
		if err == nil {
			content = body
			foundPath = path
			break
		}
		if resp != nil && resp.StatusCode == 404 {
			continue
		}
		return nil, err
	}

	rules := parseCodeowners(string(content))
	mqlRules := make([]any, 0, len(rules))
	for _, rule := range rules {
		owners := make([]any, 0, len(rule.Owners))
		for _, o := range rule.Owners {
			owners = append(owners, o)
		}
		mqlRule, err := CreateResource(p.MqlRuntime, "gitlab.project.codeowners.rule", map[string]*llx.RawData{
			"lineNumber":        llx.IntData(int64(rule.LineNumber)),
			"section":           llx.StringData(rule.Section),
			"optional":          llx.BoolData(rule.Optional),
			"required":          llx.BoolData(rule.Required),
			"approvalsRequired": llx.IntData(int64(rule.ApprovalsRequired)),
			"pattern":           llx.StringData(rule.Pattern),
			"owners":            llx.ArrayData(owners, types.String),
		})
		if err != nil {
			return nil, err
		}
		mqlRules = append(mqlRules, mqlRule)
	}

	res, err := CreateResource(p.MqlRuntime, "gitlab.project.codeowners", map[string]*llx.RawData{
		"present": llx.BoolData(foundPath != ""),
		"path":    llx.StringData(foundPath),
		"content": llx.StringData(string(content)),
		"rules":   llx.ArrayData(mqlRules, types.Resource("gitlab.project.codeowners.rule")),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGitlabProjectCodeowners), nil
}

// codeownersRule is the parsed form of a single line/pattern in a
// CODEOWNERS file.
type codeownersRule struct {
	LineNumber        int
	Section           string
	Optional          bool
	Required          bool
	ApprovalsRequired int
	Pattern           string
	Owners            []string
}

// sectionHeader matches `[Section]`, `[Section][2]`, `^[Section]`,
// `^[Section][2]`. The first group captures the optional `^`, the
// second the section name, the third the optional approval count.
var sectionHeader = regexp.MustCompile(`^(\^?)\[([^\]]+)\](?:\[(\d+)\])?\s*$`)

// parseCodeowners turns the raw CODEOWNERS text into a slice of
// rules. Comments (lines starting with #) and blank lines are
// skipped. Section state carries forward until the next header.
//
// CODEOWNERS spec (the bits we care about for audit):
//   - `pattern owner [owner …]` — owner is `@user`, `@@group/path`,
//     or an email address.
//   - `[Section]` — required section. Subsequent rules inherit
//     required=true. `[Section][2]` overrides the section-level
//     approvals-required count.
//   - `^[Section]` — optional section. Subsequent rules inherit
//     optional=true (and required=false).
//
// Edge cases we intentionally do not model: `\` line-continuation
// (rare in practice), nested sections (not supported by GitLab),
// pattern globs that include literal whitespace (CODEOWNERS doesn't
// support quoting).
func parseCodeowners(content string) []codeownersRule {
	if content == "" {
		return nil
	}
	var rules []codeownersRule
	var currentSection string
	currentRequired := true
	currentOptional := false
	currentApprovals := 0

	for lineNum, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if m := sectionHeader.FindStringSubmatch(line); m != nil {
			optional := m[1] == "^"
			currentSection = m[2]
			currentOptional = optional
			currentRequired = !optional
			currentApprovals = 0
			if m[3] != "" {
				if n, err := strconv.Atoi(m[3]); err == nil {
					currentApprovals = n
				}
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		pattern := fields[0]
		owners := fields[1:]
		rules = append(rules, codeownersRule{
			LineNumber:        lineNum + 1,
			Section:           currentSection,
			Optional:          currentOptional,
			Required:          currentRequired,
			ApprovalsRequired: currentApprovals,
			Pattern:           pattern,
			Owners:            owners,
		})
	}
	return rules
}

// Compile-time guards: panics with a clear stack if the generated
// resource ever loses these fields (e.g. via a .lr rename) instead
// of producing an opaque nil-deref later.
var _ = plugin.StateIsSet
