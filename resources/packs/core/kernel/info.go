package kernel

import (
	"errors"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
)

var LINUX_KERNEL_ARGUMENTS_REGEX = regexp.MustCompile(`(?:^BOOT_IMAGE=([^\s]*)\s)?(?:root=([^\s]*)\s)?(.*)`)

type LinuxKernelArguments struct {
	Path      string
	Device    string
	Arguments map[string]string
}

func ParseLinuxKernelArguments(r io.Reader) (LinuxKernelArguments, error) {
	res := LinuxKernelArguments{
		Arguments: map[string]string{},
	}

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return res, err
	}

	m := LINUX_KERNEL_ARGUMENTS_REGEX.FindStringSubmatch(string(data))

	if len(m) > 0 {
		res.Path = m[1]
		res.Device = m[2]

		args := m[3]
		keypairs := strings.Split(args, " ")

		for i := range keypairs {
			keypair := keypairs[i]
			vals := strings.Split(keypair, "=")

			key := vals[0]
			value := ""
			if len(vals) > 1 {
				value = vals[1]
			}

			res.Arguments[key] = value
		}
	}

	return res, nil
}

// kernel version includes the kernel version, build data, buildhost, compiler version and an optional build date
func ParseLinuxKernelVersion(r io.Reader) (string, error) {

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}

	values := strings.Split(string(data), " ")
	if len(values) > 2 && values[1] == "version" {
		return values[2], nil
	}

	return "", errors.New("cannot determine kernel version")
}
