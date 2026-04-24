// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"

	providerresources "go.mondoo.com/mql/v13/providers-sdk/v1/resources"
)

func fileContentOrEmpty(file *mqlFile) (string, error) {
	if file == nil {
		return "", nil
	}

	content := file.GetContent()
	if content.Error != nil {
		var notFound providerresources.NotFoundError
		if errors.As(content.Error, &notFound) {
			return "", nil
		}
		return "", content.Error
	}

	return content.Data, nil
}
