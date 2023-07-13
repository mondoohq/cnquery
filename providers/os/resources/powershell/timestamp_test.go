package powershell

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPSJsonTimestamp(t *testing.T) {

	timestamp := PSJsonTimestamp("\\/Date(1599609600000)\\/")
	assert.Equal(t, time.Unix(1599609600, 0).Unix(), timestamp.Unix())
}
