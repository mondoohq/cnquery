// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
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

// -----------------------------------------------------------------------------
// Vulnerabilities — surfaced from GitLab's security report scanners (SAST,
// DAST, dependency, secret, container). Ultimate-tier feature; on lower
// tiers we degrade to an empty list rather than failing the resource graph,
// mirroring the auditEvents() pattern.
//
// The REST surface (ProjectVulnerabilitiesService) is fully deprecated in
// the SDK; the GraphQL surface returns the full typed Vulnerability node
// including location, identifiers, and scanner. The provider's existing
// gitlab client already exposes GraphQL via Client().GraphQL.Do.
// -----------------------------------------------------------------------------

func (v *mqlGitlabProjectVulnerability) id() (string, error) {
	return "gitlab.project.vulnerability/" + v.Id.Data, nil
}

func (s *mqlGitlabProjectVulnerabilityScanner) id() (string, error) {
	return "gitlab.project.vulnerability.scanner/" + s.Id.Data, nil
}

type mqlGitlabProjectVulnerabilityInternal struct {
	scannerData   *gqlVulnScanner
	projectID     int64
	confirmedByID int64
	resolvedByID  int64
	dismissedByID int64
}

type gqlVulnUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type gqlVulnScanner struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Vendor     string `json:"vendor"`
	Version    string `json:"version"`
	ExternalID string `json:"externalId"`
}

type gqlVulnIdentifier struct {
	ExternalType string `json:"externalType"`
	ExternalID   string `json:"externalId"`
	Name         string `json:"name"`
	URL          string `json:"url"`
}

type gqlVulnProject struct {
	ID       string `json:"id"`
	FullPath string `json:"fullPath"`
}

type gqlVulnerability struct {
	ID                      string              `json:"id"`
	Title                   string              `json:"title"`
	Description             string              `json:"description"`
	Severity                string              `json:"severity"`
	State                   string              `json:"state"`
	ReportType              string              `json:"reportType"`
	DetectedAt              *time.Time          `json:"detectedAt"`
	ConfirmedAt             *time.Time          `json:"confirmedAt"`
	ResolvedAt              *time.Time          `json:"resolvedAt"`
	DismissedAt             *time.Time          `json:"dismissedAt"`
	DismissalReason         string              `json:"dismissalReason"`
	ResolvedOnDefaultBranch bool                `json:"resolvedOnDefaultBranch"`
	WebURL                  string              `json:"webUrl"`
	Location                map[string]any      `json:"location"`
	Identifiers             []gqlVulnIdentifier `json:"identifiers"`
	Scanner                 *gqlVulnScanner     `json:"scanner"`
	Project                 *gqlVulnProject     `json:"project"`
	ConfirmedBy             *gqlVulnUser        `json:"confirmedBy"`
	ResolvedBy              *gqlVulnUser        `json:"resolvedBy"`
	DismissedBy             *gqlVulnUser        `json:"dismissedBy"`
}

type gqlPageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type gqlVulnConnection struct {
	PageInfo gqlPageInfo        `json:"pageInfo"`
	Nodes    []gqlVulnerability `json:"nodes"`
}

type gqlGraphQLError struct {
	Message string `json:"message"`
}

type gqlProjectVulnResponse struct {
	Data struct {
		Project *struct {
			Vulnerabilities gqlVulnConnection `json:"vulnerabilities"`
		} `json:"project"`
	} `json:"data"`
	Errors []gqlGraphQLError `json:"errors"`
}

type gqlGroupVulnResponse struct {
	Data struct {
		Group *struct {
			Vulnerabilities gqlVulnConnection `json:"vulnerabilities"`
		} `json:"group"`
	} `json:"data"`
	Errors []gqlGraphQLError `json:"errors"`
}

type gqlSeverityCountsResponse struct {
	Data struct {
		Project *struct {
			VulnerabilitySeveritiesCount *struct {
				Critical int64 `json:"critical"`
				High     int64 `json:"high"`
				Medium   int64 `json:"medium"`
				Low      int64 `json:"low"`
				Info     int64 `json:"info"`
				Unknown  int64 `json:"unknown"`
			} `json:"vulnerabilitySeveritiesCount"`
		} `json:"project"`
	} `json:"data"`
	Errors []gqlGraphQLError `json:"errors"`
}

const vulnerabilityFields = `
  id
  title
  description
  severity
  state
  reportType
  detectedAt
  confirmedAt
  resolvedAt
  dismissedAt
  dismissalReason
  resolvedOnDefaultBranch
  webUrl
  identifiers { externalType externalId name url }
  scanner { id name vendor version externalId }
  project { id fullPath }
  confirmedBy { id username }
  resolvedBy { id username }
  dismissedBy { id username }
  location {
    __typename
    ... on VulnerabilityLocationSast {
      file startLine endLine blobPath vulnerableClass vulnerableMethod
    }
    ... on VulnerabilityLocationDependencyScanning {
      file dependency { package { name } version }
    }
    ... on VulnerabilityLocationContainerScanning {
      image operatingSystem dependency { package { name } version }
    }
    ... on VulnerabilityLocationDast {
      hostname path requestMethod param
    }
    ... on VulnerabilityLocationSecretDetection {
      file startLine endLine blobPath vulnerableClass vulnerableMethod
    }
  }
`

