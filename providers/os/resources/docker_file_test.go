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
	cases := []struct {
		purpose           string
		subjectDockerFile string

		expectedLabels          map[string]interface{}
		expectedEnv             map[string]interface{}
		expectedFromImage       string
		expectedFromTag         string
		expectedUser            plugin.TValue[*mqlDockerFileUser]
		expectedCmd             plugin.TValue[*mqlDockerFileRun]
		expectedEntrypoint      plugin.TValue[*mqlDockerFileRun]
		expectedRunStruct       []plugin.TValue[*mqlDockerFileRun]
		expectedCopyStruct      []plugin.TValue[*mqlDockerFileCopy]
		expectedAddStruct       []plugin.TValue[*mqlDockerFileAdd]
		expectedExposeStructArr []plugin.TValue[*mqlDockerFileExpose]
	}{
		{
			purpose: "minimal instructions with CMD",
			subjectDockerFile: `
FROM alpine
CMD ["/bin/sh", "-c", "echo 'Hola'"]
`,
			expectedLabels:    map[string]interface{}{},
			expectedEnv:       map[string]interface{}{},
			expectedFromImage: "alpine",
			expectedCmd: plugin.TValue[*mqlDockerFileRun]{
				Data: &mqlDockerFileRun{
					Script: plugin.TValue[string]{Data: "/bin/sh\n-c\necho 'Hola'"},
				},
			},
		},
		{
			purpose: "without CMD but with ENTRYPOINT",
			subjectDockerFile: `
FROM debian:stable
ENTRYPOINT ["/usr/sbin/apache2ctl", "-D", "FOREGROUND"]
`,
			expectedLabels:    map[string]interface{}{},
			expectedEnv:       map[string]interface{}{},
			expectedFromImage: "debian",
			expectedFromTag:   "stable",
			expectedEntrypoint: plugin.TValue[*mqlDockerFileRun]{
				Data: &mqlDockerFileRun{
					Script: plugin.TValue[string]{Data: "/usr/sbin/apache2ctl\n-D\nFOREGROUND"},
				},
			},
		},
		{
			purpose: "with all instructions",
			subjectDockerFile: `
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
`,
			expectedLabels: map[string]interface{}{
				"a": "b",
				"c": "d",
			},
			expectedEnv: map[string]interface{}{
				"foo": "bar",
			},
			expectedFromImage: "alpine",
			expectedFromTag:   "3.14",
			expectedUser: plugin.TValue[*mqlDockerFileUser]{
				Data: &mqlDockerFileUser{
					User:  plugin.TValue[string]{Data: "1001"},
					Group: plugin.TValue[string]{Data: "1001"},
				},
			},
			expectedEntrypoint: plugin.TValue[*mqlDockerFileRun]{
				Data: &mqlDockerFileRun{
					Script: plugin.TValue[string]{Data: "sh"},
				},
			},
			expectedCmd: plugin.TValue[*mqlDockerFileRun]{
				Data: &mqlDockerFileRun{
					Script: plugin.TValue[string]{Data: "curl\nhttp://example.com"},
				},
			},
			expectedCopyStruct: []plugin.TValue[*mqlDockerFileCopy]{
				{Data: &mqlDockerFileCopy{
					Src: plugin.TValue[[]interface{}]{
						Data: []interface{}{"/foo"}},
					Dst: plugin.TValue[string]{
						Data: "/bar"},
				}},
			},
			expectedRunStruct: []plugin.TValue[*mqlDockerFileRun]{
				{Data: &mqlDockerFileRun{
					Script: plugin.TValue[string]{
						Data: "apk add --no-cache curl"},
				}},
			},
			expectedAddStruct: []plugin.TValue[*mqlDockerFileAdd]{
				{Data: &mqlDockerFileAdd{
					Src: plugin.TValue[[]interface{}]{
						Data: []interface{}{"/foo-add"}},
					Dst: plugin.TValue[string]{
						Data: "/bar-add"},
				}},
			},
			expectedExposeStructArr: []plugin.TValue[*mqlDockerFileExpose]{
				{Data: &mqlDockerFileExpose{
					Port:     plugin.TValue[int64]{Data: int64(80)},
					Protocol: plugin.TValue[string]{Data: "udp"},
				}},
				{Data: &mqlDockerFileExpose{
					Port:     plugin.TValue[int64]{Data: int64(8080)},
					Protocol: plugin.TValue[string]{Data: "tcp"}, // this is the default
				}},
			},
		},
	}

	for _, kase := range cases {
		t.Run(kase.purpose, func(t *testing.T) {
			r := &plugin.Runtime{Resources: &syncx.Map[plugin.Resource]{}}

			file := &mqlFile{
				Content:    plugin.TValue[string]{Data: kase.subjectDockerFile, State: plugin.StateIsSet},
				Path:       plugin.TValue[string]{Data: "Dockerfile", State: plugin.StateIsSet},
				MqlRuntime: r,
			}
			dockerFile := mqlDockerFile{
				File:       plugin.TValue[*mqlFile]{Data: file, State: plugin.StateIsSet},
				MqlRuntime: r,
			}

			require.NoError(t, dockerFile.parse(file))

			actualMqlDockerFileStage := dockerFile.Stages.Data[0].(*mqlDockerFileStage)

			require.Equal(t, kase.expectedLabels, actualMqlDockerFileStage.Labels.Data)
			require.Equal(t, kase.expectedEnv, actualMqlDockerFileStage.Env.Data)
			require.Equal(t, kase.expectedFromImage, actualMqlDockerFileStage.From.Data.Image.Data)
			require.Equal(t, kase.expectedFromTag, actualMqlDockerFileStage.From.Data.Tag.Data)

			if kase.expectedCmd.Data == nil {
				require.Nil(t, actualMqlDockerFileStage.Cmd.Data)
			} else {
				require.Equal(t, kase.expectedCmd.Data.Script.Data, actualMqlDockerFileStage.Cmd.Data.Script.Data)
			}

			if kase.expectedUser.Data == nil {
				require.Nil(t, actualMqlDockerFileStage.User.Data)
			} else {
				require.Equal(t, kase.expectedUser.Data.User.Data, actualMqlDockerFileStage.User.Data.User.Data)
				require.Equal(t, kase.expectedUser.Data.Group.Data, actualMqlDockerFileStage.User.Data.Group.Data)
			}

			if kase.expectedEntrypoint.Data == nil {
				require.Nil(t, actualMqlDockerFileStage.Entrypoint.Data)
			} else {
				require.Equal(t, kase.expectedEntrypoint.Data.Script.Data, actualMqlDockerFileStage.Entrypoint.Data.Script.Data)
			}

			require.Equal(t, len(kase.expectedCopyStruct), len(actualMqlDockerFileStage.Copy.Data))
			for i, cpy := range actualMqlDockerFileStage.Copy.Data {
				actualCopy := cpy.(*mqlDockerFileCopy)
				require.Equal(t, kase.expectedCopyStruct[i].Data.Src.Data, actualCopy.Src.Data)
				require.Equal(t, kase.expectedCopyStruct[i].Data.Dst.Data, actualCopy.Dst.Data)
			}

			require.Equal(t, len(kase.expectedRunStruct), len(actualMqlDockerFileStage.Run.Data))
			for i, run := range actualMqlDockerFileStage.Run.Data {
				actualRun := run.(*mqlDockerFileRun)
				require.Equal(t, kase.expectedRunStruct[i].Data.Script.Data, actualRun.Script.Data)
			}

			require.Equal(t, len(kase.expectedAddStruct), len(actualMqlDockerFileStage.Add.Data))
			for i, cpy := range actualMqlDockerFileStage.Add.Data {
				actualAdd := cpy.(*mqlDockerFileAdd)
				require.Equal(t, kase.expectedAddStruct[i].Data.Src.Data, actualAdd.Src.Data)
				require.Equal(t, kase.expectedAddStruct[i].Data.Dst.Data, actualAdd.Dst.Data)
			}

			require.Equal(t, len(kase.expectedExposeStructArr), len(actualMqlDockerFileStage.Expose.Data))
			for i, expose := range actualMqlDockerFileStage.Expose.Data {
				actualExpose := expose.(*mqlDockerFileExpose)
				require.Equal(t, kase.expectedExposeStructArr[i].Data.Port.Data, actualExpose.Port.Data)
				require.Equal(t, kase.expectedExposeStructArr[i].Data.Protocol.Data, actualExpose.Protocol.Data)
			}
		})
	}
}
