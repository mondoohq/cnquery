package resources

import (
	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/github"
)

func githubtransport(t transports.Transport) (*github.Transport, error) {
	gt, ok := t.(*github.Transport)
	if !ok {
		return nil, errors.New("github resource is not supported on this transport")
	}
	return gt, nil
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
