package service

import (
	"context"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/routingpolicy"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

var routingPolicyProbeExecutor func(ctx context.Context, channel *model.Channel, probe operation_setting.RoutingPolicyProbe) bool

type RoutingChannelRole string

const (
	RoutingChannelRoleSubscription RoutingChannelRole = "subscription"
	RoutingChannelRolePaygo        RoutingChannelRole = "paygo"
	RoutingChannelRoleOther        RoutingChannelRole = "other"
)

func init() {
	model.RegisterRoutingPolicyHooks(model.RoutingPolicyHooks{
		IsEnabled:     routingpolicy.IsEnabled,
		IsEnforceMode: routingpolicy.IsEnforceMode,
		IsChannelHealthy: func(channelID int, now time.Time) model.RoutingPolicyDecision {
			decision := routingpolicy.IsChannelHealthy(channelID, now)
			return model.RoutingPolicyDecision{
				Healthy: decision.Healthy,
				Reason:  decision.Reason,
			}
		},
		LogSkipDecision: routingpolicy.LogSkipDecision,
		RecordFailOpen:  routingpolicy.RecordFailOpen,
	})
}

func normalizeRoutingTag(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func containsRoutingValue(list []string, value string) bool {
	normalized := normalizeRoutingTag(value)
	for _, item := range list {
		if normalizeRoutingTag(item) == normalized {
			return true
		}
	}
	return false
}

func DetectRoutingChannelRole(channel *model.Channel) RoutingChannelRole {
	if channel == nil {
		return RoutingChannelRoleOther
	}
	setting := operation_setting.GetRoutingPolicySetting()
	tag := ""
	if channel.Tag != nil {
		tag = *channel.Tag
	}
	if containsRoutingValue(setting.RoleDetection.SubscriptionTags, tag) {
		return RoutingChannelRoleSubscription
	}
	if containsRoutingValue(setting.RoleDetection.PaygoTags, tag) {
		return RoutingChannelRolePaygo
	}
	name := strings.TrimSpace(channel.Name)
	for _, pattern := range setting.RoleDetection.SubscriptionNamePatterns {
		if pattern != "" && strings.Contains(name, pattern) {
			return RoutingChannelRoleSubscription
		}
	}
	if channel.Status == common.ChannelStatusEnabled {
		return RoutingChannelRolePaygo
	}
	return RoutingChannelRoleOther
}

func RoutingStatusCodeMappingForChannel(channel *model.Channel) map[string]string {
	if DetectRoutingChannelRole(channel) != RoutingChannelRoleSubscription {
		return nil
	}
	setting := operation_setting.GetRoutingPolicySetting()
	mapping := make(map[string]string, len(setting.SubscriptionPolicy.StatusCodeMapping))
	for key, value := range setting.SubscriptionPolicy.StatusCodeMapping {
		mapping[key] = value
	}
	return mapping
}

func RoutingStatusCodeMappingStringForChannel(channel *model.Channel) string {
	mapping := RoutingStatusCodeMappingForChannel(channel)
	if len(mapping) == 0 {
		return ""
	}
	raw, err := common.Marshal(mapping)
	if err != nil {
		return ""
	}
	return string(raw)
}

func ChannelSupportsRequestPath(channel *model.Channel, requestPath string) bool {
	if channel == nil {
		return false
	}
	if requestPath == "" || channel.Type != constant.ChannelTypeAdvancedCustom {
		return true
	}
	config := channel.GetOtherSettings().AdvancedCustom
	return config != nil && config.SupportsPath(requestPath)
}

func IsRoutingPolicyEnabled() bool {
	return routingpolicy.IsEnabled()
}

func IsRoutingPolicyEnforceMode() bool {
	return routingpolicy.IsEnforceMode()
}

func IsChannelRuntimeHealthy(channelID int, now time.Time) routingpolicy.Decision {
	return routingpolicy.IsChannelHealthy(channelID, now)
}

func LogRoutingSkipDecision(ctx context.Context, channelID int, reason string, enforced bool) {
	routingpolicy.LogSkipDecision(ctx, channelID, reason, enforced)
}

func RecordRoutingFailOpen(ctx context.Context, candidateCount int) {
	routingpolicy.RecordFailOpen(ctx, candidateCount)
}

func MarkChannelRoutingFailure(channelID int, statusCode int, reason string, cooldown time.Duration) {
	routingpolicy.MarkFailure(channelID, statusCode, reason, cooldown)
}

func MarkChannelRoutingSuccess(channelID int) {
	routingpolicy.MarkSuccess(channelID)
}

func GetRoutingSummary() routingpolicy.Summary {
	return routingpolicy.GetSummary()
}

func UpdateRoutingSummary(summary routingpolicy.Summary) {
	routingpolicy.UpdateSummary(summary)
}

func RunRoutingAutomationOnce(ctx context.Context) routingpolicy.Summary {
	return routingpolicy.RunAutomationOnce(
		ctx,
		func(channel *model.Channel) routingpolicy.ChannelRole {
			return routingpolicy.ChannelRole(DetectRoutingChannelRole(channel))
		},
		routingpolicy.AutomationHooks{
			ClearAffinityByChannel: ClearChannelAffinityCacheByChannelID,
			ProbeChannel: func(ctx context.Context, channel *model.Channel, probe operation_setting.RoutingPolicyProbe) bool {
				if routingPolicyProbeExecutor == nil {
					return false
				}
				return routingPolicyProbeExecutor(ctx, channel, probe)
			},
		},
	)
}

func DescribeRoutingState(channelID int) string {
	return routingpolicy.DescribeState(channelID)
}

func SnapshotRoutingState() map[int]routingpolicy.HealthState {
	return routingpolicy.SnapshotState()
}

func RegisterRoutingPolicyProbeExecutor(executor func(ctx context.Context, channel *model.Channel, probe operation_setting.RoutingPolicyProbe) bool) {
	routingPolicyProbeExecutor = executor
}
