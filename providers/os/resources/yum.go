package resources

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
	"go.mondoo.com/cnquery/providers/os/resources/yum"
	"go.mondoo.com/cnquery/types"
	"go.mondoo.com/cnquery/utils/stringx"
)

var supportedPlatforms = []string{"amazonlinux"}

func (y *mqlYum) id() (string, error) {
	return "yum", nil
}

func (y *mqlYum) repos() ([]interface{}, error) {
	conn := y.MqlRuntime.Connection.(shared.Connection)
	platform := conn.Asset().Platform

	if !platform.IsFamily("redhat") && !stringx.Contains(supportedPlatforms, platform.Name) {
		return nil, errors.New("yum.repos is only supported on redhat-based platforms")
	}

	o, err := CreateResource(y.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("yum -v repolist all"),
	})
	cmd := o.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return nil, errors.New("could not retrieve yum repo list")
	}

	repos, err := yum.ParseRepos(strings.NewReader(cmd.Stdout.Data))
	if err != nil {
		return nil, err
	}

	mqlRepos := make([]interface{}, len(repos))
	for i, repo := range repos {
		f, err := CreateResource(y.MqlRuntime, "file", map[string]*llx.RawData{
			"path": llx.StringData(repo.Filename),
		})
		if err != nil {
			return nil, err
		}

		mqlRepo, err := CreateResource(y.MqlRuntime, "yum.repo", map[string]*llx.RawData{
			"id":       llx.StringData(repo.Id),
			"name":     llx.StringData(repo.Name),
			"status":   llx.StringData(repo.Status),
			"baseurl":  llx.ArrayData(llx.TArr2Raw(repo.Baseurl), types.String),
			"expire":   llx.StringData(repo.Expire),
			"filename": llx.StringData(repo.Filename),
			"file":     llx.ResourceData(f, "file"),
			"revision": llx.StringData(repo.Revision),
			"pkgs":     llx.StringData(repo.Pkgs),
			"size":     llx.StringData(repo.Size),
			"mirrors":  llx.StringData(repo.Mirrors),
		})
		if err != nil {
			return nil, err
		}
		mqlRepos[i] = mqlRepo
	}

	return mqlRepos, nil
}

var rhel67release = regexp.MustCompile(`^[6|7].*$`)

func (y *mqlYum) vars() (map[string]interface{}, error) {
	conn := y.MqlRuntime.Connection.(shared.Connection)
	platform := conn.Asset().Platform

	if !platform.IsFamily("redhat") && !stringx.Contains(supportedPlatforms, platform.Name) {
		return nil, errors.New("yum.vars is only supported on redhat-based platforms")
	}

	// use dnf script as default
	script := fmt.Sprintf(yum.DnfVarsCommand, yum.PythonRhel)
	if !platform.IsFamily("redhat") {
		// eg. amazon linux does not ship with /usr/libexec/platform-python
		script = fmt.Sprintf(yum.DnfVarsCommand, yum.Python3)
	}

	// fallback for older versions like 6 and 7 version to use yum script
	if rhel67release.MatchString(platform.Version) {
		script = yum.Rhel6VarsCommand
	}

	o, err := CreateResource(y.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData(script),
	})
	cmd := o.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return nil, errors.New("could not retrieve yum repo list")
	}

	vars, err := yum.ParseVariables(strings.NewReader(cmd.Stdout.Data))
	if err != nil {
		return nil, err
	}

	res := map[string]interface{}{}
	for k := range vars {
		res[k] = vars[k]
	}

	return res, nil
}

func (y *mqlYumRepo) id() (string, error) {
	return y.Id.Data, nil
}

func initYumRepo(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	nameRaw := args["id"]
	if nameRaw == nil {
		return args, nil, nil
	}

	name, ok := nameRaw.Value.(string)
	if !ok {
		return args, nil, nil
	}

	o, err := CreateResource(runtime, "yum", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	yumResource := o.(*mqlYum)

	repos := yumResource.GetRepos()
	if repos.Error != nil {
		return nil, nil, repos.Error
	}

	for i := range repos.Data {
		selected := repos.Data[i].(*mqlYumRepo)
		if selected.Id.Data == name {
			return nil, selected, nil
		}
	}

	// if the repo cannot be found we return an error
	return nil, nil, errors.New("could not find yum repo " + name)
}

func (y *mqlYumRepo) enabled() (bool, error) {
	status := y.GetStatus()
	if status.Error != nil {
		return false, status.Error
	}

	return strings.ToLower(status.Data) == "enabled", nil
}
