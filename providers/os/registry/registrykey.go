// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package registry

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
)

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

type keyKindRaw struct {
	Kind int
	Data interface{}
}

func (k *RegistryKeyValue) UnmarshalJSON(b []byte) error {
	var raw keyKindRaw

	// try to unmarshal the type
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}
	k.Kind = raw.Kind

	if raw.Data == nil {
		return nil
	}

	// see https://docs.microsoft.com/en-us/powershell/scripting/samples/working-with-registry-entries?view=powershell-7
	switch raw.Kind {
	case NONE:
		// ignore
	case SZ: // Any string value
		value, ok := raw.Data.(string)
		if !ok {
			return fmt.Errorf("registry key value is not a string: %v", raw.Data)
		}
		k.String = value
	case EXPAND_SZ: // A string that can contain environment variables that are dynamically expanded
		value, ok := raw.Data.(string)
		if !ok {
			return fmt.Errorf("registry key value is not a string: %v", raw.Data)
		}
		k.String = value
	case BINARY: // Binary data
		rawData, ok := raw.Data.([]interface{})
		if !ok {
			return fmt.Errorf("registry key value is not a byte array: %v", raw.Data)
		}
		data := make([]byte, len(rawData))
		for i, v := range rawData {
			val, ok := v.(float64)
			if !ok {
				return fmt.Errorf("registry key value is not a byte array: %v", raw.Data)
			}
			data[i] = byte(val)
		}
		k.Binary = data
	case DWORD: // A number that is a valid UInt32
		data, ok := raw.Data.(float64)
		if !ok {
			return fmt.Errorf("registry key value is not a number: %v", raw.Data)
		}
		number := int64(data)
		// string fallback
		k.Number = number
		k.String = strconv.FormatInt(number, 10)
	case DWORD_BIG_ENDIAN:
		log.Warn().Msg("DWORD_BIG_ENDIAN for registry key is not supported")
	case LINK:
		log.Warn().Msg("LINK for registry key is not supported")
	case MULTI_SZ: // A multiline string
		switch value := raw.Data.(type) {
		case string:
			k.String = value
			if value != "" {
				k.MultiString = []string{value}
			}
		case []interface{}:
			if len(value) > 0 {
				var multiString []string
				for _, v := range value {
					multiString = append(multiString, v.(string))
				}
				// NOTE: this is to be consistent with the output before we moved to multi-datatype support for registry keys
				k.String = strings.Join(multiString, " ")
				k.MultiString = multiString
			}
		}
	case RESOURCE_LIST:
		log.Warn().Msg("RESOURCE_LIST for registry key is not supported")
	case FULL_RESOURCE_DESCRIPTOR:
		log.Warn().Msg("FULL_RESOURCE_DESCRIPTOR for registry key is not supported")
	case RESOURCE_REQUIREMENTS_LIST:
		log.Warn().Msg("RESOURCE_REQUIREMENTS_LIST for registry key is not supported")
	case QWORD: // 8 bytes of binary data
		f, ok := raw.Data.(float64)
		if !ok {
			return fmt.Errorf("registry key value is not a number: %v", raw.Data)
		}
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf[:], math.Float64bits(f))
		k.Binary = buf
	}
	return nil
}
