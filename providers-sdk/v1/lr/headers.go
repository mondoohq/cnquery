package lr

import (
	"strings"
	"text/template"
)

const defaultLicenseHeaderTpl = `{{.LineStarter}} Copyright (c) Mondoo, Inc.
{{.LineStarter}} SPDX-License-Identifier: BUSL-1.1
`

type LicenseHeaderOptions struct {
	LineStarter string
}

func LicenseHeader(tpl *template.Template, opts LicenseHeaderOptions) (string, error) {
	var err error
	if tpl == nil {
		tpl, err = template.New("license_header").Parse(defaultLicenseHeaderTpl)
		if err != nil {
			return "", err
		}
	}

	var header strings.Builder
	if err := tpl.Execute(&header, opts); err != nil {
		return "", err
	}

	return header.String() + "\n", nil
}
