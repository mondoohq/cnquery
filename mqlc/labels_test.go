// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mqlc_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/mqlc"
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
					"p+Ev/uaC5uYn5PCkwtGWYG7KBBSQZGdjFrxOIgGtCl4rEn2aeegTkvGPA0CQFsln0u30+/G9f+SEvyX1Pf0R2A==": "users.list",
					"1BF9D6QaYM83qOLACgwaUWnwU1TW0SJlOwFE9TktNq/7sV4LzUbv1O3gPBUKO08ZJZNHQ+ucFQRC726bqcTncw==": "uid",
				},
			},
		},
		{
			"users.list[0]",
			&llx.Labels{
				Labels: map[string]string{
					"DkmEtbI05F7wdVsbPvb/WYmz23krsv+dzyGtLbOvaZAfBdue+1sWEFjWOeTfDvwH0+QkcOAr2VkoXYwjypBOWg==": "users.list[0]",
					"YZAwDOgeLgfhdF4u2ZXYAEtAOKIJ+wCb6ngklIu3zN/k2wYfQAjGB73PbxromnPw9Fcg4gDB3EwDoZOehdkyPA==": "uid",
					"h+MQ3LiiaZv39U2piraB1akgzfFdnBXVpbKH8Ak7+e44Hnt15Dkvjls9GU8SSVRuWyxNhjINxUUoW1Tyq7MFYg==": "gid",
					"lq0/cF0a/88fFC/0iEmNVILRf68BM92KtITqSh/WSb+UD1QtnydjwcBpC7IW9CSRXekh74bHSm88taykkFx77w==": "name",
				},
			},
		},
		{
			"users.list[0] { uid }",
			&llx.Labels{
				Labels: map[string]string{
					"SGUkje4+8mnTUORpDNQOIDg0aY1aOMSSQv1GXftjDJuMCqLal+bEj0efrbp6glq3UO9Ttmd1NEguh5JE9gS/gQ==": "users.list[0]",
					"YZAwDOgeLgfhdF4u2ZXYAEtAOKIJ+wCb6ngklIu3zN/k2wYfQAjGB73PbxromnPw9Fcg4gDB3EwDoZOehdkyPA==": "uid",
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
		{
			"1 >= 2",
			&llx.Labels{
				Labels: map[string]string{
					"KN8O7dRC1dktiemLwSo3kqNRIwRf5jXqEqPsa2O7bgs31z7LM3fxYrm6tOCyzYAC7Jpic3q6CUYbbYtn7yaifQ==": " >= 2",
				},
			},
		},
		{
			"1 > 2",
			&llx.Labels{
				Labels: map[string]string{
					"hPdqqG1LQ4F3OnnS3gey0/665f+p0XLMoOVCTUScM/pMQuothxnwd0TyuzQOIPn3fkpf+kTtxZNgi0y688AN9Q==": " > 2",
				},
			},
		},
		{
			"1 <= 2",
			&llx.Labels{
				Labels: map[string]string{
					"JxA4wNPJRq2CsRwXpkvnx7leIVGoeg1e8s3En5Aize9mdZoPD7GpLb9JG86dh30DzdgncT+Hgm87nZehVdgw2w==": " <= 2",
				},
			},
		},
		{
			"1 < 2",
			&llx.Labels{
				Labels: map[string]string{
					"PaamyN/AAZNMNmh4OAzjmG/ArLYReuNzi4p2KyDyDE/CZw+puUX0A2oCV8aHh+QJvF5o/4TfjvL+vCmb7Ge9KA==": " < 2",
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
