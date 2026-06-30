package model

import (
	"context"
	"time"
)

type RoutingPolicyDecision struct {
	Healthy bool
	Reason  string
}

type RoutingPolicyHooks struct {
	IsEnabled       func() bool
	IsEnforceMode   func() bool
	IsChannelHealthy func(channelID int, now time.Time) RoutingPolicyDecision
	LogSkipDecision  func(ctx context.Context, channelID int, reason string, enforced bool)
	RecordFailOpen   func(ctx context.Context, candidateCount int)
}

var routingPolicyHooks = RoutingPolicyHooks{
	IsEnabled: func() bool {
		return false
	},
	IsEnforceMode: func() bool {
		return false
	},
	IsChannelHealthy: func(channelID int, now time.Time) RoutingPolicyDecision {
		return RoutingPolicyDecision{Healthy: true}
	},
	LogSkipDecision: func(ctx context.Context, channelID int, reason string, enforced bool) {},
	RecordFailOpen:  func(ctx context.Context, candidateCount int) {},
}

func RegisterRoutingPolicyHooks(hooks RoutingPolicyHooks) {
	if hooks.IsEnabled != nil {
		routingPolicyHooks.IsEnabled = hooks.IsEnabled
	}
	if hooks.IsEnforceMode != nil {
		routingPolicyHooks.IsEnforceMode = hooks.IsEnforceMode
	}
	if hooks.IsChannelHealthy != nil {
		routingPolicyHooks.IsChannelHealthy = hooks.IsChannelHealthy
	}
	if hooks.LogSkipDecision != nil {
		routingPolicyHooks.LogSkipDecision = hooks.LogSkipDecision
	}
	if hooks.RecordFailOpen != nil {
		routingPolicyHooks.RecordFailOpen = hooks.RecordFailOpen
	}
}
