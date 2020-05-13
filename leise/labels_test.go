package leise

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/llx/registry"
)

func label(t *testing.T, s string, f func(res *llx.Labels)) {
	res, err := Compile(s, registry.Default.Schema())
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
				"actnqYQLWDh5cxtVy8rfK3l1fuNL/GKx7+AhXMz5p94+Owz53454WHYsJ/QHb0yNDae5vTjgKnpRFRVKXrOiBw==": "sshd.config.params",
			}}},
		{"sshd.config(\"/my/path\").params",
			&llx.Labels{Labels: map[string]string{
				"MM6K7Xt2myq6kCdTrfBpbWjR1+ehr4QjM+Kx6Y0cUWFmznjodWF7ALNZOa//9sZbjwrNK6cqlhYqCKyBDpF2XQ==": "sshd.config.params",
			}}},
		{"platform.name platform.release",
			&llx.Labels{Labels: map[string]string{
				"EpnHIF31KeNgY/3Z4KyBuKHQ0kk/i+MyYbTX+ZWiQIAvK6lv4P2Nlf9CKAIrn2KOfCWICteI96BN1e8GA6sNZA==": "platform.name",
				"yUHOZ/pJzgQ3FLcnKAPphE4TgWqFptqPWA8GYl4e5Dqg0/YzQWcDml2cbrTEj3nj1rm0azm9povOYMRjTgSvZg==": "platform.release",
			}}},
		{"platform { name release }",
			&llx.Labels{
				Labels: map[string]string{
					"EpnHIF31KeNgY/3Z4KyBuKHQ0kk/i+MyYbTX+ZWiQIAvK6lv4P2Nlf9CKAIrn2KOfCWICteI96BN1e8GA6sNZA==": "name",
					"mTunkQvf2+3hvQXOVe9aZvuyULh8FAY9UERpZnawCtDF0FLgQ06M17OE7sKCjE1JwsmB3Hl57YvT6Brk1JKZEw==": "platform",
					"yUHOZ/pJzgQ3FLcnKAPphE4TgWqFptqPWA8GYl4e5Dqg0/YzQWcDml2cbrTEj3nj1rm0azm9povOYMRjTgSvZg==": "release",
				},
			}},
		{"users.list { uid }",
			&llx.Labels{
				Labels: map[string]string{
					"puRCaea3UwDYUekHCB0RwC+ezqX7ENxHZXtCP1sox6O177ZGNbyShvkKadiYamiVGnhsn2EvGWqA3KBKkcT6gg==": "users.list",
					"PwdDaV8xrCNIvOSwTLwmNR301BKQRqkZa2q7ZLLPoblw5AuQfqqIfjpWt5nNh+FBnr+vu62u0aphsamJKX63cg==": "uid",
				},
			}},

		{"users.list[0]",
			&llx.Labels{
				Labels: map[string]string{
					"JRBY9Hgf44wZCNyBMqI/3qnRI6Vn9okb5sBH1rm93GzmhMsxfgS5GOZ17hmuUK+ZAtTRlW+WL6KNs9+L9CEXfw==": "users.list[0]",
				},
			}},

		{"users.list[0] { uid }",
			&llx.Labels{
				Labels: map[string]string{
					"iZXN2sEhIQ4vBZtKqztQP9GdO8nYwFcrbNnI4TJ2W5mY3lUuKuQZJN36oyentTUFo9GL1GDG+b6ELfULFrhdPA==": "users.list[0]",
					"vXUmtyv5Thql2cXelEul3Xaa8v7oNbP8ve/kufi3J8reZNVvp2dnoKPW+av/wIL6x6ma2cCxB/UoHuovKwuypw==": "uid",
				},
			}},

		{"sshd.config.params[\"UsePAM\"]",
			&llx.Labels{
				Labels: map[string]string{
					"ANn7ciWfTVSHM5K6f4zOlY6BhSEURGhlL0W+2T1aWWLDz4Lz4QCVntNNBUXr0xHTyMoYRuomj13o/LpNEf+VVQ==": "sshd.config.params[UsePAM]",
				},
			}},

		{"sshd.config { file { path } }",
			&llx.Labels{
				Labels: map[string]string{
					"U/wvYhMvx+9ErgRV+fkYjf3gSMF6dJdxiCb8bti9qWTmkoS0QGWlID5QaUbBtNlrQeYfiUI5fcVtssOUHeWD+A==": "sshd.config",
					"ttBFT/PXbbh/0LvFg94+cPLCfyDljxab2tjh/BFj03HoqRhK6AsX8mGGhAA0pTqNjyTAXv9OwSC236ojlnVjhQ==": "file",
					"7UaqV74XSP+zpDe2jWCIK7knE3Oq+OWA79/8o/iQcBisCqUcafc878wFLOzGqDVOZZAiqGOKcSwZXVivDpnmjQ==": "path",
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
