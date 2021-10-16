package awsec2ebs

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/tj/assert"
)

func TestNewVolumeAttachmentLoc(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	loc1 := newVolumeAttachmentLoc()
	rand.Seed(time.Now().UnixNano())
	loc2 := newVolumeAttachmentLoc()
	assert.NotEqual(t, loc1, loc2)
	assert.Equal(t, len(loc1), 9)
	assert.Equal(t, strings.HasPrefix(loc1, "/dev/xvd"), true)
}
