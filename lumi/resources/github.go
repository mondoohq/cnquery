package resources

import (
	"context"
	"strconv"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-github/v43/github"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/transports"
	gh_transport "go.mondoo.io/mondoo/motor/transports/github"
)

func githubtransport(t transports.Transport) (*gh_transport.Transport, error) {
	gt, ok := t.(*gh_transport.Transport)
	if !ok {
		return nil, errors.New("github resource is not supported on this transport")
	}
	return gt, nil
}

func githubTimestamp(ts *github.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	return &ts.Time
}

func (g *lumiGithubOrganization) id() (string, error) {
	return "github.organization", nil
}

func (g *lumiGithubOrganization) init(args *lumi.Args) (*lumi.Args, GithubOrganization, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	gt, err := githubtransport(g.Runtime.Motor.Transport)
	if err != nil {
		return nil, nil, err
	}

	org, err := gt.Organization()
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = toInt64(org.ID)
	(*args)["name"] = toString(org.Name)
	(*args)["login"] = toString(org.Login)
	(*args)["node_id"] = toString(org.NodeID)
	(*args)["company"] = toString(org.Company)
	(*args)["blog"] = toString(org.Blog)
	(*args)["location"] = toString(org.Location)
	(*args)["email"] = toString(org.Email)
	(*args)["twitter_username"] = toString(org.TwitterUsername)
	(*args)["description"] = toString(org.Description)
	(*args)["created_at"] = org.CreatedAt
	(*args)["updated_at"] = org.UpdatedAt
	(*args)["total_private_repos"] = toInt(org.TotalPrivateRepos)
	(*args)["owned_private_repos"] = toInt(org.OwnedPrivateRepos)
	(*args)["private_gists"] = toInt(org.PrivateGists)
	(*args)["disk_usage"] = toInt(org.DiskUsage)
	(*args)["collaborators"] = toInt(org.Collaborators)
	(*args)["billing_email"] = toString(org.BillingEmail)

	plan, _ := jsonToDict(org.Plan)
	(*args)["plan"] = plan

	(*args)["two_factor_requirement_enabled"] = toBool(org.TwoFactorRequirementEnabled)
	(*args)["is_verified"] = toBool(org.IsVerified)

	(*args)["default_repository_permission"] = toString(org.DefaultRepoPermission)
	(*args)["members_can_create_repositories"] = toBool(org.MembersCanCreateRepos)
	(*args)["members_can_create_public_repositories"] = toBool(org.MembersCanCreatePublicRepos)
	(*args)["members_can_create_private_repositories"] = toBool(org.MembersCanCreatePrivateRepos)
	(*args)["members_can_create_internal_repositories"] = toBool(org.MembersCanCreateInternalRepos)
	(*args)["members_can_create_pages"] = toBool(org.MembersCanCreatePages)
	(*args)["members_can_create_public_pages"] = toBool(org.MembersCanCreatePublicPages)
	(*args)["members_can_create_private_pages"] = toBool(org.MembersCanCreatePrivateRepos)

	return args, nil, nil
}

func (g *lumiGithubOrganization) GetMembers() ([]interface{}, error) {
	gt, err := githubtransport(g.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	orgLogin, err := g.Login()
	if err != nil {
		return nil, err
	}

	members, _, err := gt.Client().Organizations.ListMembers(context.Background(), orgLogin, nil)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range members {
		member := members[i]

		var id int64
		if member.ID != nil {
			id = *member.ID
		}

		r, err := g.Runtime.CreateResource("github.user",
			"id", id,
			"login", toString(member.Login),
			"name", toString(member.Name),
			"email", toString(member.Email),
			"bio", toString(member.Bio),
			"createdAt", githubTimestamp(member.CreatedAt),
			"updatedAt", githubTimestamp(member.UpdatedAt),
			"suspendedAt", githubTimestamp(member.SuspendedAt),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *lumiGithubOrganization) GetOwners() ([]interface{}, error) {
	gt, err := githubtransport(g.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	orgLogin, err := g.Login()
	if err != nil {
		return nil, err
	}

	members, _, err := gt.Client().Organizations.ListMembers(context.Background(), orgLogin, &github.ListMembersOptions{
		Role: "admin",
	})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range members {
		member := members[i]

		var id int64
		if member.ID != nil {
			id = *member.ID
		}

		r, err := g.Runtime.CreateResource("github.user",
			"id", id,
			"login", toString(member.Login),
			"name", toString(member.Name),
			"email", toString(member.Email),
			"bio", toString(member.Bio),
			"createdAt", githubTimestamp(member.CreatedAt),
			"updatedAt", githubTimestamp(member.UpdatedAt),
			"suspendedAt", githubTimestamp(member.SuspendedAt),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *lumiGithubOrganization) GetRepositories() ([]interface{}, error) {
	gt, err := githubtransport(g.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	orgLogin, err := g.Login()
	if err != nil {
		return nil, err
	}

	repos, _, err := gt.Client().Repositories.ListByOrg(context.Background(), orgLogin, &github.RepositoryListByOrgOptions{Type: "all"})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range repos {
		repo := repos[i]

		var id int64
		if repo.ID != nil {
			id = *repo.ID
		}

		r, err := g.Runtime.CreateResource("github.repository",
			"id", id,
			"name", toString(repo.Name),
			"fullName", toString(repo.FullName),
			"description", toString(repo.Description),
			"homepage", toString(repo.Homepage),
			"createdAt", githubTimestamp(repo.CreatedAt),
			"updatedAt", githubTimestamp(repo.UpdatedAt),
			"archived", toBool(repo.Archived),
			"disabled", toBool(repo.Disabled),
			"private", toBool(repo.Private),
			"visibility", toString(repo.Visibility),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *lumiGithubOrganization) GetInstallations() ([]interface{}, error) {
	gt, err := githubtransport(g.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	orgLogin, err := g.Login()
	if err != nil {
		return nil, err
	}

	apps, _, err := gt.Client().Organizations.ListInstallations(context.Background(), orgLogin, &github.ListOptions{})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range apps.Installations {
		app := apps.Installations[i]

		var id int64
		if app.ID != nil {
			id = *app.ID
		}

		r, err := g.Runtime.CreateResource("github.installation",
			"id", id,
			"appId", toInt64(app.AppID),
			"appSlug", toString(app.AppSlug),
			"createdAt", githubTimestamp(app.CreatedAt),
			"updatedAt", githubTimestamp(app.UpdatedAt),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *lumiGithubUser) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}

func (g *lumiGithubRepository) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}

func (g *lumiGithubInstallation) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}
