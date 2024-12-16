// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bufio"
	"context"
	"github.com/Ullaakut/nmap/v3"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/types"
	"io"
	"strings"
	"time"
)

type nmapVersion struct {
	Version               string
	Platform              string
	CompiledWith          []string
	CompiledWithout       []string
	AvailableNsockEngines []string
}

func parseNmapVersionOutput(r io.Reader) nmapVersion {
	version := nmapVersion{
		CompiledWith:          []string{},
		CompiledWithout:       []string{},
		AvailableNsockEngines: []string{},
	}
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Nmap version") {
			version.Version = strings.TrimSpace(strings.Split(line, " ")[2])
			continue
		}
		m := strings.Split(line, ":")
		if len(m) != 2 {
			continue
		}
		key := strings.TrimSpace(m[0])
		value := strings.TrimSpace(m[1])
		if value == "" {
			continue
		}
		switch key {
		case "Platform":
			version.Platform = value
		case "Compiled with":
			version.CompiledWith = strings.Split(value, " ")
		case "Compiled without":
			version.CompiledWithout = strings.Split(value, " ")
		case "Available nsock engines":
			version.AvailableNsockEngines = strings.Split(value, " ")
		}
	}

	return version
}

func (r *mqlNmap) version() (*mqlNmapVersionInformation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// retrieve nmap version
	scanner, err := nmap.NewScanner(
		ctx,
		nmap.WithBinaryPath("/opt/homebrew/bin/nmap"),
		// we can ignore the deprecation warning since the -V flag is not supported by the nmap library
		nmap.WithCustomArguments("-V"),
	)
	if err != nil {
		return nil, err
	}

	// NOTE: -V does not return xml output so run does not parse the output
	// Therefore we cannot trust the err return value
	results, _, _ := scanner.Run()

	info := parseNmapVersionOutput(results.ToReader())

	runtime := r.MqlRuntime
	resource, err := CreateResource(runtime, "nmap.versionInformation", map[string]*llx.RawData{
		"__id":            llx.StringData("nmap.versionInformation"),
		"version":         llx.StringData(info.Version),
		"platform":        llx.StringData(info.Platform),
		"compiledWith":    llx.ArrayData(convert.SliceAnyToInterface(info.CompiledWith), types.String),
		"compiledWithout": llx.ArrayData(convert.SliceAnyToInterface(info.CompiledWithout), types.String),
		"nsockEngines":    llx.ArrayData(convert.SliceAnyToInterface(info.AvailableNsockEngines), types.String),
	})
	return resource.(*mqlNmapVersionInformation), err
}
