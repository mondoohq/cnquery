package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToPlatformIdDetectors(t *testing.T) {
	t.Run("aliases", func(t *testing.T) {
		assert.ElementsMatch(
			t,
			[]PlatformIdDetector{AWSEc2Detector, HostnameDetector},
			ToPlatformIdDetectors([]string{"awsec2", "hostname"}))

		assert.ElementsMatch(
			t,
			[]PlatformIdDetector{AWSEc2Detector, HostnameDetector},
			ToPlatformIdDetectors([]string{"aws-ec2", "hostname"}))
	})
}
