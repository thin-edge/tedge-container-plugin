package tedge

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Target(t *testing.T) {
	target := NewTarget("", "device/main//")
	assert.Equal(t, "te/device/main//", target.Topic())
}

func Test_TargetServiceTopic(t *testing.T) {
	target := NewTarget("", "device/main//")
	assert.Equal(t, "te/device/main/service/foo", target.Service("foo").Topic())
}

func Test_TargetChildServiceTopic(t *testing.T) {
	target := NewTarget("", "device/child01/service/other")
	assert.Equal(t, "te/device/child01/service/foo", target.Service("foo").Topic())
}

func Test_TargetExternalID(t *testing.T) {
	target := &Target{
		RootPrefix:    "te",
		TopicID:       "device/main//",
		CloudIdentity: "device0001",
	}
	assert.Equal(t, "device0001", target.ExternalID())

	target2 := &Target{
		RootPrefix:    "te",
		TopicID:       "device/child01//",
		CloudIdentity: "device0001",
	}
	assert.Equal(t, "device0001:device:child01", target2.ExternalID())

	target3 := target2.Service("foo")
	assert.Equal(t, "device0001:device:child01:service:foo", target3.ExternalID())
}
