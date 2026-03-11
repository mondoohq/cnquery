// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package zfs

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Pool represents a parsed ZFS storage pool.
type Pool struct {
	Name          string
	GUID          string
	Size          int64
	Allocated     int64
	Free          int64
	Fragmentation int64
	PercentUsed   int64
	Dedupratio    float64
	Health        string
	Readonly      bool
	Autoexpand    bool
	Autoreplace   bool
	Autotrim      bool
}

// Vdev represents a virtual device in a ZFS pool's topology.
type Vdev struct {
	Name           string
	Type           string
	State          string
	Path           string
	ReadErrors     int64
	WriteErrors    int64
	ChecksumErrors int64
	SlowIOs        int64
	Devices        []Vdev
}

// Dataset represents a parsed ZFS dataset.
type Dataset struct {
	Name          string
	Type          string
	Used          int64
	Available     int64
	Referenced    int64
	Mountpoint    string
	Compression   string
	Compressratio float64
	Mounted       bool
	Recordsize    int64
	Quota         int64
	Reservation   int64
	Origin        string
	Creation      *time.Time
	Encryption    string
}

// JSON output structure shared by zpool get and zfs get commands.
// Both produce: {"pools"|"datasets": {"name": {"properties": {"key": {"value": "..."}}}}}
type propertyValue struct {
	Value string `json:"value"`
}

type zpoolGetOutput struct {
	Pools map[string]zpoolGetPool `json:"pools"`
}

type zpoolGetPool struct {
	Properties map[string]propertyValue `json:"properties"`
}

type zfsGetOutput struct {
	Datasets map[string]zfsGetDataset `json:"datasets"`
}

type zfsGetDataset struct {
	Properties map[string]propertyValue `json:"properties"`
}

// JSON output structure for zpool status -jp (vdev topology).
type zpoolStatusOutput struct {
	Pools map[string]zpoolStatusPool `json:"pools"`
}

type zpoolStatusPool struct {
	Vdevs map[string]zpoolStatusVdev `json:"vdevs"`
}

type zpoolStatusVdev struct {
	Name           string                     `json:"name"`
	VdevType       string                     `json:"vdev_type"`
	State          string                     `json:"state"`
	Path           string                     `json:"path"`
	ReadErrors     string                     `json:"read_errors"`
	WriteErrors    string                     `json:"write_errors"`
	ChecksumErrors string                     `json:"checksum_errors"`
	SlowIOs        string                     `json:"slow_ios"`
	Vdevs          map[string]zpoolStatusVdev `json:"vdevs"`
}

// ParsePools parses the JSON output of `zpool get -jp all`.
// All pool properties (health, guid, size, etc.) come from this single command.
func ParsePools(jsonOutput string) ([]Pool, error) {
	if strings.TrimSpace(jsonOutput) == "" {
		return nil, nil
	}

	var output zpoolGetOutput
	if err := json.Unmarshal([]byte(jsonOutput), &output); err != nil {
		return nil, fmt.Errorf("parsing zpool get JSON: %w", err)
	}

	if len(output.Pools) == 0 {
		return nil, nil
	}

	pools := make([]Pool, 0, len(output.Pools))
	for name, gp := range output.Pools {
		props := gp.Properties
		pool := Pool{
			Name:   name,
			GUID:   propVal(props, "guid"),
			Health: propVal(props, "health"),
		}

		var err error
		pool.Size, err = parseInt(propVal(props, "size"))
		if err != nil {
			return nil, fmt.Errorf("parsing pool %q size: %w", name, err)
		}
		pool.Allocated, err = parseInt(propVal(props, "allocated"))
		if err != nil {
			return nil, fmt.Errorf("parsing pool %q allocated: %w", name, err)
		}
		pool.Free, err = parseInt(propVal(props, "free"))
		if err != nil {
			return nil, fmt.Errorf("parsing pool %q free: %w", name, err)
		}
		pool.Fragmentation, err = parseInt(propVal(props, "fragmentation"))
		if err != nil {
			return nil, fmt.Errorf("parsing pool %q fragmentation: %w", name, err)
		}
		pool.PercentUsed, err = parseInt(propVal(props, "capacity"))
		if err != nil {
			return nil, fmt.Errorf("parsing pool %q capacity: %w", name, err)
		}
		pool.Dedupratio, err = parseRatio(propVal(props, "dedupratio"))
		if err != nil {
			return nil, fmt.Errorf("parsing pool %q dedupratio: %w", name, err)
		}
		pool.Readonly = parseBool(propVal(props, "readonly"))
		pool.Autoexpand = parseBool(propVal(props, "autoexpand"))
		pool.Autoreplace = parseBool(propVal(props, "autoreplace"))
		pool.Autotrim = parseBool(propVal(props, "autotrim"))

		pools = append(pools, pool)
	}

	return pools, nil
}

