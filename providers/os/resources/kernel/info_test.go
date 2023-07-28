package kernel

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestParseLinuxKernelArguments(t *testing.T) {
	// testing output of /proc/cmdline

	output := "BOOT_IMAGE=/boot/vmlinuz-3.10.0-1127.19.1.el7.x86_64 root=UUID=ff6cbb65-ccab-489c-91a5-61b9b09e4d49 ro crashkernel=auto console=ttyS0,38400n8 elevator=noop\n"
	args, err := ParseLinuxKernelArguments(strings.NewReader(output))
	require.NoError(t, err)
	assert.Equal(t, "/boot/vmlinuz-3.10.0-1127.19.1.el7.x86_64", args.Path)
	assert.Equal(t, "UUID=ff6cbb65-ccab-489c-91a5-61b9b09e4d49", args.Device)
	assert.Equal(t, map[string]string{"console": "ttyS0,38400n8", "crashkernel": "auto", "elevator": "noop", "ro": ""}, args.Arguments)

	output = "earlyprintk=serial console=ttyS0 console=ttyS1 page_poison=1 vsyscall=emulate panic=1 nospec_store_bypass_disable noibrs noibpb no_stf_barrier mitigations=off\n"
	args, err = ParseLinuxKernelArguments(strings.NewReader(output))
	require.NoError(t, err)
	assert.Equal(t, "", args.Path)
	assert.Equal(t, "", args.Device)
	assert.Equal(t, map[string]string{"console": "ttyS1", "earlyprintk": "serial", "mitigations": "off", "no_stf_barrier": "", "noibpb": "", "noibrs": "", "nospec_store_bypass_disable": "", "page_poison": "1", "panic": "1", "vsyscall": "emulate"}, args.Arguments)
}
