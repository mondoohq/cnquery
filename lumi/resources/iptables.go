package resources

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
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

func (i *lumiIptables) id() (string, error) {
	return "iptables", nil
}

func (i *lumiIp6tables) id() (string, error) {
	return "ip6tables", nil
}

func (ie *lumiIptablesEntry) id() (string, error) {
	ln, err := ie.LineNumber()
	if err != nil {
		return "", err
	}
	chain, err := ie.Chain()
	if err != nil {
		return "", err
	}

	return strconv.FormatInt(ln, 10) + chain, nil
}

func (i *lumiIptables) GetOutput() ([]interface{}, error) {
	ipstats := []interface{}{}
	cmd, err := i.Runtime.Motor.Transport.RunCommand("iptables -L OUTPUT -v -n -x --line-numbers")
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}
	if cmd.ExitStatus != 0 {
		outErr, _ := ioutil.ReadAll(cmd.Stderr)
		return nil, errors.New(string(outErr))
	}
	lines := getLines(string(data))
	stats, err := ParseStat(lines, false)
	if err != nil {
		return nil, err
	}
	for _, stat := range stats {
		entry, err := i.Runtime.CreateResource("iptables.entry",
			"lineNumber", stat.LineNumber,
			"packets", stat.Packets,
			"bytes", stat.Bytes,
			"target", stat.Target,
			"protocol", stat.Protocol,
			"opt", stat.Opt,
			"in", stat.Input,
			"out", stat.Output,
			"source", stat.Source,
			"destination", stat.Destination,
			"options", stat.Options,
			"chain", "output",
		)
		if err != nil {
			return nil, err
		}
		ipstats = append(ipstats, entry.(IptablesEntry))
	}
	return ipstats, nil
}

func (i *lumiIptables) GetInput() ([]interface{}, error) {
	ipstats := []interface{}{}
	cmd, err := i.Runtime.Motor.Transport.RunCommand("iptables -L INPUT -v -n -x --line-numbers")
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}
	if cmd.ExitStatus != 0 {
		outErr, _ := ioutil.ReadAll(cmd.Stderr)
		return nil, errors.New(string(outErr))
	}
	lines := getLines(string(data))
	stats, err := ParseStat(lines, false)
	if err != nil {
		return nil, err
	}
	for _, stat := range stats {
		entry, err := i.Runtime.CreateResource("iptables.entry",
			"lineNumber", stat.LineNumber,
			"packets", stat.Packets,
			"bytes", stat.Bytes,
			"target", stat.Target,
			"protocol", stat.Protocol,
			"opt", stat.Opt,
			"in", stat.Input,
			"out", stat.Output,
			"source", stat.Source,
			"destination", stat.Destination,
			"options", stat.Options,
			"chain", "input",
		)
		if err != nil {
			return nil, err
		}
		ipstats = append(ipstats, entry.(IptablesEntry))
	}
	return ipstats, nil
}

func (i *lumiIp6tables) GetOutput() ([]interface{}, error) {
	ipstats := []interface{}{}
	cmd, err := i.Runtime.Motor.Transport.RunCommand("ip6tables -L OUTPUT -v -n -x --line-numbers")
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}
	if cmd.ExitStatus != 0 {
		outErr, _ := ioutil.ReadAll(cmd.Stderr)
		return nil, errors.New(string(outErr))
	}
	lines := getLines(string(data))
	stats, err := ParseStat(lines, true)
	if err != nil {
		return nil, err
	}
	for _, stat := range stats {
		entry, err := i.Runtime.CreateResource("iptables.entry",
			"lineNumber", stat.LineNumber,
			"packets", stat.Packets,
			"bytes", stat.Bytes,
			"target", stat.Target,
			"protocol", stat.Protocol,
			"opt", stat.Opt,
			"in", stat.Input,
			"out", stat.Output,
			"source", stat.Source,
			"destination", stat.Destination,
			"options", stat.Options,
			"chain", "output6",
		)
		if err != nil {
			return nil, err
		}
		ipstats = append(ipstats, entry.(IptablesEntry))
	}
	return ipstats, nil
}

func (i *lumiIp6tables) GetInput() ([]interface{}, error) {
	ipstats := []interface{}{}
	cmd, err := i.Runtime.Motor.Transport.RunCommand("ip6tables -L INPUT -v -n -x --line-numbers")
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	if cmd.ExitStatus != 0 {
		outErr, _ := ioutil.ReadAll(cmd.Stderr)
		return nil, errors.New(string(outErr))
	}
	lines := getLines(string(data))
	stats, err := ParseStat(lines, true)
	if err != nil {
		return nil, err
	}
	for _, stat := range stats {
		entry, err := i.Runtime.CreateResource("iptables.entry",
			"lineNumber", stat.LineNumber,
			"packets", stat.Packets,
			"bytes", stat.Bytes,
			"target", stat.Target,
			"protocol", stat.Protocol,
			"opt", stat.Opt,
			"in", stat.Input,
			"out", stat.Output,
			"source", stat.Source,
			"destination", stat.Destination,
			"options", stat.Options,
			"chain", "input6",
		)
		if err != nil {
			return nil, err
		}
		ipstats = append(ipstats, entry.(IptablesEntry))
	}
	return ipstats, nil
}

//Credit to github.com/coreos/go-iptables for some of the parsing logic
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
		//combine options if they exist
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