// ParseDatasets parses the JSON output of `zfs get -jp all`.
// All dataset properties come from this single command.
func ParseDatasets(jsonOutput string) ([]Dataset, error) {
	if strings.TrimSpace(jsonOutput) == "" {
		return nil, nil
	}

	var output zfsGetOutput
	if err := json.Unmarshal([]byte(jsonOutput), &output); err != nil {
		return nil, fmt.Errorf("parsing zfs get JSON: %w", err)
	}

	if len(output.Datasets) == 0 {
		return nil, nil
	}

	datasets := make([]Dataset, 0, len(output.Datasets))
	for name, dsProp := range output.Datasets {
		props := dsProp.Properties

		ds := Dataset{
			Name:        name,
			Type:        strings.ToLower(propVal(props, "type")),
			Mountpoint:  dashToEmpty(propVal(props, "mountpoint")),
			Compression: dashToEmpty(propVal(props, "compression")),
			Origin:      dashToEmpty(propVal(props, "origin")),
			Encryption:  dashToEmpty(propVal(props, "encryption")),
		}

		var err error
		ds.Used, err = parseInt(propVal(props, "used"))
		if err != nil {
			return nil, fmt.Errorf("parsing dataset %q used: %w", name, err)
		}
		ds.Available, err = parseInt(propVal(props, "available"))
		if err != nil {
			return nil, fmt.Errorf("parsing dataset %q available: %w", name, err)
		}
		ds.Referenced, err = parseInt(propVal(props, "referenced"))
		if err != nil {
			return nil, fmt.Errorf("parsing dataset %q referenced: %w", name, err)
		}
		ds.Compressratio, err = parseRatio(propVal(props, "compressratio"))
		if err != nil {
			return nil, fmt.Errorf("parsing dataset %q compressratio: %w", name, err)
		}
		ds.Mounted = parseBool(propVal(props, "mounted"))
		ds.Recordsize, err = parseInt(propVal(props, "recordsize"))
		if err != nil {
			return nil, fmt.Errorf("parsing dataset %q recordsize: %w", name, err)
		}
		ds.Quota, err = parseInt(propVal(props, "quota"))
		if err != nil {
			return nil, fmt.Errorf("parsing dataset %q quota: %w", name, err)
		}
		ds.Reservation, err = parseInt(propVal(props, "reservation"))
		if err != nil {
			return nil, fmt.Errorf("parsing dataset %q reservation: %w", name, err)
		}

		creationStr := propVal(props, "creation")
		if creationStr != "" && creationStr != "-" {
			epoch, err := strconv.ParseInt(creationStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parsing dataset %q creation: %w", name, err)
			}
			t := time.Unix(epoch, 0)
			ds.Creation = &t
		}

		datasets = append(datasets, ds)
	}

	return datasets, nil
}

// ParseProperties parses the JSON output of `zpool get -jp all '<name>'`
// or `zfs get -jp all '<name>'` into a flat key-value map.
// Works for both pool and dataset properties since the JSON structure
// is the same (just keyed under "pools" vs "datasets").
func ParseProperties(jsonOutput string) (map[string]string, error) {
	props := make(map[string]string)
	if strings.TrimSpace(jsonOutput) == "" {
		return props, nil
	}

	// Try parsing as pool properties first, then dataset properties.
	// The JSON has either "pools" or "datasets" as the top-level key.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonOutput), &raw); err != nil {
		return nil, fmt.Errorf("parsing properties JSON: %w", err)
	}

	// Generic structure: {"pools"|"datasets": {"name": {"properties": {"k": {"value": "v"}}}}}
	type propsContainer struct {
		Properties map[string]propertyValue `json:"properties"`
	}

	for key, data := range raw {
		if key == "output_version" {
			continue
		}
		var items map[string]propsContainer
		if err := json.Unmarshal(data, &items); err != nil {
			return nil, fmt.Errorf("parsing properties for key %q: %w", key, err)
		}
		for _, item := range items {
			for k, v := range item.Properties {
				props[k] = v.Value
			}
		}
	}

	return props, nil
}

