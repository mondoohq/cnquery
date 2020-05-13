package resources

import (
	"testing"
)

const passwdContent = `root:x:0:0::/root:/bin/bash
bin:x:1:1::/:/usr/bin/nologin
`

func TestResource_File(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"file(\"/etc/passwd\").exists",
			true,
		},
		{
			"file(\"/etc/passwd\").basename",
			"passwd",
		},
		{
			"file(\"/etc/passwd\").dirname",
			"/etc",
		},
		{
			"file(\"/etc/passwd\").size",
			int64(58),
		},
		{
			"file(\"/etc/passwd\").permissions.mode",
			int64(420),
		},
		{
			"file(\"/etc/passwd\").content",
			passwdContent,
		},
	})
}
