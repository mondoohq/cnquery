// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"github.com/Masterminds/semver"
	"go.mondoo.com/cnquery/v9/types"
)

func semverLT(left interface{}, right interface{}) *RawData {
	leftv, err := semver.NewVersion(left.(string))
	if err != nil {
		return BoolData(left.(string) < right.(string))
	}
	rightv, err := semver.NewVersion(right.(string))
	if err != nil {
		return BoolData(left.(string) < right.(string))
	}
	return BoolData(leftv.LessThan(rightv))
}

func semverGT(left interface{}, right interface{}) *RawData {
	leftv, err := semver.NewVersion(left.(string))
	if err != nil {
		return BoolData(left.(string) > right.(string))
	}
	rightv, err := semver.NewVersion(right.(string))
	if err != nil {
		return BoolData(left.(string) > right.(string))
	}
	return BoolData(leftv.GreaterThan(rightv))
}

func semverLTE(left interface{}, right interface{}) *RawData {
	leftv, err := semver.NewVersion(left.(string))
	if err != nil {
		return BoolData(left.(string) <= right.(string))
	}
	rightv, err := semver.NewVersion(right.(string))
	if err != nil {
		return BoolData(left.(string) <= right.(string))
	}
	return BoolData(!leftv.GreaterThan(rightv))
}

func semverGTE(left interface{}, right interface{}) *RawData {
	leftv, err := semver.NewVersion(left.(string))
	if err != nil {
		return BoolData(left.(string) >= right.(string))
	}
	rightv, err := semver.NewVersion(right.(string))
	if err != nil {
		return BoolData(left.(string) >= right.(string))
	}
	return BoolData(!leftv.LessThan(rightv))
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
