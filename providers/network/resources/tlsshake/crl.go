// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tlsshake

import (
	"crypto/x509"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/mql/utils/multierr"
)

const (
	// crlHTTPTimeout bounds a single CRL download. Revocation is one input to a
	// certificate's findings, not a reason to hold up the whole scan.
	crlHTTPTimeout = 10 * time.Second
	// maxCRLSize caps what is read from a distribution point. Public CRLs run to
	// tens of kilobytes; the cap is what stops a hostile or misconfigured
	// endpoint from streaming until the scanner runs out of memory.
	maxCRLSize = 8 << 20
	// maxCachedCRLs bounds the cache. A chain touches one or two distribution
	// points and a scan reuses an issuer's list across hosts, so this is well
	// above what a real run needs.
	maxCachedCRLs = 16
)

var (
	crlCacheLock sync.Mutex
	crlCache     = map[string]*x509.RevocationList{}
)

// fetchCRL downloads and parses a certificate revocation list, reusing a cached
// copy until the list's own NextUpdate has passed.
//
// The list states how long it is good for, so honoring NextUpdate is both the
// correct cache policy and the one that keeps a scan of many hosts behind the
// same issuer down to a single download.
func fetchCRL(url string) (*x509.RevocationList, error) {
	crlCacheLock.Lock()
	if cached, ok := crlCache[url]; ok && time.Now().Before(cached.NextUpdate) {
		crlCacheLock.Unlock()
		return cached, nil
	}
	crlCacheLock.Unlock()

	client := &http.Client{Timeout: crlHTTPTimeout}
	res, err := client.Get(url)
	if err != nil {
		return nil, multierr.Wrap(err, "failed to download CRL")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("CRL request returned " + res.Status)
	}

	raw, err := io.ReadAll(io.LimitReader(res.Body, maxCRLSize))
	if err != nil {
		return nil, multierr.Wrap(err, "failed to read CRL")
	}

	list, err := x509.ParseRevocationList(raw)
	if err != nil {
		return nil, multierr.Wrap(err, "failed to parse CRL")
	}

	crlCacheLock.Lock()
	// Dropping new entries rather than evicting keeps this allocation-free in
	// the steady state; the cap is far above what a real chain needs.
	if len(crlCache) < maxCachedCRLs {
		crlCache[url] = list
	}
	crlCacheLock.Unlock()

	return list, nil
}

// crlRevocation reports whether a certificate appears on one of the CRLs it
// points at.
//
// The first return says whether a determination was reached at all. A CRL that
// cannot be downloaded, parsed, or attributed to the issuer decides nothing:
// reporting "not revoked" from a list that was never read is the failure this
// whole path exists to avoid.
//
// The list's signature is checked against the issuing certificate, so a CRL
// substituted on the wire cannot vouch for a certificate that has been revoked.
func crlRevocation(cert *x509.Certificate, issuer *x509.Certificate) (bool, *Revocation, error) {
	if len(cert.CRLDistributionPoints) == 0 {
		return false, nil, errors.New("no CRL distribution point specified for revocation check")
	}

	var errs multierr.Errors
	for _, url := range cert.CRLDistributionPoints {
		list, err := fetchCRL(url)
		if err != nil {
			errs.Add(err)
			continue
		}

		if issuer != nil {
			if err := list.CheckSignatureFrom(issuer); err != nil {
				errs.Add(multierr.Wrap(err, "CRL at "+url+" is not signed by the issuing certificate"))
				continue
			}
		}

		for i := range list.RevokedCertificateEntries {
			entry := list.RevokedCertificateEntries[i]
			if entry.SerialNumber.Cmp(cert.SerialNumber) != 0 {
				continue
			}
			return true, &Revocation{
				At:     entry.RevocationTime,
				Via:    url,
				Reason: entry.ReasonCode,
			}, nil
		}

		// The list was read and does not carry this serial, which is what a CRL
		// says when a certificate is good.
		return true, nil, nil
	}

	return false, nil, errs.Deduplicate()
}