// isVulnerabilitiesUnavailable returns true when the GraphQL errors indicate the
// vulnerability API is gated off — i.e. the project/group exists but the caller's
// tier or token doesn't grant access. Surfaces as an empty list instead of an error.
func isVulnerabilitiesUnavailable(errs []gqlGraphQLError) bool {
	if len(errs) == 0 {
		return false
	}
	for _, e := range errs {
		m := strings.ToLower(e.Message)
		if strings.Contains(m, "permission") ||
			strings.Contains(m, "not available") ||
			strings.Contains(m, "doesn't exist on type") ||
			strings.Contains(m, "license") {
			continue
		}
		return false
	}
	return true
}

// parseGIDInt extracts the trailing integer from a GitLab GraphQL global id
// like "gid://gitlab/User/123". Returns 0 when the id is empty or unparseable.
func parseGIDInt(gid string) int64 {
	if gid == "" {
		return 0
	}
	idx := strings.LastIndex(gid, "/")
	if idx < 0 || idx >= len(gid)-1 {
		return 0
	}
	n, _ := strconv.ParseInt(gid[idx+1:], 10, 64)
	return n
}

func fetchVulnerabilities(conn *connection.GitLabConnection, scope, fullPath string) ([]gqlVulnerability, error) {
	var all []gqlVulnerability
	cursor := ""
	for {
		afterClause := ""
		if cursor != "" {
			afterClause = ", after: " + strconv.Quote(cursor)
		}
		query := fmt.Sprintf(`query { %s(fullPath: %s) { vulnerabilities(first: 100%s) { pageInfo { hasNextPage endCursor } nodes { %s } } } }`,
			scope, strconv.Quote(fullPath), afterClause, vulnerabilityFields)

		var page gqlVulnConnection
		var errs []gqlGraphQLError
		switch scope {
		case "project":
			var resp gqlProjectVulnResponse
			if _, err := conn.Client().GraphQL.Do(gitlab.GraphQLQuery{Query: query}, &resp); err != nil {
				return nil, err
			}
			errs = resp.Errors
			if resp.Data.Project != nil {
				page = resp.Data.Project.Vulnerabilities
			}
		case "group":
			var resp gqlGroupVulnResponse
			if _, err := conn.Client().GraphQL.Do(gitlab.GraphQLQuery{Query: query}, &resp); err != nil {
				return nil, err
			}
			errs = resp.Errors
			if resp.Data.Group != nil {
				page = resp.Data.Group.Vulnerabilities
			}
		default:
			return nil, fmt.Errorf("unsupported scope %q", scope)
		}
		if len(errs) > 0 {
			if isVulnerabilitiesUnavailable(errs) {
				return nil, nil
			}
			return nil, fmt.Errorf("gitlab vulnerabilities query failed: %s", errs[0].Message)
		}
		all = append(all, page.Nodes...)
		if !page.PageInfo.HasNextPage {
			break
		}
		cursor = page.PageInfo.EndCursor
	}
	return all, nil
}

func vulnsToMqlResources(runtime *plugin.Runtime, nodes []gqlVulnerability) ([]any, error) {
	out := make([]any, 0, len(nodes))
	for _, n := range nodes {
		identifiers := make([]any, 0, len(n.Identifiers))
		for _, id := range n.Identifiers {
			identifiers = append(identifiers, map[string]any{
				"externalType": id.ExternalType,
				"externalId":   id.ExternalID,
				"name":         id.Name,
				"url":          id.URL,
			})
		}

		location := n.Location
		if location == nil {
			location = map[string]any{}
		}
		delete(location, "__typename")

		args := map[string]*llx.RawData{
			"id":                      llx.StringData(n.ID),
			"title":                   llx.StringData(n.Title),
			"description":             llx.StringData(n.Description),
			"severity":                llx.StringData(n.Severity),
			"state":                   llx.StringData(n.State),
			"reportType":              llx.StringData(n.ReportType),
			"detectedAt":              llx.TimeDataPtr(n.DetectedAt),
			"confirmedAt":             llx.TimeDataPtr(n.ConfirmedAt),
			"resolvedAt":              llx.TimeDataPtr(n.ResolvedAt),
			"dismissedAt":             llx.TimeDataPtr(n.DismissedAt),
			"dismissalReason":         llx.StringData(n.DismissalReason),
			"resolvedOnDefaultBranch": llx.BoolData(n.ResolvedOnDefaultBranch),
			"webURL":                  llx.StringData(n.WebURL),
			"location":                llx.DictData(location),
			"identifiers":             llx.ArrayData(identifiers, types.Dict),
		}
		res, err := CreateResource(runtime, "gitlab.project.vulnerability", args)
		if err != nil {
			return nil, err
		}
		mqlV := res.(*mqlGitlabProjectVulnerability)
		if n.Scanner != nil {
			mqlV.scannerData = n.Scanner
		}
		if n.Project != nil {
			mqlV.projectID = parseGIDInt(n.Project.ID)
		}
		if n.ConfirmedBy != nil {
			mqlV.confirmedByID = parseGIDInt(n.ConfirmedBy.ID)
		}
		if n.ResolvedBy != nil {
			mqlV.resolvedByID = parseGIDInt(n.ResolvedBy.ID)
		}
		if n.DismissedBy != nil {
			mqlV.dismissedByID = parseGIDInt(n.DismissedBy.ID)
		}
		out = append(out, mqlV)
	}
	return out, nil
}

