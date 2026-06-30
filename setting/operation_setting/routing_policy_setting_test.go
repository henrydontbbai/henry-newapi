package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRoutingPolicySetting_Defaults(t *testing.T) {
	orig := routingPolicySetting
	t.Cleanup(func() { routingPolicySetting = orig })

	routingPolicySetting = RoutingPolicySetting{}

	setting := GetRoutingPolicySetting()
	require.NotNil(t, setting)
	assert.Equal(t, RoutingPolicyModeObserve, setting.Mode)
	assert.Equal(t, []string{"subscription"}, setting.RoleDetection.SubscriptionTags)
	assert.Equal(t, []string{"paygo"}, setting.RoleDetection.PaygoTags)
	assert.Equal(t, []string{"\u8ba2\u9605"}, setting.RoleDetection.SubscriptionNamePatterns)
	assert.Equal(t, map[string]string{"429": "500", "403": "500"}, setting.SubscriptionPolicy.StatusCodeMapping)
	assert.False(t, setting.ProbePolicy.ActiveProbeEnabled)
	assert.True(t, setting.SlowChannelPolicy.SummaryEnabled)
	assert.False(t, setting.SlowChannelPolicy.AutoDisableEnabled)
	assert.False(t, setting.SlowChannelPolicy.WeightDegradeEnabled)
}

func TestGetRoutingPolicySetting_InvalidModeFallsBackToObserve(t *testing.T) {
	orig := routingPolicySetting
	t.Cleanup(func() { routingPolicySetting = orig })

	routingPolicySetting = RoutingPolicySetting{
		Mode: "weird-mode",
	}

	setting := GetRoutingPolicySetting()
	require.NotNil(t, setting)
	assert.Equal(t, RoutingPolicyModeObserve, setting.Mode)
}

func TestGetRoutingPolicySetting_NormalizesReservedAndProbeFields(t *testing.T) {
	orig := routingPolicySetting
	t.Cleanup(func() { routingPolicySetting = orig })

	routingPolicySetting = RoutingPolicySetting{
		Mode: RoutingPolicyModeObserve,
	}

	setting := GetRoutingPolicySetting()
	require.NotNil(t, setting)
	assert.Equal(t, 900, setting.ProbePolicy.ActiveProbeIntervalSeconds)
	assert.Equal(t, "gpt-5.5", setting.ProbePolicy.ProbeModel)
	assert.Equal(t, "openai-response-compact", setting.ProbePolicy.ProbeEndpointType)
	assert.Equal(t, 600, setting.ProbePolicy.ProbeRetrySeconds)
	assert.Equal(t, 20, setting.PaygoHardFailurePolicy.ForceThreshold)
	assert.Equal(t, 30, setting.SlowChannelPolicy.ConfirmWindowMinutes)
	assert.False(t, setting.SlowChannelPolicy.WeightDegradeEnabled)
	assert.Equal(t, 1800, setting.SlowChannelPolicy.AutoDisableCooldownSeconds)
	assert.Equal(t, 4, setting.SubscriptionPolicy.TransientUpstreamMinEnabledSubscriptions)
}

func TestGetRoutingPolicySetting_AllowsSlowSummaryToBeDisabled(t *testing.T) {
	orig := routingPolicySetting
	t.Cleanup(func() { routingPolicySetting = orig })

	routingPolicySetting = RoutingPolicySetting{
		Mode: RoutingPolicyModeObserve,
		SlowChannelPolicy: RoutingPolicySlowChannel{
			SummaryEnabled: false,
			WindowMinutes:  10,
		},
	}

	setting := GetRoutingPolicySetting()
	require.NotNil(t, setting)
	assert.False(t, setting.SlowChannelPolicy.SummaryEnabled)
}
