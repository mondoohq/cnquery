// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package bundle

import (
	"context"
	"fmt"
	"go.mondoo.com/cnquery/v10/explorer"
	"go.mondoo.com/cnquery/v10/providers"
	"strconv"
)

func Lint(queryPackBundle *explorer.Bundle) []string {
	errors := []string{}

	// check that we have uids for packs and queries
	for i := range queryPackBundle.Packs {
		pack := queryPackBundle.Packs[i]
		packId := strconv.Itoa(i)

		if pack.Uid == "" {
			errors = append(errors, fmt.Sprintf("pack %s does not define a uid", packId))
		} else {
			packId = pack.Uid
		}

		if pack.Name == "" {
			errors = append(errors, fmt.Sprintf("pack %s does not define a name", packId))
		}

		for j := range pack.Queries {
			query := pack.Queries[j]
			queryId := strconv.Itoa(j)
			if query.Uid == "" {
				errors = append(errors, fmt.Sprintf("query %s/%s does not define a uid", packId, queryId))
			} else {
				queryId = query.Uid
			}

			if query.Title == "" {
				errors = append(errors, fmt.Sprintf("query %s/%s does not define a name", packId, queryId))
			}
		}
	}

	// we compile after the checks because it removes the uids and replaces it with mrns
	schema := providers.DefaultRuntime().Schema()
	_, err := queryPackBundle.Compile(context.Background(), schema)
	if err != nil {
		errors = append(errors, "could not compile the query pack bundle", err.Error())
	}

	return errors
}
