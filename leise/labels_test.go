package leise

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/llx"
)

func label(t *testing.T, s string, f func(res *llx.Labels)) {
	res, err := Compile(s, llx.DefaultRegistry.Schema())
	assert.Nil(t, err)
	assert.NotNil(t, res)
	if res == nil {
		return
	}

	assert.NotNil(t, res.Labels)
	if res.Labels == nil {
		return
	}

	t.Run(s, func(t *testing.T) { f(res.Labels) })
}

func TestLabels(t *testing.T) {
	tests := []struct {
		src    string
		labels *llx.Labels
	}{
		{"true",
			&llx.Labels{Labels: map[int32]string{
				1: "",
			}}},
		{"1",
			&llx.Labels{Labels: map[int32]string{
				1: "",
			}}},
		{"1.23",
			&llx.Labels{Labels: map[int32]string{
				1: "",
			}}},
		{"\"string\"",
			&llx.Labels{Labels: map[int32]string{
				1: "",
			}}},
		{"sshd",
			&llx.Labels{Labels: map[int32]string{
				1: "sshd",
			}}},
		{"sshd.config",
			&llx.Labels{Labels: map[int32]string{
				1: "sshd.config",
			}}},
		{"sshd.config.params",
			&llx.Labels{Labels: map[int32]string{
				2: "sshd.config.params",
			}}},
		{"sshd.config(\"/my/path\").params",
			&llx.Labels{Labels: map[int32]string{
				2: "sshd.config.params",
			}}},
		{"platform.name platform.release",
			&llx.Labels{Labels: map[int32]string{
				2: "platform.name",
				4: "platform.release",
			}}},
		{"platform { name release }",
			&llx.Labels{
				Labels: map[int32]string{
					2: "platform",
				},
				Functions: map[int32]*llx.Labels{
					2: &llx.Labels{Labels: map[int32]string{
						2: "name",
						3: "release",
					}},
				},
			}},
		{"users.list { uid }",
			&llx.Labels{
				Labels: map[int32]string{
					3: "users.list",
				},
				Functions: map[int32]*llx.Labels{
					3: &llx.Labels{Labels: map[int32]string{
						2: "uid",
					}},
				},
			}},

		{"users.list[0]",
			&llx.Labels{
				Labels: map[int32]string{
					3: "users.list[0]",
				},
			}},

		{"users.list[0] { uid }",
			&llx.Labels{
				Labels: map[int32]string{
					4: "users.list[0]",
				},
				Functions: map[int32]*llx.Labels{
					4: &llx.Labels{Labels: map[int32]string{
						2: "uid",
					}},
				},
			}},

		{"sshd.config.params[\"UsePAM\"]",
			&llx.Labels{
				Labels: map[int32]string{
					3: "sshd.config.params[UsePAM]",
				},
			}},

		// 	// FIXME: failes compilation right now vv
		// {"sshd.config { file { path } }",
		// 	&llx.Labels{
		// 		Labels: map[int32]string{
		// 			2: "sshd.config",
		// 		},
		// 		Functions: map[int32]*llx.Labels{
		// 			2: &llx.Labels{
		// 				Labels: map[int32]string{
		// 					3: "file",
		// 				},
		// 				Functions: map[int32]*llx.Labels{
		// 					3: &llx.Labels{Labels: map[int32]string{2: "path"}},
		// 				}},
		// 		},
		// 	}},
	}

	for i := range tests {
		test := tests[i]
		label(t, test.src, func(labels *llx.Labels) {
			assert.Equal(t, test.labels, labels)
		})
	}
}
