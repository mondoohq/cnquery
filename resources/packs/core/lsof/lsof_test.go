package lsof

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestParseLsofWithoutFileDescriptors(t *testing.T) {
	s := `p388
g388
R1
cloginwindow
u501
f10
au
l 
tIPv4
G0x3;0x0
d0x6949610f649739df
o0t0
PUDP
n*:*
`

	processes, err := Parse(strings.NewReader(s))
	require.NoError(t, err)
	assert.Equal(t, 1, len(processes))

	process := processes[0]
	assert.Equal(t, "388", process.PID)
	assert.Equal(t, "loginwindow", process.Command)
	assert.Equal(t, "501", process.UID)
}

func TestParseLsofWithFileDescriptors(t *testing.T) {
	s := `p37224
g678
R678
cHyper Helper
u501
f24
au
l 
tIPv4
G0x10007;0x0
d0x6949610a99d53797
o0t0
PTCP
n10.184.10.188:64647->76.76.21.61:443
TST=ESTABLISHED
TQR=0
TQS=0
f25
au
l 
tIPv4
G0x10007;0x0
d0x6949610a9a1bec8f
o0t0
PTCP
n10.184.10.188:64645->76.76.21.241:443
TST=ESTABLISHED
TQR=0
TQS=0
p38266
g38266
R1
cMail
u501
f68
au
l 
tIPv4
G0x10007;0x1
d0x6949610a99a77797
o0t0
PTCP
n10.184.10.188:59637->142.251.163.108:993
TST=ESTABLISHED
TQR=0
TQS=0
`

	processes, err := Parse(strings.NewReader(s))
	require.NoError(t, err)

	assert.Equal(t, 2, len(processes))

	process := processes[0]
	assert.Equal(t, "37224", process.PID)
	assert.Equal(t, "Hyper Helper", process.Command)
	assert.Equal(t, "501", process.UID)

	assert.Equal(t, 2, len(process.FileDescriptors))

	fd := process.FileDescriptors[0]
	assert.Equal(t, "24", fd.FileDescriptor)
	assert.Equal(t, FileTypeIPv4, fd.Type)
	assert.Equal(t, "10.184.10.188:64647->76.76.21.61:443", fd.Name)

	fd = process.FileDescriptors[1]
	assert.Equal(t, "25", fd.FileDescriptor)
	assert.Equal(t, FileTypeIPv4, fd.Type)
	assert.Equal(t, "10.184.10.188:64645->76.76.21.241:443", fd.Name)
}

func TestParseEmpty(t *testing.T) {
	processes, err := Parse(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(processes) != 0 {
		t.Fatal("Failed parsing empty")
	}
}
