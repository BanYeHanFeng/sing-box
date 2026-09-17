package quicx

import (
	"testing"

	"github.com/sagernet/sing-box/option"
)

func fecBoolPtr(value bool) *bool {
	return &value
}

func TestBuildFECOptionsRecoveredPacketFeedbackDefault(t *testing.T) {
	options := buildFECOptions(nil)
	if options == nil {
		t.Fatal("FEC is enabled by default")
	}
	if !options.RecoveredPacketFeedback {
		t.Fatal("recovered packet feedback is not enabled by default")
	}
}

func TestBuildFECOptionsRecoveredPacketFeedbackExplicit(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		configured *bool
		expected   bool
	}{
		{name: "explicit true", configured: fecBoolPtr(true), expected: true},
		{name: "explicit false", configured: fecBoolPtr(false), expected: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			options := buildFECOptions(&option.QUICXFECOptions{RecoveredPacketFeedback: testCase.configured})
			if options == nil {
				t.Fatal("FEC is enabled by default")
			}
			if options.RecoveredPacketFeedback != testCase.expected {
				t.Fatalf("recovered packet feedback = %v, expected %v", options.RecoveredPacketFeedback, testCase.expected)
			}
		})
	}
}

func TestBuildFECOptionsDisabled(t *testing.T) {
	if options := buildFECOptions(&option.QUICXFECOptions{Enabled: fecBoolPtr(false)}); options != nil {
		t.Fatal("FEC options were built for an explicitly disabled FEC")
	}
}
