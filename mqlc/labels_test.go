// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mqlc_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/mqlc"
)

func label(t *testing.T, s string, f func(res *llx.Labels)) {
	res, err := mqlc.Compile(s, nil, conf)
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
		{
			"mondoo.version == 'yo'",
			&llx.Labels{
				Labels: map[string]string{
					"J084T/tsf/V2gVKuPUCMiaSli6jjgbrZfBlLtC06P3JdvDMMg2jLWvO8q8CXcZ4rtZf08GRZ5Qsgcoz/4Ph6Vw==": "mondoo.version == \"yo\"",
					"J4anmJ+mXJX380Qslh563U7Bs5d6fiD2ghVxV9knAU0iy/P+IVNZsDhBbCmbpJch3Tm0NliAMiaY47lmw887Jw==": "mondoo.version",
				},
			},
		},

		{
			"true",
			&llx.Labels{Labels: map[string]string{
				"13VXYfnMnc74H8XVgiMbH6ZSHxTGQxkhJfUkIiYOBCfUDxHAIJWopMcsea7hXkBTFpbM9lCDnbDBev1z+uagBw==": "",
			}},
		},
		{
			"1",
			&llx.Labels{Labels: map[string]string{
				"zcXMKiq4b4QGFVMCvyyFhLXFQKOYn7NKqbV/47XBrKcFwirRjWPgReFt4kdD9G7/ZZCJPsmS4pdCfM32VdTAiQ==": "",
			}},
		},
		{
			"1.23",
			&llx.Labels{Labels: map[string]string{
				"wYfzvA9Xuue3Dr0AcPOM9Y8yyd9t+DWggiInRLU5bSoWOoQtxVrt+aNkeOAorYDYV26ni1v6nIGzL6/3EqxSqQ==": "",
			}},
		},
		{
			"\"string\"",
			&llx.Labels{Labels: map[string]string{
				"YKg4KBZELSGbdx6hE2dqiH5YWTTjYQDYjVzgUsOxnZs9djRb3SHjCadjEsPq6KlmcRLwo9kpv2fPYEJoQJb2qw==": "",
			}},
		},
		{
			"sshd",
			&llx.Labels{Labels: map[string]string{
				"fAVT9TdeX6puAiM5lRS0Rd7jFmfKMI48wFngwRNW9Vbo220GbeDAxaIvXLSF/hZcU5749fc26y6fwAwFgg3agA==": "sshd",
			}},
		},
		{
			"sshd.config",
			&llx.Labels{Labels: map[string]string{
				"h1EPuzo5A02wYUOeDzbzv9YfwPO5Km0r1tmJ0UOceHGyO+M2vrEpnF3/XVJu0hOtyAITe0M4O6XOjLOTc8i8lA==": "sshd.config",
			}},
		},
		{
			"sshd.config.params",
			&llx.Labels{Labels: map[string]string{
				"mhgTAYWyl4RGL8my4EskNtiC8WdZdCnvto9+Vp+vdGvTXrsmNCZF2I1dGbbT/2LS8npk1ULPyVFyX4MEE7zwkw==": "sshd.config.params",
			}},
		},
		{
			"sshd.config(\"/my/path\").params",
			&llx.Labels{Labels: map[string]string{
				"WuRyBukFpZbzB1eSaci2IBPTPd+JnEVlEeEfBBTPR7xZjvFvPS/Hhn9WY5z/D7bVhwtddpaxrAPFWfk6djgr1Q==": "sshd.config.params",
			}},
		},
		{
			"asset.name asset.version",
			&llx.Labels{Labels: map[string]string{
				"dfc6mvEo04hkhtJJiFc22KX6/AMf6Fy2kQhrtpTW4TxGWTtwNH19ATbrfbhWlXSxx0BBFCRU4emVM/LsxJdhhw==": "asset.name",
				"5d4FZxbPkZu02MQaHp3C356NJ9TeVsJBw8Enu+TDyBGdWlZM/AE+J5UT/TQ72AmDViKZe97Hxz1Jt3MjcEH/9Q==": "asset.version",
			}},
		},
		{
			"asset { name version }",
			&llx.Labels{
				Labels: map[string]string{
					"5d4FZxbPkZu02MQaHp3C356NJ9TeVsJBw8Enu+TDyBGdWlZM/AE+J5UT/TQ72AmDViKZe97Hxz1Jt3MjcEH/9Q==": "version",
					"HsQJ6Pn7MoZb1V80cTdxHFHZks9QCOBga68ug9JHSivLxNNlGNwGr7dzWVkZhAuVBLgloAWvLnpfr5SzFlG7KA==": "asset",
					"dfc6mvEo04hkhtJJiFc22KX6/AMf6Fy2kQhrtpTW4TxGWTtwNH19ATbrfbhWlXSxx0BBFCRU4emVM/LsxJdhhw==": "name",
				},
			},
		},
		{
			"users.list { uid }",
			&llx.Labels{
				Labels: map[string]string{
					"IB4yJOaaWXlkuGCEIjatVrL5rQZWQucCaOM55RqFxHYXGFvano6W1uqe55OJVo3joocfdpiSZjNqRjse8SMfiA==": "users.list",
					"kijfKPV0fU/MBdcby4ng65mWcsH/kOn5PcVmYvbDBfUlSSSqGKiyhy1Qte+BO/GqMfL62iaaIRP8LgfRZ0/3pg==": "uid",
				},
			},
		},
		{
			"users.list[0]",
			&llx.Labels{
				Labels: map[string]string{
					"IWmJEZKJxco/zD8JR+g8Lqmw49kbCYWSsxQm3QBFf0D0xhVqK8ukpiHhF0TCcDYLm/SrnvpWCnUelRJhqahnZw==": "users.list[0]",
					"MCqGdk4puEdBb/fxS3qDqAV/8gv3DIxFT+InTY7+JcySIzGMDzq8L1t2C8W6qh4z8GI3MvR6ZQ64bVQl0f2Xww==": "uid",
					"T4APLiU1zCnhKjG6cI0dADH4zDmV9qAZ7cwqmY4oUX3iVUDa4VLSotQ3whx+FRFbhaHkg8GI6cyEpN/nyT2jkQ==": "gid",
					"lq0/cF0a/88fFC/0iEmNVILRf68BM92KtITqSh/WSb+UD1QtnydjwcBpC7IW9CSRXekh74bHSm88taykkFx77w==": "name",
				},
			},
		},
		{
			"users.list[0] { uid }",
			&llx.Labels{
				Labels: map[string]string{
					"ITQmg8B2q1g7hUGGDcnFYQjiQ/w1TPr9xyd4dlWAPjwGyRdH2CrtCv55kn7v4SUVqGaJ8k021tUTZznlRzeXNg==": "users.list[0]",
					"MCqGdk4puEdBb/fxS3qDqAV/8gv3DIxFT+InTY7+JcySIzGMDzq8L1t2C8W6qh4z8GI3MvR6ZQ64bVQl0f2Xww==": "uid",
				},
			},
		},
		{
			"sshd.config.params[\"UsePAM\"]",
			&llx.Labels{
				Labels: map[string]string{
					"PSwPW4/H4l4oMTVi+uJnCzKqWAbakhxMi8HjdZMixpF3/CpjPFePhE5Vpe5sgf3DGVIZENZtfB+i0SjMK/c7rw==": "sshd.config.params[UsePAM]",
				},
			},
		},
		{
			"sshd.config { file { path } }",
			&llx.Labels{
				Labels: map[string]string{
					"4dkTkPWdGYANJNsnlIoZxztiguA32f0UoKeYLeVb5Iry/nSYR0RmK6cveUCA6t4fqQJ5RkTwrDlEDHjoE0vURg==": "sshd.config",
					"OuTdAjszQCmHLzp7Y5W3QICyVbGVX3UcnUllLGIXPjQToitI3LKzJ78iVUzMWOmNJZxmbpP7iySzzuFXgflQ+g==": "file",
					"k6rlXoYpV48Qd19gKeNl+/IiPnkI5VNQBiqZBca3gDKsIRiLcpXQUlDv52x9sscIWiqOMpC7+x/aBpY0IUq0ww==": "path",
				},
			},
		},

		// vars
		{
			"a = 1; a",
			&llx.Labels{
				Labels: map[string]string{
					"M3Zw1U5oVhZQeXdyvlpQc6tJz7LG6NiZ7oGQCr1eDSloV75R7lRObrv53UuaHvBOuZG3zBt5BDx9MRoRJwIlfA==": "a",
				},
			},
		},
		{
			"a = 1; b = 2; c = a+b; c",
			&llx.Labels{
				Labels: map[string]string{
					"FU1/hdJ5vadWluEfeQHhklVNU86zhW3zNwxraoHDXJYJj7X2AsjJkhuQjaCfx607pvV/Yjez346tOwzg7i9inQ==": "c",
					// TODO: optimize the code so we don't generate these 2 labels vv
					// they are not needed
					"M3Zw1U5oVhZQeXdyvlpQc6tJz7LG6NiZ7oGQCr1eDSloV75R7lRObrv53UuaHvBOuZG3zBt5BDx9MRoRJwIlfA==": "a",
					"lUa7PEZHR8EfRzYDn+Q38ZZTckepgNlv1sFhRL6l+v7gmV+v/7IxTAoJ2VlAHkCpNU5p5KFLPzPwn6K1Eq27XQ==": "b",
					// ^^
				},
			},
		},
	}

	for i := range tests {
		test := tests[i]
		label(t, test.src, func(labels *llx.Labels) {
			assert.Equal(t, test.labels, labels)
		})
	}
}
