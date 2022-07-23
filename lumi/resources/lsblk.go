package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
)

func (l *lumiLsblk) id() (string, error) {
	return "lsblk", nil
}

func (l *lumiLsblk) GetList() ([]interface{}, error) {
	cmd, err := l.MotorRuntime.Motor.Transport.RunCommand("lsblk --json --fs")
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
	blockEntries, err := parseBlockEntries(data)
	if err != nil {
		return nil, err
	}

	lumiBlockEntries := []interface{}{}
	for i := range blockEntries.Blockdevices {
		d := blockEntries.Blockdevices[i]
		for i := range d.Children {
			entry := d.Children[i]
			entry.Mountpoints = append(entry.Mountpoints, entry.Mountpoint)
			lumiLsblkEntry, err := l.MotorRuntime.CreateResource("lsblk.entry",
				"name", entry.Name,
				"fstype", entry.Fstype,
				"label", entry.Label,
				"uuid", entry.Uuid,
				"mountpoints", entry.Mountpoints,
			)
			if err != nil {
				return nil, err
			}
			lumiBlockEntries = append(lumiBlockEntries, lumiLsblkEntry)
		}
	}
	return lumiBlockEntries, nil
}

func parseBlockEntries(data []byte) (blockdevices, error) {
	blockEntries := blockdevices{}
	if err := json.Unmarshal(data, &blockEntries); err != nil {
		return blockEntries, err
	}
	return blockEntries, nil
}

func (l *lumiLsblkEntry) id() (string, error) {
	name, err := l.Name()
	if err != nil {
		return "", err
	}
	fstype, err := l.Fstype()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s", name, fstype), nil
}

type blockdevices struct {
	Blockdevices []blockdevice `json:"blockdevices,omitempty"`
}

type blockdevice struct {
	Name        string        `json:"name,omitempty"`
	Fstype      string        `json:"fstype,omitempty"`
	Label       string        `json:"label,omitempty"`
	Uuid        string        `json:"uuid,omitempty"`
	Mountpoints []interface{} `json:"mountpoints,omitempty"`
	Mountpoint  string        `json:"mountpoint,omitempty"`
	Children    []blockdevice `json:"children,omitempty"`
}
