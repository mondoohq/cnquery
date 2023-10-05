// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"errors"
	"io"
	"net/http"

	"go.mondoo.com/cnquery/v9"
	"go.mondoo.com/cnquery/v9/explorer"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/utils/multierr"
)

type fetcher struct {
	cache map[string]*explorer.Bundle
}

func newFetcher() *fetcher {
	return &fetcher{
		cache: map[string]*explorer.Bundle{},
	}
}

func (f *fetcher) fetchBundles(ctx context.Context, schema llx.Schema, urls ...string) (*explorer.Bundle, error) {
	var res *explorer.Bundle = &explorer.Bundle{}

	for i := range urls {
		url := urls[i]
		if cur, ok := f.cache[url]; ok {
			res.AddBundle(cur)
			continue
		}

		cur, err := f.fetchBundle(url)
		if err != nil {
			return nil, err
		}

		// need to generate MRNs for everything
		if _, err := cur.Compile(ctx, schema); err != nil {
			return nil, multierr.Wrap(err, "failed to compile fetched bundle")
		}

		if err = res.AddBundle(cur); err != nil {
			return nil, multierr.Wrap(err, "failed to add fetched bundle")
		}
	}

	return res, nil
}

func (f *fetcher) fetchBundle(url string) (*explorer.Bundle, error) {
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, multierr.Wrap(err, "failed to set up request to fetch bundle")
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; cnquery/"+cnquery.Version+"; +http://www.mondoo.com)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("failed to fetch policy bundle from " + url + ": " + resp.Status)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return explorer.BundleFromYAML(raw)
}