// vulnerabilities fetches confirmed vulnerabilities for the project.
// Requires GitLab Ultimate; lower tiers degrade to an empty list.
func (p *mqlGitlabProject) vulnerabilities() ([]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)
	fullPath := p.FullPath.Data
	if fullPath == "" {
		return []any{}, nil
	}
	nodes, err := fetchVulnerabilities(conn, "project", fullPath)
	if err != nil {
		return nil, err
	}
	return vulnsToMqlResources(p.MqlRuntime, nodes)
}

// vulnerabilities fetches confirmed vulnerabilities across all projects in
// the group and its subgroups. Requires GitLab Ultimate; lower tiers degrade
// to an empty list.
func (g *mqlGitlabGroup) vulnerabilities() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GitLabConnection)
	fullPath := g.FullPath.Data
	if fullPath == "" {
		return []any{}, nil
	}
	nodes, err := fetchVulnerabilities(conn, "group", fullPath)
	if err != nil {
		return nil, err
	}
	return vulnsToMqlResources(g.MqlRuntime, nodes)
}

// vulnerabilityCountsBySeverity returns the confirmed-vulnerability count
// for each severity bucket. One GraphQL roundtrip, no pagination.
func (p *mqlGitlabProject) vulnerabilityCountsBySeverity() (map[string]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)
	fullPath := p.FullPath.Data
	if fullPath == "" {
		return map[string]any{}, nil
	}
	query := fmt.Sprintf(`query { project(fullPath: %s) { vulnerabilitySeveritiesCount { critical high medium low info unknown } } }`,
		strconv.Quote(fullPath))
	var resp gqlSeverityCountsResponse
	if _, err := conn.Client().GraphQL.Do(gitlab.GraphQLQuery{Query: query}, &resp); err != nil {
		return nil, err
	}
	if isVulnerabilitiesUnavailable(resp.Errors) {
		return map[string]any{}, nil
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("gitlab vulnerability counts query failed: %s", resp.Errors[0].Message)
	}
	if resp.Data.Project == nil || resp.Data.Project.VulnerabilitySeveritiesCount == nil {
		return map[string]any{}, nil
	}
	c := resp.Data.Project.VulnerabilitySeveritiesCount
	return map[string]any{
		"CRITICAL": c.Critical,
		"HIGH":     c.High,
		"MEDIUM":   c.Medium,
		"LOW":      c.Low,
		"INFO":     c.Info,
		"UNKNOWN":  c.Unknown,
	}, nil
}

// scanner returns the scanner sub-resource built from cached creator data.
func (v *mqlGitlabProjectVulnerability) scanner() (*mqlGitlabProjectVulnerabilityScanner, error) {
	if v.scannerData == nil {
		v.Scanner.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := CreateResource(v.MqlRuntime, "gitlab.project.vulnerability.scanner", map[string]*llx.RawData{
		"id":         llx.StringData(v.scannerData.ID),
		"name":       llx.StringData(v.scannerData.Name),
		"vendor":     llx.StringData(v.scannerData.Vendor),
		"version":    llx.StringData(v.scannerData.Version),
		"externalId": llx.StringData(v.scannerData.ExternalID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGitlabProjectVulnerabilityScanner), nil
}

func (v *mqlGitlabProjectVulnerability) project() (*mqlGitlabProject, error) {
	if v.projectID <= 0 {
		v.Project.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(v.MqlRuntime, "gitlab.project", map[string]*llx.RawData{
		"id": llx.IntData(v.projectID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGitlabProject), nil
}

func (v *mqlGitlabProjectVulnerability) confirmedBy() (*mqlGitlabUser, error) {
	return vulnUserRef(v.MqlRuntime, v.confirmedByID, &v.ConfirmedBy.State)
}

func (v *mqlGitlabProjectVulnerability) resolvedBy() (*mqlGitlabUser, error) {
	return vulnUserRef(v.MqlRuntime, v.resolvedByID, &v.ResolvedBy.State)
}

func (v *mqlGitlabProjectVulnerability) dismissedBy() (*mqlGitlabUser, error) {
	return vulnUserRef(v.MqlRuntime, v.dismissedByID, &v.DismissedBy.State)
}

func vulnUserRef(runtime *plugin.Runtime, userID int64, state *plugin.State) (*mqlGitlabUser, error) {
	if userID <= 0 {
		*state = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(runtime, "gitlab.user", map[string]*llx.RawData{
		"id": llx.IntData(userID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGitlabUser), nil
}
