// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import "go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"

// derived from "golang.org/x/sys/windows/registry"
// see https://github.com/golang/sys/blob/master/windows/registry/value.go#L17-L31
const (
	NONE                       = 0
	SZ                         = 1
	EXPAND_SZ                  = 2
	BINARY                     = 3
	DWORD                      = 4
	DWORD_BIG_ENDIAN           = 5
	LINK                       = 6
	MULTI_SZ                   = 7
	RESOURCE_LIST              = 8
	FULL_RESOURCE_DESCRIPTOR   = 9
	RESOURCE_REQUIREMENTS_LIST = 10
	QWORD                      = 11
)

type RegistryKeyItem struct {
	Key   string
	Value RegistryKeyValue
}

func (k RegistryKeyItem) Kind() string {
	switch k.Value.Kind {
	case NONE:
		return "bone"
	case SZ:
		return "string"
	case EXPAND_SZ:
		return "expandstring"
	case BINARY:
		return "binary"
	case DWORD:
		return "dword"
	case DWORD_BIG_ENDIAN:
		return "dword"
	case LINK:
		return "link"
	case MULTI_SZ:
		return "multistring"
	case RESOURCE_LIST:
		return "<unsupported>"
	case FULL_RESOURCE_DESCRIPTOR:
		return "<unsupported>"
	case RESOURCE_REQUIREMENTS_LIST:
		return "<unsupported>"
	case QWORD:
		return "qword"
	}
	return "<unsupported>"
}

func (k RegistryKeyItem) GetRawValue() interface{} {
	switch k.Value.Kind {
	case NONE:
		return nil
	case SZ:
		return k.Value.String
	case EXPAND_SZ:
		return k.Value.String
	case BINARY:
		return k.Value.Binary
	case DWORD:
		return k.Value.Number
	case DWORD_BIG_ENDIAN:
		return nil
	case LINK:
		return nil
	case MULTI_SZ:
		return convert.SliceAnyToInterface(k.Value.MultiString)
	case RESOURCE_LIST:
		return nil
	case FULL_RESOURCE_DESCRIPTOR:
		return nil
	case RESOURCE_REQUIREMENTS_LIST:
		return nil
	case QWORD:
		return k.Value.Number
	}
	return nil
}

// String returns a string representation of the registry key value
func (k RegistryKeyItem) String() string {
	return k.Value.String // conversion to string is handled in UnmarshalJSON
}

type RegistryKeyValue struct {
	Kind        int
	Binary      []byte
	Number      int64
	String      string
	MultiString []string
}

type RegistryKeyChild struct {
	Name       string
	Path       string
	Properties []string
}
