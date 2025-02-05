// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"strings"

	"github.com/google/go-github/v68/github"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/github/connection"
)

const (
	GITHUB_PACKAGE_VISIBILITY_PUBLIC   = "public"
	GITHUB_PACKAGE_VISIBILITY_PRIVATE  = "private"
	GITHUB_PACKAGE_VISIBILITY_INTERNAL = "internal"
)

func (g *mqlGithubPackage) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "github.package/" + strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubPackages) id() (string, error) {
	return "github.packages", nil
}

func (g *mqlGithubPackages) list() ([]any, error) {
	return newMqlGithubPackages(g.MqlRuntime, nil)
}

func (g *mqlGithubPackages) public() ([]any, error) {
	visibility := GITHUB_PACKAGE_VISIBILITY_PUBLIC
	return newMqlGithubPackages(g.MqlRuntime, &visibility)
}

func (g *mqlGithubPackages) private() ([]any, error) {
	visibility := GITHUB_PACKAGE_VISIBILITY_PRIVATE
	return newMqlGithubPackages(g.MqlRuntime, &visibility)
}

func (g *mqlGithubPackages) internal() ([]any, error) {
	visibility := GITHUB_PACKAGE_VISIBILITY_INTERNAL
	return newMqlGithubPackages(g.MqlRuntime, &visibility)
}

func newMqlGithubPackages(runtime *plugin.Runtime, visibility *string) ([]any, error) {
	conn := runtime.Connection.(*connection.GithubConnection)
	orgId, err := conn.Organization()
	if err != nil {
		return nil, err
	}

	pkgTypes := []string{"npm", "maven", "rubygems", "docker", "nuget", "container"}
	res := []any{}
	for i := range pkgTypes {
		listOpts := &github.PackageListOptions{
			ListOptions: github.ListOptions{PerPage: paginationPerPage},
			PackageType: github.String(pkgTypes[i]),
			Visibility:  visibility,
		}

		var allPackages []*github.Package
		for {
			packages, resp, err := conn.Client().Organizations.ListPackages(conn.Context(), orgId.Name, listOpts)
			if err != nil {
				if strings.Contains(err.Error(), "404") {
					return nil, nil
				}
				return nil, err
			}
			allPackages = append(allPackages, packages...)
			if resp.NextPage == 0 {
				break
			}
			listOpts.Page = resp.NextPage
		}

		for i := range allPackages {
			p := allPackages[i]

			owner, err := NewResource(runtime, "github.user", map[string]*llx.RawData{
				"id":    llx.IntData(p.GetOwner().GetID()),
				"login": llx.StringData(p.GetOwner().GetLogin()),
			})
			if err != nil {
				return nil, err
			}

			mqlGhPackage, err := CreateResource(runtime, "github.package", map[string]*llx.RawData{
				"id":           llx.IntDataPtr(p.ID),
				"name":         llx.StringDataPtr(p.Name),
				"packageType":  llx.StringDataPtr(p.PackageType),
				"owner":        llx.ResourceData(owner, owner.MqlName()),
				"createdAt":    llx.TimeDataPtr(githubTimestamp(p.CreatedAt)),
				"updatedAt":    llx.TimeDataPtr(githubTimestamp(p.UpdatedAt)),
				"versionCount": llx.IntDataPtr(p.VersionCount),
				"visibility":   llx.StringDataPtr(p.Visibility),
			})
			if err != nil {
				return nil, err
			}
			pkg := mqlGhPackage.(*mqlGithubPackage)

			// NOTE: we need to fetch repo separately because the Github repo object is not complete, instead of
			// call the repo fetching all the time, we make this lazy loading
			if p.Repository != nil && p.Repository.Name != nil {
				pkg.packageRepository = convert.ToString(p.Repository.Name)
			}
			res = append(res, pkg)
		}
	}

	return res, nil
}
