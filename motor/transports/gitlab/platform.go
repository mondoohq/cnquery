package gitlab

import (
	"strconv"

	"github.com/xanzy/go-gitlab"
)

func (t *Transport) Identifier() (string, error) {
	grp, err := t.Group()
	if err != nil {
		return "", err
	}

	return "//platformid.api.mondoo.app/runtime/gitlab/group/" + strconv.Itoa(grp.ID), nil
}

func (t *Transport) Group() (*gitlab.Group, error) {
	grp, _, err := t.Client().Groups.GetGroup(t.GroupPath, nil)
	if err != nil {
		return nil, err
	}
	return grp, err
}
