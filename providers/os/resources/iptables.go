// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
)

// Stat represents a structured statistic entry.
type Stat struct {
	LineNumber  int64
	Packets     int64
	Bytes       int64
	Target      string
	Protocol    string
	Opt         string
	Input       string
	Output      string
	Source      string
	Destination string
	Options     string
}

func (ie *mqlIptablesEntry) id() (string, error) {
	return strconv.FormatInt(ie.LineNumber.Data, 10) + ie.Chain.Data, nil
}

func (i *mqlIptables) output() ([]interface{}, error) {
	conn := i.MqlRuntime.Connection.(shared.Connection)

	ipstats := []interface{}{}
	cmd, err := conn.RunCommand("iptables -L OUTPUT -v -n -x --line-numbers")
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}
	if cmd.ExitStatus != 0 {
		outErr, _ := io.ReadAll(cmd.Stderr)
		return nil, errors.New(string(outErr))
	}
	lines := getLines(string(data))
	stats, err := ParseStat(lines, false)
	if err != nil {
		return nil, err
	}
	for _, stat := range stats {
		entry, err := CreateResource(i.MqlRuntime, "iptables.entry", map[string]*llx.RawData{
			"lineNumber":  llx.IntData(stat.LineNumber),
			"packets":     llx.IntData(stat.Packets),
			"bytes":       llx.IntData(stat.Bytes),
			"target":      llx.StringData(stat.Target),
			"protocol":    llx.StringData(stat.Protocol),
			"opt":         llx.StringData(stat.Opt),
			"in":          llx.StringData(stat.Input),
			"out":         llx.StringData(stat.Output),
			"source":      llx.StringData(stat.Source),
			"destination": llx.StringData(stat.Destination),
			"options":     llx.StringData(stat.Options),
			"chain":       llx.StringData("output"),
		})
		if err != nil {
			return nil, err
		}
		ipstats = append(ipstats, entry.(*mqlIptablesEntry))
	}
	return ipstats, nil
}

func (i *mqlIptables) input() ([]interface{}, error) {
	conn := i.MqlRuntime.Connection.(shared.Connection)

	ipstats := []interface{}{}
	cmd, err := conn.RunCommand("iptables -L INPUT -v -n -x --line-numbers")
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}
	if cmd.ExitStatus != 0 {
		outErr, _ := io.ReadAll(cmd.Stderr)
		return nil, errors.New(string(outErr))
	}
	lines := getLines(string(data))
	stats, err := ParseStat(lines, false)
	if err != nil {
		return nil, err
	}
	for _, stat := range stats {
		entry, err := CreateResource(i.MqlRuntime, "iptables.entry", map[string]*llx.RawData{
			"lineNumber":  llx.IntData(stat.LineNumber),
			"packets":     llx.IntData(stat.Packets),
			"bytes":       llx.IntData(stat.Bytes),
			"target":      llx.StringData(stat.Target),
			"protocol":    llx.StringData(stat.Protocol),
			"opt":         llx.StringData(stat.Opt),
			"in":          llx.StringData(stat.Input),
			"out":         llx.StringData(stat.Output),
			"source":      llx.StringData(stat.Source),
			"destination": llx.StringData(stat.Destination),
			"options":     llx.StringData(stat.Options),
			"chain":       llx.StringData("input"),
		})
		if err != nil {
			return nil, err
		}
		ipstats = append(ipstats, entry.(*mqlIptablesEntry))
	}
	return ipstats, nil
}

func (i *mqlIp6tables) output() ([]interface{}, error) {
	conn := i.MqlRuntime.Connection.(shared.Connection)

	ipstats := []interface{}{}
	cmd, err := conn.RunCommand("ip6tables -L OUTPUT -v -n -x --line-numbers")
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}
	if cmd.ExitStatus != 0 {
		outErr, _ := io.ReadAll(cmd.Stderr)
		return nil, errors.New(string(outErr))
	}
	lines := getLines(string(data))
	stats, err := ParseStat(lines, true)
	if err != nil {
		return nil, err
	}
	for _, stat := range stats {
		entry, err := CreateResource(i.MqlRuntime, "iptables.entry", map[string]*llx.RawData{
			"lineNumber":  llx.IntData(stat.LineNumber),
			"packets":     llx.IntData(stat.Packets),
			"bytes":       llx.IntData(stat.Bytes),
			"target":      llx.StringData(stat.Target),
			"protocol":    llx.StringData(stat.Protocol),
			"opt":         llx.StringData(stat.Opt),
			"in":          llx.StringData(stat.Input),
			"out":         llx.StringData(stat.Output),
			"source":      llx.StringData(stat.Source),
			"destination": llx.StringData(stat.Destination),
			"options":     llx.StringData(stat.Options),
			"chain":       llx.StringData("output6"),
		})
		if err != nil {
			return nil, err
		}
		ipstats = append(ipstats, entry.(*mqlIptablesEntry))
	}
	return ipstats, nil
}

