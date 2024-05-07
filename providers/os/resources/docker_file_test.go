// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/utils/syncx"
)

func TestParseDockerfile(t *testing.T) {
	dockerfile := `
FROM alpine:3.14
ENV foo=bar
LABEL a=b
RUN apk add --no-cache curl
LABEL c=d
USER 1001:1001
CMD ["curl", "http://example.com"]
ENTRYPOINT ["sh"]
EXPOSE 80/udp
EXPOSE 8080
COPY /foo /bar
ADD /foo-add /bar-add
`

	r := &plugin.Runtime{Resources: &syncx.Map[plugin.Resource]{}}

	file := &mqlFile{
		Content:    plugin.TValue[string]{Data: dockerfile, State: plugin.StateIsSet},
		Path:       plugin.TValue[string]{Data: "Dockerfile", State: plugin.StateIsSet},
		MqlRuntime: r,
	}
	dockerFile := mqlDockerFile{
		File:       plugin.TValue[*mqlFile]{Data: file, State: plugin.StateIsSet},
		MqlRuntime: r,
	}
	err := dockerFile.parse(file)

	require.NoError(t, err)

	s := dockerFile.Stages.Data[0].(*mqlDockerFileStage)
	expectedLabels := map[string]interface{}{
		"a": "b",
		"c": "d",
	}
	expectedEnv := map[string]interface{}{
		"foo": "bar",
	}
	require.Equal(t, "alpine", s.From.Data.Image.Data)
	require.Equal(t, "3.14", s.From.Data.Tag.Data)
	require.Equal(t, expectedLabels, s.Labels.Data)
	require.Equal(t, expectedEnv, s.Env.Data)

	copy := s.Copy.Data[0].(*mqlDockerFileCopy)
	require.Equal(t, []interface{}{"/foo"}, copy.Src.Data)
	require.Equal(t, "/bar", copy.Dst.Data)

	run := s.Run.Data[0].(*mqlDockerFileRun)
	require.Equal(t, "apk add --no-cache curl", run.Script.Data)

	require.Equal(t, "curl\nhttp://example.com", s.Cmd.Data.Script.Data)
	require.Equal(t, "sh", s.Entrypoint.Data.Script.Data)

	require.Equal(t, "1001", s.User.Data.User.Data)
	require.Equal(t, "1001", s.User.Data.Group.Data)

	exposes := []*mqlDockerFileExpose{
		s.Expose.Data[0].(*mqlDockerFileExpose),
		s.Expose.Data[1].(*mqlDockerFileExpose),
	}

	require.Equal(t, int64(80), exposes[0].Port.Data)
	require.Equal(t, "udp", exposes[0].Protocol.Data)
	require.Equal(t, int64(8080), exposes[1].Port.Data)
	// verify default protocol if not specified
	require.Equal(t, "tcp", exposes[1].Protocol.Data)

	add := s.Add.Data[0].(*mqlDockerFileAdd)
	require.Equal(t, []interface{}{"/foo-add"}, add.Src.Data)
	require.Equal(t, "/bar-add", add.Dst.Data)
}
