// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package wheelegg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMimeParser(t *testing.T) {

	content := `Metadata-Version: 2.1
Name: pyftpdlib
Version: 1.5.7
Summary: Very fast asynchronous FTP server library
Home-page: https://github.com/giampaolo/pyftpdlib/
Author: Giampaolo Rodola'
Author-email: g.rodola@gmail.com
License: MIT
Keywords: ftp,ftps,server,ftpd,daemon,python,ssl,sendfile,asynchronous,nonblocking,eventdriven,rfc959,rfc1123,rfc2228,rfc2428,rfc2640,rfc3659
Platform: Platform Independent
Classifier: Development Status :: 5 - Production/Stable
Classifier: Environment :: Console
Classifier: Intended Audience :: Developers
Classifier: Intended Audience :: System Administrators
Classifier: License :: OSI Approved :: MIT License
Classifier: Operating System :: OS Independent
Classifier: Programming Language :: Python
Classifier: Topic :: Internet :: File Transfer Protocol (FTP)
Classifier: Topic :: Software Development :: Libraries :: Python Modules
Classifier: Topic :: System :: Filesystems
Classifier: Programming Language :: Python
Classifier: Programming Language :: Python :: 2
Classifier: Programming Language :: Python :: 3
Provides-Extra: ssl
License-File: LICENSE
`
	pkg, err := ParseMIME(strings.NewReader(content), "/usr/lib/python3.11/site-packages/pyftpdlib-1.5.7-py3.11.egg-info/PKG-INFO")
	require.NoError(t, err)

	assert.Equal(t, "Giampaolo Rodola'", pkg.Author)
	assert.Equal(t, "g.rodola@gmail.com", pkg.AuthorEmail)
	assert.Equal(t, "pyftpdlib", pkg.Name)
	assert.Equal(t, "1.5.7", pkg.Version)
	assert.Equal(t, "MIT", pkg.License)
}