func (i *mqlIp6tables) input() ([]interface{}, error) {
	conn := i.MqlRuntime.Connection.(shared.Connection)

	ipstats := []interface{}{}
	cmd, err := conn.RunCommand("ip6tables -L INPUT -v -n -x --line-numbers")
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	if cmd.ExitStatus != 0 {
		outErr, _ := io.ReadAll(cmd.Stderr)
		return nil, errors.New(string(outErr))
	}
	lines := getLines(string(data))
	stats, err := ParseStat(lines, true)
	if err != nil {
		return nil, err
	}
	for _, stat := range stats {
		entry, err := CreateResource(i.MqlRuntime, "iptables.entry", map[string]*llx.RawData{
			"lineNumber":  llx.IntData(stat.LineNumber),
			"packets":     llx.IntData(stat.Packets),
			"bytes":       llx.IntData(stat.Bytes),
			"target":      llx.StringData(stat.Target),
			"protocol":    llx.StringData(stat.Protocol),
			"opt":         llx.StringData(stat.Opt),
			"in":          llx.StringData(stat.Input),
			"out":         llx.StringData(stat.Output),
			"source":      llx.StringData(stat.Source),
			"destination": llx.StringData(stat.Destination),
			"options":     llx.StringData(stat.Options),
			"chain":       llx.StringData("input6"),
		})
		if err != nil {
			return nil, err
		}
		ipstats = append(ipstats, entry.(*mqlIptablesEntry))
	}
	return ipstats, nil
}

// Credit to github.com/coreos/go-iptables for some of the parsing logic
func getLines(data string) []string {
	rules := strings.Split(data, "\n")

	// strip trailing newline
	if len(rules) > 0 && rules[len(rules)-1] == "" {
		rules = rules[:len(rules)-1]
	}

	return rules
}

func ParseStat(lines []string, ipv6 bool) ([]Stat, error) {
	entries := []Stat{}
	for i, line := range lines {
		// Skip over chain name and field header
		if i < 2 {
			continue
		}

		// Fields:
		// 0=linenumber 1=pkts 2=bytes 3=target 4=prot 5=opt 6=in 7=out 8=source 9=destination 10=options
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)

		// The ip6tables verbose output cannot be naively split due to the default "opt"
		// field containing 2 single spaces.
		if ipv6 {
			// Check if field 7 is "out" or "source" address
			dest := fields[7]
			ip, _, _ := net.ParseCIDR(dest)
			if ip == nil {
				ip = net.ParseIP(dest)
			}

			// If we detected a CIDR or IP, the "opt" field is empty.. insert it.
			if ip != nil {
				f := []string{}
				f = append(f, fields[:5]...)
				f = append(f, "  ") // Empty "opt" field for ip6tables
				f = append(f, fields[5:]...)
				fields = f
			}
		}
		ln, err := strconv.ParseInt(fields[0], 0, 64)
		if err != nil {
			return entries, fmt.Errorf(err.Error(), "could not parse line number")
		}
		pkts, err := strconv.ParseInt(fields[1], 0, 64)
		if err != nil {
			return entries, fmt.Errorf(err.Error(), "could not parse packets")
		}
		bts, err := strconv.ParseInt(fields[2], 0, 64)
		if err != nil {
			return entries, fmt.Errorf(err.Error(), "could not parse bytes")
		}
		var opts string
		// combine options if they exist
		if len(fields) > 10 {
			o := fields[10:]
			opts = strings.Join(o, " ")
		}
		entry := Stat{
			LineNumber:  ln,
			Packets:     pkts,
			Bytes:       bts,
			Target:      fields[3],
			Protocol:    fields[4],
			Opt:         fields[5],
			Input:       fields[6],
			Output:      fields[7],
			Source:      fields[8],
			Destination: fields[9],
			Options:     opts,
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
