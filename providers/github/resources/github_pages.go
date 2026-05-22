// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/github/connection"
	"go.mondoo.com/mql/v13/types"
)

func (g *mqlGithubRepositoryPages) id() (string, error) {
	return g.__id, nil
}

func initGithubRepositoryPages(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if _, ok := args["__id"]; ok {
		return args, nil, nil
	}
	repo, err := NewResource(runtime, "github.repository", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	pages := repo.(*mqlGithubRepository).GetPages()
	if pages.Error != nil {
		return nil, nil, pages.Error
	}
	if pages.Data == nil {
		return nil, nil, errors.New("GitHub Pages is not enabled for this repository")
	}
	return args, pages.Data, nil
}

func (g *mqlGithubRepository) pages() (*mqlGithubRepositoryPages, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data
	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data
	if owner.Login.Error != nil {
		return nil, owner.Login.Error
	}
	ownerLogin := owner.Login.Data

	pages, _, err := conn.Client().Repositories.GetPagesInfo(conn.Context(), ownerLogin, repoName)
	if err != nil {
		switch githubResponseStatus(err) {
		case 404:
			// Pages not enabled for this repository
			g.Pages.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		case 403, 401:
			log.Debug().Err(err).Msg("GitHub Pages configuration not accessible")
			g.Pages.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	if pages == nil {
		g.Pages.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	args := map[string]*llx.RawData{
		"__id":          llx.StringData("github.repository.pages/" + ownerLogin + "/" + repoName),
		"url":           llx.StringDataPtr(pages.URL),
		"htmlUrl":       llx.StringDataPtr(pages.HTMLURL),
		"status":        llx.StringDataPtr(pages.Status),
		"cname":         llx.StringDataPtr(pages.CNAME),
		"custom404":     llx.BoolData(pages.GetCustom404()),
		"buildType":     llx.StringDataPtr(pages.BuildType),
		"public":        llx.BoolData(pages.GetPublic()),
		"httpsEnforced": llx.BoolData(pages.GetHTTPSEnforced()),
	}

	if cert := pages.HTTPSCertificate; cert != nil {
		args["httpsCertificateState"] = llx.StringDataPtr(cert.State)
		args["httpsCertificateDescription"] = llx.StringDataPtr(cert.Description)
		args["httpsCertificateDomains"] = llx.ArrayData(convert.SliceAnyToInterface[string](cert.Domains), types.String)
		args["httpsCertificateExpiresAt"] = llx.StringDataPtr(cert.ExpiresAt)
	} else {
		args["httpsCertificateState"] = llx.StringData("")
		args["httpsCertificateDescription"] = llx.StringData("")
		args["httpsCertificateDomains"] = llx.ArrayData([]any{}, types.String)
		args["httpsCertificateExpiresAt"] = llx.StringData("")
	}

	if src := pages.Source; src != nil {
		args["sourceBranch"] = llx.StringDataPtr(src.Branch)
		args["sourcePath"] = llx.StringDataPtr(src.Path)
	} else {
		args["sourceBranch"] = llx.StringData("")
		args["sourcePath"] = llx.StringData("")
	}

	res, err := CreateResource(g.MqlRuntime, "github.repository.pages", args)
	if err != nil {
		return nil, err
	}
	return res.(*mqlGithubRepositoryPages), nil
}
