package controller

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
)

func TestRoutingAutomationHandlerType(t *testing.T) {
	assert.Equal(t, model.SystemTaskTypeRoutingAutomation, (routingAutomationHandler{}).Type())
}

func TestRoutingAutomationHandlerEnabledDependsOnMode(t *testing.T) {
	setting := operation_setting.GetRoutingPolicySetting()
	orig := setting.Mode
	t.Cleanup(func() { setting.Mode = orig })

	setting.Mode = operation_setting.RoutingPolicyModeDisabled
	assert.False(t, (routingAutomationHandler{}).Enabled())

	setting.Mode = operation_setting.RoutingPolicyModeObserve
	assert.True(t, (routingAutomationHandler{}).Enabled())
}

func TestRoutingAutomationHandlerInterval(t *testing.T) {
	assert.Equal(t, time.Minute, (routingAutomationHandler{}).Interval())
}

func TestRoutingAutomationHandlerIntervalDoesNotDependOnSlowScanSetting(t *testing.T) {
	setting := operation_setting.GetRoutingPolicySetting()
	origSlow := setting.SlowChannelPolicy
	t.Cleanup(func() { setting.SlowChannelPolicy = origSlow })

	setting.SlowChannelPolicy.ScanIntervalSeconds = 999
	assert.Equal(t, time.Minute, (routingAutomationHandler{}).Interval())
}
