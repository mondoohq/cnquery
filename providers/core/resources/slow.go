package resources

import (
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/util/convert"
	"time"
)

func (u *mqlSlow) id() (string, error) {
	return "slow", nil
}

func (u *mqlSlow) field() (string, error) {
	time.Sleep(30 * time.Second)
	return "field", nil
}

func (u *mqlSlow) field2() (string, error) {
	time.Sleep(60 * time.Second)
	return "field", nil
}

func (u *mqlSlow) list() ([]interface{}, error) {
	time.Sleep(240 * time.Second)
	return convert.SliceAnyToInterface([]string{"a", "b", "b", "d"}), nil
}