// ParseVdevs parses the JSON output of `zpool status -jp '<pool>'` and returns
// the top-level vdev groups (skipping the root vdev). Each vdev has nested devices.
func ParseVdevs(statusJSON string) ([]Vdev, error) {
	if strings.TrimSpace(statusJSON) == "" {
		return nil, nil
	}

	var status zpoolStatusOutput
	if err := json.Unmarshal([]byte(statusJSON), &status); err != nil {
		return nil, fmt.Errorf("parsing zpool status JSON: %w", err)
	}

	// There should be exactly one pool in the output.
	for _, pool := range status.Pools {
		// The top-level vdevs map contains one root vdev (named after the pool).
		// Its children are the actual vdev groups (raidz, mirror, etc.).
		for _, rootVdev := range pool.Vdevs {
			if rootVdev.VdevType != "root" {
				continue
			}
			return convertVdevs(rootVdev.Vdevs)
		}
	}

	return nil, nil
}

func convertVdevs(vdevMap map[string]zpoolStatusVdev) ([]Vdev, error) {
	if len(vdevMap) == 0 {
		return nil, nil
	}

	vdevs := make([]Vdev, 0, len(vdevMap))
	for _, sv := range vdevMap {
		v, err := convertVdev(sv)
		if err != nil {
			return nil, err
		}
		vdevs = append(vdevs, v)
	}
	return vdevs, nil
}

func convertVdev(sv zpoolStatusVdev) (Vdev, error) {
	var v Vdev
	v.Name = sv.Name
	v.Type = sv.VdevType
	v.State = sv.State
	v.Path = sv.Path

	var err error
	v.ReadErrors, err = parseInt(sv.ReadErrors)
	if err != nil {
		return v, fmt.Errorf("parsing vdev %q read_errors: %w", sv.Name, err)
	}
	v.WriteErrors, err = parseInt(sv.WriteErrors)
	if err != nil {
		return v, fmt.Errorf("parsing vdev %q write_errors: %w", sv.Name, err)
	}
	v.ChecksumErrors, err = parseInt(sv.ChecksumErrors)
	if err != nil {
		return v, fmt.Errorf("parsing vdev %q checksum_errors: %w", sv.Name, err)
	}
	v.SlowIOs, err = parseInt(sv.SlowIOs)
	if err != nil {
		return v, fmt.Errorf("parsing vdev %q slow_ios: %w", sv.Name, err)
	}

	v.Devices, err = convertVdevs(sv.Vdevs)
	if err != nil {
		return v, err
	}

	return v, nil
}

// propVal extracts a property value string from a properties map, returning "" if not found.
func propVal(props map[string]propertyValue, key string) string {
	if v, ok := props[key]; ok {
		return v.Value
	}
	return ""
}

// parseInt parses a string to int64, treating "-" and "" as 0.
func parseInt(s string) (int64, error) {
	if s == "-" || s == "" {
		return 0, nil
	}
	return strconv.ParseInt(s, 10, 64)
}

// parseRatio parses a ZFS ratio like "1.50x" or "1.50" to float64.
func parseRatio(s string) (float64, error) {
	if s == "-" || s == "" {
		return 0, nil
	}
	s = strings.TrimSuffix(s, "x")
	return strconv.ParseFloat(s, 64)
}

// parseBool parses ZFS boolean values ("on"/"off", "yes"/"no").
func parseBool(s string) bool {
	return s == "on" || s == "yes"
}

// dashToEmpty converts "-" to an empty string.
func dashToEmpty(s string) string {
	if s == "-" {
		return ""
	}
	return s
}
