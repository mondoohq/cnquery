package resources

import (
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go/transport/http"
	"go.mondoo.com/cnquery/providers/aws/connection"
)

func (a *mqlAws) regions() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	regions, err := conn.Regions()
	for i := range regions {
		res = append(res, regions[i])
	}
	return res, err
}

func Is400AccessDeniedError(err error) bool {
	var respErr *http.ResponseError
	if errors.As(err, &respErr) {
		if respErr.HTTPStatusCode() == 400 && strings.Contains(respErr.Error(), "AccessDeniedException") {
			return true
		}
	}
	return false
}

func Ec2TagsToMap(tags []types.Tag) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[toString(tag.Key)] = toString(tag.Value)
		}
	}

	return tagsMap
}

func toBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func toString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func strMapToInterface(m map[string]string) map[string]interface{} {
	res := map[string]interface{}{}
	for k, v := range m {
		res[k] = v
	}
	return res
}
