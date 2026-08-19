// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"

	madmin "github.com/minio/madmin-go/v4"
	minio "github.com/minio/minio-go/v7"
)

// s3ConfigAbsentCodes are the S3 error codes MinIO answers with when a bucket
// exists but carries no configuration of the requested kind. Only these may be
// turned into an empty result.
//
// NoSuchBucket is deliberately excluded: a bucket that disappeared between the
// listing and the read is a real change worth reporting, not an empty setting.
// A permission failure is excluded for the same reason a 403 is never "none" --
// an access key that may not read a setting tells us nothing about what the
// setting is, so reporting "not configured" would turn a missing permission
// into a clean audit pass.
var s3ConfigAbsentCodes = map[string]struct{}{
	"ServerSideEncryptionConfigurationNotFoundError": {},
	"NoSuchLifecycleConfiguration":                   {},
	"NoSuchTagSet":                                   {},
	"ObjectLockConfigurationNotFoundError":           {},
	"ReplicationConfigurationNotFoundError":          {},
	"NoSuchBucketPolicy":                             {},
	"NoSuchObjectLockConfiguration":                  {},
	"NoSuchReplicationConfiguration":                 {},
}

// isS3ConfigAbsent reports whether the server answered that the bucket carries
// no configuration of the requested kind.
//
// The classifier matches on the structured error the server returned, never on
// the error text, so a transport failure is not mistaken for a definitive
// answer. A connection refused or a TLS failure produces no minio.ErrorResponse
// at all, which is why the type assertion comes first: without it a network
// blip would degrade to "not configured" and an audit would pass on data that
// was never read.
func isS3ConfigAbsent(err error) bool {
	if err == nil {
		return false
	}
	var respErr minio.ErrorResponse
	if !errors.As(err, &respErr) {
		return false
	}
	_, ok := s3ConfigAbsentCodes[respErr.Code]
	return ok
}

// isS3NoSuchBucket reports whether the server answered that the bucket does not
// exist. It is separate from isS3ConfigAbsent because a vanished bucket means
// every one of its settings is unknown rather than unset.
func isS3NoSuchBucket(err error) bool {
	if err == nil {
		return false
	}
	var respErr minio.ErrorResponse
	if !errors.As(err, &respErr) {
		return false
	}
	return respErr.Code == "NoSuchBucket"
}

// adminNotFoundCodes are the admin API error codes MinIO answers with when the
// named object does not exist.
var adminNotFoundCodes = map[string]struct{}{
	"XMinioAdminNoSuchUser":           {},
	"XMinioAdminNoSuchGroup":          {},
	"XMinioAdminNoSuchPolicy":         {},
	"XMinioAdminNoSuchServiceAccount": {},
	"XMinioAdminNoSuchAccessKey":      {},
}

// isAdminNotFound reports whether the admin API answered that the named object
// does not exist. As with the S3 classifier it matches on the structured error
// only, so a transport failure never reads as an absent object.
func isAdminNotFound(err error) bool {
	if err == nil {
		return false
	}
	var respErr madmin.ErrorResponse
	if !errors.As(err, &respErr) {
		return false
	}
	_, ok := adminNotFoundCodes[respErr.Code]
	return ok
}
