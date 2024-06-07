// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package registry

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	"golang.org/x/sys/windows/registry"
)

// parseRegistryKeyPath parses a registry key path into the hive and the path
// https://learn.microsoft.com/en-us/windows/win32/sysinfo/registry-hives
func parseRegistryKeyPath(path string) (registry.Key, string, error) {
	if strings.HasPrefix(path, "HKEY_LOCAL_MACHINE") {
		return registry.LOCAL_MACHINE, strings.TrimPrefix(path, "HKEY_LOCAL_MACHINE\\"), nil
	}
	if strings.HasPrefix(path, "HKLM") {
		return registry.LOCAL_MACHINE, strings.TrimPrefix(path, "HKLM\\"), nil
	}

	if strings.HasPrefix(path, "HKEY_CURRENT_USER") {
		return registry.CURRENT_USER, strings.TrimPrefix(path, "HKEY_CURRENT_USER\\"), nil
	}

	if strings.HasPrefix(path, "HKCU") {
		return registry.CURRENT_USER, strings.TrimPrefix(path, "HKCU\\"), nil
	}

	if strings.HasPrefix(path, "HKEY_USERS") {
		return registry.USERS, strings.TrimPrefix(path, "HKEY_USERS\\"), nil
	}

	return registry.LOCAL_MACHINE, "", errors.New("invalid registry key hive: " + path)
}

func GetNativeRegistryKeyItems(path string) ([]RegistryKeyItem, error) {
	log.Debug().Str("path", path).Msg("search registry key values using native registry api")
	key, path, err := parseRegistryKeyPath(path)
	if err != nil {
		return nil, err
	}
	regKey, err := registry.OpenKey(key, path, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
	if err != nil && registry.ErrNotExist == err {
		return nil, status.Error(codes.NotFound, "registry key not found: "+path)
	} else if err != nil {
		return nil, err
	}
	defer regKey.Close()

	res := []RegistryKeyItem{}
	values, err := regKey.ReadValueNames(0)
	if err != nil {
		return nil, err
	}
	for _, value := range values {
		stringValue, valtype, err := regKey.GetStringValue(value)
		if err != registry.ErrUnexpectedType && err != nil {
			return nil, err
		}

		regValue := RegistryKeyValue{
			Kind:   int(valtype),
			String: stringValue,
		}

		switch valtype {
		case registry.SZ, registry.EXPAND_SZ:
			// covered by GetStringValue, nothing to do
		case registry.BINARY:
			binaryValue, _, err := regKey.GetBinaryValue(value)
			if err != nil {
				return nil, err
			}
			regValue.Binary = binaryValue
		case registry.DWORD:
			fallthrough
		case registry.QWORD:
			intVal, _, err := regKey.GetIntegerValue(value)
			if err != nil {
				return nil, err
			}
			regValue.Number = int64(intVal)
			regValue.String = strconv.FormatInt(int64(intVal), 10)
		case registry.MULTI_SZ:
			entries, _, err := regKey.GetStringsValue(value)
			if err != nil {
				return nil, err
			}
			regValue.MultiString = entries
			if len(entries) > 0 {
				// NOTE: this is to be consistent with the output before we moved to multi-datatype support for registry keys
				regValue.String = strings.Join(entries, " ")
			}
		case registry.DWORD_BIG_ENDIAN, registry.LINK, registry.RESOURCE_LIST, registry.FULL_RESOURCE_DESCRIPTOR, registry.RESOURCE_REQUIREMENTS_LIST:
			// not supported by golang.org/x/sys/windows/registry
		}
		res = append(res, RegistryKeyItem{
			Key:   value,
			Value: regValue,
		})
	}
	return res, nil
}

func GetNativeRegistryKeyChildren(fullPath string) ([]RegistryKeyChild, error) {
	log.Debug().Str("path", fullPath).Msg("search registry key children using native registry api")
	key, path, err := parseRegistryKeyPath(fullPath)
	if err != nil {
		return nil, err
	}

	regKey, err := registry.OpenKey(key, path, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
	if err != nil && registry.ErrNotExist == err {
		return nil, status.Error(codes.NotFound, "registry key not found: "+path)
	} else if err != nil {
		return nil, err
	}
	defer regKey.Close()

	// reads all child keys
	entries, err := regKey.ReadSubKeyNames(0)
	if err != nil {
		return nil, err
	}

	res := make([]RegistryKeyChild, len(entries))

	for i, entry := range entries {
		res[i] = RegistryKeyChild{
			Path: fullPath,
			Name: entry,
		}
	}

	return res, nil
}

func GetNativeRegistryKeyItem(path, key string) (RegistryKeyItem, error) {
	values, err := GetNativeRegistryKeyItems(path)
	if err != nil {
		return RegistryKeyItem{}, err
	}
	for _, value := range values {
		if value.Key == key {
			return value, nil
		}
	}
	return RegistryKeyItem{}, status.Error(codes.NotFound, fmt.Sprintf("registry value %s not found under %s", key, path))
}
