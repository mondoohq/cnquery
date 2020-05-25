package resources

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

const passwdContent = `root:x:0:0::/root:/bin/bash
bin:x:1:1::/:/usr/bin/nologin
`

func TestResource_File(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"file(\"/etc/passwd\").exists",
			0, true,
		},
		{
			"file(\"/etc/passwd\").basename",
			0, "passwd",
		},
		{
			"file(\"/etc/passwd\").dirname",
			0, "/etc",
		},
		{
			"file(\"/etc/passwd\").size",
			0, int64(58),
		},
		{
			"file(\"/etc/passwd\").permissions.mode",
			0, int64(420),
		},
		{
			"file(\"/etc/passwd\").content",
			0, passwdContent,
		},
	})
}

func TestResource_File_NotExist(t *testing.T) {
	res := testQuery(t, "file('Nope').content")
	assert.Equal(t, errors.New("file 'Nope' does not exist"), res[0].Data.Error)
}
