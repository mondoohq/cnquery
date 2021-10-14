package gitlab

import (
	"strconv"
)

func (t *Transport) Identifier() (string, error) {
	grp, _, err := t.Client().Groups.GetGroup(t.GroupPath, nil)
	if err != nil {
		return "", err
	}

	return "//platformid.api.mondoo.app/runtime/gitlab/group/" + strconv.Itoa(grp.ID), nil
}
