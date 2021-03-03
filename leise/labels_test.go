package leise

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/llx/registry"
)

func label(t *testing.T, s string, f func(res *llx.Labels)) {
	res, err := Compile(s, registry.Default.Schema(), nil)
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
		{"mondoo.version == 'yo'",
			&llx.Labels{
				Labels: map[string]string{
					"J084T/tsf/V2gVKuPUCMiaSli6jjgbrZfBlLtC06P3JdvDMMg2jLWvO8q8CXcZ4rtZf08GRZ5Qsgcoz/4Ph6Vw==": "mondoo.version == \"yo\"",
					"J4anmJ+mXJX380Qslh563U7Bs5d6fiD2ghVxV9knAU0iy/P+IVNZsDhBbCmbpJch3Tm0NliAMiaY47lmw887Jw==": "mondoo.version",
				},
			}},

		{"true",
			&llx.Labels{Labels: map[string]string{
				"13VXYfnMnc74H8XVgiMbH6ZSHxTGQxkhJfUkIiYOBCfUDxHAIJWopMcsea7hXkBTFpbM9lCDnbDBev1z+uagBw==": "",
			}}},
		{"1",
			&llx.Labels{Labels: map[string]string{
				"zcXMKiq4b4QGFVMCvyyFhLXFQKOYn7NKqbV/47XBrKcFwirRjWPgReFt4kdD9G7/ZZCJPsmS4pdCfM32VdTAiQ==": "",
			}}},
		{"1.23",
			&llx.Labels{Labels: map[string]string{
				"wYfzvA9Xuue3Dr0AcPOM9Y8yyd9t+DWggiInRLU5bSoWOoQtxVrt+aNkeOAorYDYV26ni1v6nIGzL6/3EqxSqQ==": "",
			}}},
		{"\"string\"",
			&llx.Labels{Labels: map[string]string{
				"YKg4KBZELSGbdx6hE2dqiH5YWTTjYQDYjVzgUsOxnZs9djRb3SHjCadjEsPq6KlmcRLwo9kpv2fPYEJoQJb2qw==": "",
			}}},
		{"sshd",
			&llx.Labels{Labels: map[string]string{
				"fAVT9TdeX6puAiM5lRS0Rd7jFmfKMI48wFngwRNW9Vbo220GbeDAxaIvXLSF/hZcU5749fc26y6fwAwFgg3agA==": "sshd",
			}}},
		{"sshd.config",
			&llx.Labels{Labels: map[string]string{
				"h1EPuzo5A02wYUOeDzbzv9YfwPO5Km0r1tmJ0UOceHGyO+M2vrEpnF3/XVJu0hOtyAITe0M4O6XOjLOTc8i8lA==": "sshd.config",
			}}},
		{"sshd.config.params",
			&llx.Labels{Labels: map[string]string{
				"mhgTAYWyl4RGL8my4EskNtiC8WdZdCnvto9+Vp+vdGvTXrsmNCZF2I1dGbbT/2LS8npk1ULPyVFyX4MEE7zwkw==": "sshd.config.params",
			}}},
		{"sshd.config(\"/my/path\").params",
			&llx.Labels{Labels: map[string]string{
				"WuRyBukFpZbzB1eSaci2IBPTPd+JnEVlEeEfBBTPR7xZjvFvPS/Hhn9WY5z/D7bVhwtddpaxrAPFWfk6djgr1Q==": "sshd.config.params",
			}}},
		{"platform.name platform.release",
			&llx.Labels{Labels: map[string]string{
				"EpnHIF31KeNgY/3Z4KyBuKHQ0kk/i+MyYbTX+ZWiQIAvK6lv4P2Nlf9CKAIrn2KOfCWICteI96BN1e8GA6sNZA==": "platform.name",
				"yUHOZ/pJzgQ3FLcnKAPphE4TgWqFptqPWA8GYl4e5Dqg0/YzQWcDml2cbrTEj3nj1rm0azm9povOYMRjTgSvZg==": "platform.release",
			}}},
		{"platform { name release }",
			&llx.Labels{
				Labels: map[string]string{
					"tjadxMHtfOxMg9sUP0rM7pXETzjJYmJFKhUcS6GWSiqwPrttgTjKUsOvHo2dutc0Ao2x+rS0REELtEv4Vcuf7Q==": "platform",
					"EpnHIF31KeNgY/3Z4KyBuKHQ0kk/i+MyYbTX+ZWiQIAvK6lv4P2Nlf9CKAIrn2KOfCWICteI96BN1e8GA6sNZA==": "name",
					"yUHOZ/pJzgQ3FLcnKAPphE4TgWqFptqPWA8GYl4e5Dqg0/YzQWcDml2cbrTEj3nj1rm0azm9povOYMRjTgSvZg==": "release",
				},
			}},
		{"users.list { uid }",
			&llx.Labels{
				Labels: map[string]string{
					"mSGW03diY13ig6wZcPouTUrH6fq06Ie9lz23s1a/CwdVGiEBEyhqi+9T8RrkrdoRrn/QKSrC/kcvYE4yd1qo7g==": "users.list",
					"kijfKPV0fU/MBdcby4ng65mWcsH/kOn5PcVmYvbDBfUlSSSqGKiyhy1Qte+BO/GqMfL62iaaIRP8LgfRZ0/3pg==": "uid",
				},
			}},

		{"users.list[0]",
			&llx.Labels{
				Labels: map[string]string{
					"TxBWFcRsfJWnLkUQy4pJkosddFcGzQ9MGz7LyR6IhzC9CrFjA6CZhTx73gj/pcyGG9HZwW3wMwUvnokVnkZqYA==": "users.list[0]",
				},
			}},

		{"users.list[0] { uid }",
			&llx.Labels{
				Labels: map[string]string{
					"JhFnMMVbSxPycbnGcRinWgHu6v4SetPNceTQlWMcv1q0MhVhab9av8DcspcLuPpfuI3IScUfD09rH0ZrB2ea+g==": "users.list[0]",
					"MCqGdk4puEdBb/fxS3qDqAV/8gv3DIxFT+InTY7+JcySIzGMDzq8L1t2C8W6qh4z8GI3MvR6ZQ64bVQl0f2Xww==": "uid",
				},
			}},

		{"sshd.config.params[\"UsePAM\"]",
			&llx.Labels{
				Labels: map[string]string{
					"PSwPW4/H4l4oMTVi+uJnCzKqWAbakhxMi8HjdZMixpF3/CpjPFePhE5Vpe5sgf3DGVIZENZtfB+i0SjMK/c7rw==": "sshd.config.params[UsePAM]",
				},
			}},

		{"sshd.config { file { path } }",
			&llx.Labels{
				Labels: map[string]string{
					"nq3SFDuqajaULpvYBxsfJbvHQzMFY3RDDhLEjg0HXSFPvzthdNZHl8oRuNTx+Z0Zq+bg9MZ+t2CK4WNRj4ru0A==": "sshd.config",
					"g/rqzDTGUq+d5jE7JD/FBhx5WZqA8kd2m0RNNB1lEBv+E8sIx1TgKoxJbmOfInDTrulh4mwpYDHwvlZoGAWXjg==": "file",
					"k6rlXoYpV48Qd19gKeNl+/IiPnkI5VNQBiqZBca3gDKsIRiLcpXQUlDv52x9sscIWiqOMpC7+x/aBpY0IUq0ww==": "path",
				},
			}},
	}

	for i := range tests {
		test := tests[i]
		label(t, test.src, func(labels *llx.Labels) {
			assert.Equal(t, test.labels, labels)
		})
	}
}
