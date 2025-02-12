// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"regexp"
	"strconv"

	"github.com/Masterminds/semver"
	"go.mondoo.com/cnquery/v11/types"
)

type VersionType byte

const (
	UNKNOWN_VERSION VersionType = iota + 1
	SEMVER
	DEBIAN_VERSION
	PYTHON_VERSION
)

// Version type, an abstract representation of a software version.
// It is designed to parse and compare version strings.
// It is built on semver and adds support for epochs (deb, rpm, python).
type Version struct {
	*semver.Version
	src   string
	typ   VersionType
	epoch int
}

var reEpoch = regexp.MustCompile("^[0-9]+[:!]")

func NewVersion(s string) Version {
	epoch := 0

	var typ VersionType
	x, err := semver.NewVersion(s)
	if err == nil {
		typ = SEMVER
	} else {
		x, epoch, typ = parseEpoch(s)
	}

	return Version{
		Version: x,
		src:     s,
		typ:     typ,
		epoch:   epoch,
	}
}

func parseEpoch(v string) (*semver.Version, int, VersionType) {
	prefix := reEpoch.FindString(v)
	if prefix == "" {
		return nil, 0, UNKNOWN_VERSION
	}

	remainder := v[len(prefix):]
	epochStr := v[:len(prefix)-1]
	res, err := semver.NewVersion(remainder)
	if err != nil {
		return nil, 0, UNKNOWN_VERSION
	}

	// invalid epoch means we discard the entire version string
	epoch, err := strconv.Atoi(epochStr)
	if err != nil {
		return nil, 0, UNKNOWN_VERSION
	}

	if prefix[len(prefix)-1] == ':' {
		return res, epoch, DEBIAN_VERSION
	}
	return res, epoch, PYTHON_VERSION
}

// Compare compares this version to another one. It returns -1, 0, or 1 if
// the version smaller, equal, or larger than the other version.
//
// Versions are compared by X.Y.Z. Build metadata is ignored. Prerelease is
// lower than the version without a prerelease.
func (v Version) Compare(o Version) int {
	if v.epoch != o.epoch {
		return v.epoch - o.epoch
	}
	if v.Version == nil || o.Version == nil {
		if v.src < o.src {
			return -1
		} else if v.src > o.src {
			return 1
		}
		return 0
	}

	return v.Version.Compare(o.Version)
}

func versionLT(left interface{}, right interface{}) *RawData {
	l := NewVersion(left.(string))
	r := NewVersion(right.(string))
	return BoolData(l.Compare(r) < 0)
}

func versionGT(left interface{}, right interface{}) *RawData {
	l := NewVersion(left.(string))
	r := NewVersion(right.(string))
	return BoolData(l.Compare(r) > 0)
}

func versionLTE(left interface{}, right interface{}) *RawData {
	l := NewVersion(left.(string))
	r := NewVersion(right.(string))
	return BoolData(l.Compare(r) <= 0)
}

func versionGTE(left interface{}, right interface{}) *RawData {
	l := NewVersion(left.(string))
	r := NewVersion(right.(string))
	return BoolData(l.Compare(r) >= 0)
}

func versionCmpVersion(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left, right interface{}) *RawData {
		return BoolData(left.(string) == right.(string))
	})
}

func versionNotVersion(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left, right interface{}) *RawData {
		return BoolData(left.(string) != right.(string))
	})
}

func versionLTversion(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, versionLT)
}

func versionGTversion(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, versionGT)
}

func versionLTEversion(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, versionLTE)
}

func versionGTEversion(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, versionGTE)
}

func versionEpoch(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}

	v := NewVersion(bind.Value.(string))
	return IntData(v.epoch), 0, nil
}
