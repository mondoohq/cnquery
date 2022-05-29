package awsec2ebs

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewVolumeAttachmentLoc(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	loc1 := newVolumeAttachmentLoc()
	require.Equal(t, len(loc1), 8)
	require.Equal(t, strings.HasPrefix(loc1, "/dev/sd"), true)
}
