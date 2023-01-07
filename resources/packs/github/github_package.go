package github

import "strconv"

func (g *mqlGithubPackage) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "github.package/" + strconv.FormatInt(id, 10), nil
}
