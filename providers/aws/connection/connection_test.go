package connection

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFirstNonEmptyRegion(t *testing.T) {
	tests := []struct {
		name          string
		generalRegion []string
		ec2Region     []string
		expected      []string
	}{
		{
			name:          "EC2Region Non-Empty, GeneralRegion Empty",
			generalRegion: []string{},
			ec2Region:     []string{"us-west-2", "us-east-1"},
			expected:      []string{"us-west-2", "us-east-1"},
		},
		{
			name:          "EC2Region Empty, GeneralRegion Non-Empty",
			generalRegion: []string{"eu-central-1", "eu-west-3"},
			ec2Region:     []string{},
			expected:      []string{"eu-central-1", "eu-west-3"},
		},
		{
			name:          "Both Empty",
			generalRegion: []string{},
			ec2Region:     []string{},
			expected:      []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := firstNonEmptyRegion(tt.generalRegion, tt.ec2Region)
			require.Equal(t, tt.expected, result, "firstNonEmptyRegion did not return the expected result")
		})
	}
}
