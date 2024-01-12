// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v10/types"
)

var BsdCertFiles = []string{
	"/usr/local/etc/ssl/cert.pem",            // FreeBSD
	"/etc/ssl/cert.pem",                      // OpenBSD
	"/usr/local/share/certs/ca-root-nss.crt", // DragonFly
	"/etc/openssl/certs/ca-certificates.crt", // NetBSD
}

var LinuxCertFiles = []string{
	"/etc/ssl/certs/ca-certificates.crt",                // Debian/Ubuntu/Gentoo etc.
	"/etc/pki/tls/certs/ca-bundle.crt",                  // Fedora/RHEL 6
	"/etc/ssl/ca-bundle.pem",                            // OpenSUSE
	"/etc/pki/tls/cacert.pem",                           // OpenELEC
	"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem", // CentOS/RHEL 7
	"/etc/ssl/cert.pem",                                 // Alpine Linux
}

var LinuxCertDirectories = []string{
	"/etc/ssl/certs",               // SLES10/SLES11, https://golang.org/issue/12139
	"/system/etc/security/cacerts", // Android
	"/usr/local/share/certs",       // FreeBSD
	"/etc/pki/tls/certs",           // Fedora/RHEL
	"/etc/openssl/certs",           // NetBSD
	"/var/ssl/certs",               // AIX
}

func (s *mqlOsRootCertificates) id() (string, error) {
	return "osrootcertificates", nil
}

func initOsRootCertificates(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(shared.Connection)
	platform := conn.Asset().Platform

	var paths []string
	if platform.IsFamily("linux") {
		paths = LinuxCertFiles
	} else if platform.IsFamily("bsd") {
		paths = BsdCertFiles
	} else {
		return nil, nil, errors.New("root certificates are not unsupported on this platform: " + platform.Name + " " + platform.Version)
	}

	// search the first file that exists, it mimics the behavior go is doing
	files := []interface{}{}
	for i := range paths {
		f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path": llx.StringData(paths[i]),
		})
		if err != nil {
			return nil, nil, err
		}

		file := f.(*mqlFile)
		if !file.GetExists().Data {
			log.Trace().Err(err).Str("path", paths[i]).Msg("os.rootcertificates> file does not exist")
			continue
		}
		perm := file.GetPermissions()
		if perm.Error != nil {
			log.Trace().Err(err).Str("path", paths[i]).Msg("os.rootcertificates> failed to get permissions")
			continue
		}
		if !perm.Data.GetIsFile().Data {
			continue
		}

		files = append(files, file)
	}

	args["files"] = llx.ArrayData(files, types.Resource("file"))

	return args, nil, nil
}

func (s *mqlOsRootCertificates) content(files []interface{}) ([]interface{}, error) {
	contents := []interface{}{}

	for i := range files {
		file := files[i].(*mqlFile)

		content := file.GetContent()
		if content.Error != nil {
			return nil, content.Error
		}
		contents = append(contents, content.Data)
	}

	return contents, nil
}

func (p *mqlOsRootCertificates) list(contents []interface{}) ([]interface{}, error) {
	var res []interface{}
	for i := range contents {
		certificates, err := p.MqlRuntime.CreateSharedResource("certificates", map[string]*llx.RawData{
			"pem": llx.StringData(contents[i].(string)),
		})
		if err != nil {
			return nil, err
		}

		list, err := p.MqlRuntime.GetSharedData("certificates", certificates.MqlID(), "list")
		if err != nil {
			return nil, err
		}

		res = append(res, list.Value.([]interface{})...)
	}

	return res, nil
}
