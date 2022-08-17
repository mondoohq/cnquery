package resources

import (
	"errors"
	"io/ioutil"
	"strings"
)

func (f *lumiSocketstats) id() (string, error) {
	return "socketstats", nil
}

func (f *lumiSocketstats) GetOpenPorts() ([]interface{}, error) {
	osProvider, err := osProvider(f.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	cmd, err := osProvider.RunCommand("ss -4tuln")
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
	stats := strings.Split(string(data), "\n")
	// strip trailing newline
	if len(stats) > 0 && stats[len(stats)-1] == "" {
		stats = stats[:len(stats)-1]
	}

	ports := []interface{}{}
	for i, stat := range stats {
		if i < 1 {
			continue
		}
		fields := strings.Fields(stat)
		state := fields[1]
		laport := fields[4]
		lap := strings.Split(laport, ":")
		la := lap[0]
		port := lap[1]
		if err != nil {
			return nil, err
		}

		// If we are listening on a non-local host then add to open ports list
		if state == "LISTEN" && la != "127.0.0.1" {
			ports = append(ports, port)
		}
	}
	return ports, nil
}
