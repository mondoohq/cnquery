// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"regexp"
	"strconv"

	"github.com/Masterminds/semver"
	"go.mondoo.com/cnquery/v11/types"
)

// Version type, an abstract representation of a software version.
// It is designed to parse and compare version strings.
// It is built on semver and adds support for epochs (deb, rpm, python).
type Version struct {
	*semver.Version
	src   string
	epoch int
}

var reEpoch = regexp.MustCompile("^[0-9]+[:!]")

func NewVersion(s string) Version {
	epoch := 0

	x, err := semver.NewVersion(s)
	if err != nil {
		x, epoch = parseEpoch(s)
	}

	return Version{
		Version: x,
		src:     s,
		epoch:   epoch,
	}
}

func parseEpoch(v string) (*semver.Version, int) {
	prefix := reEpoch.FindString(v)
	if prefix == "" {
		return nil, 0
	}

	remainder := v[len(prefix):]
	epochStr := v[:len(prefix)-1]
	res, err := semver.NewVersion(remainder)
	if err != nil {
		return nil, 0
	}

	// invalid epoch means we discard the entire version string
	epoch, err := strconv.Atoi(epochStr)
	if err != nil {
		return nil, 0
	}

	return res, epoch
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

func semverLT(left interface{}, right interface{}) *RawData {
	l := NewVersion(left.(string))
	r := NewVersion(right.(string))
	return BoolData(l.Compare(r) < 0)
}

func semverGT(left interface{}, right interface{}) *RawData {
	l := NewVersion(left.(string))
	r := NewVersion(right.(string))
	return BoolData(l.Compare(r) > 0)
}

func semverLTE(left interface{}, right interface{}) *RawData {
	l := NewVersion(left.(string))
	r := NewVersion(right.(string))
	return BoolData(l.Compare(r) <= 0)
}

func semverGTE(left interface{}, right interface{}) *RawData {
	l := NewVersion(left.(string))
	r := NewVersion(right.(string))
	return BoolData(l.Compare(r) >= 0)
}

func semverCmpSemver(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left, right interface{}) *RawData {
		return BoolData(left.(string) == right.(string))
	})
}

func semverNotSemver(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left, right interface{}) *RawData {
		return BoolData(left.(string) != right.(string))
	})
}

func semverLTsemver(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, semverLT)
}

func semverGTsemver(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, semverGT)
}

func semverLTEsemver(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, semverLTE)
}

func semverGTEsemver(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, semverGTE)
}

func semverEpoch(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}

	v := NewVersion(bind.Value.(string))
	return IntData(v.epoch), 0, nil
}
