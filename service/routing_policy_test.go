package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/routingpolicy"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetRoutingRuntimeStateForTest(t *testing.T) {
	t.Helper()
	routingpolicy.ResetForTest()
	RegisterRoutingPolicyProbeExecutor(nil)
}

func waitForRoutingHoldUntil(t *testing.T, holdUntil int64) {
	t.Helper()
	require.Positive(t, holdUntil)
	timeout := time.Until(time.Unix(holdUntil, 0)) + 2*time.Second
	if timeout < 100*time.Millisecond {
		timeout = 100 * time.Millisecond
	}
	require.Eventually(t, func() bool {
		return time.Now().Unix() >= holdUntil
	}, timeout, 10*time.Millisecond)
}

func TestDetectRoutingChannelRole_TagWinsOverName(t *testing.T) {
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origTags := append([]string(nil), setting.RoleDetection.SubscriptionTags...)
	origPaygo := append([]string(nil), setting.RoleDetection.PaygoTags...)
	origPatterns := append([]string(nil), setting.RoleDetection.SubscriptionNamePatterns...)
	t.Cleanup(func() {
		setting.RoleDetection.SubscriptionTags = origTags
		setting.RoleDetection.PaygoTags = origPaygo
		setting.RoleDetection.SubscriptionNamePatterns = origPatterns
	})

	setting.RoleDetection.SubscriptionTags = []string{"subscription"}
	setting.RoleDetection.PaygoTags = []string{"paygo"}
	setting.RoleDetection.SubscriptionNamePatterns = []string{"\u8ba2\u9605"}

	tag := "paygo"
	channel := &model.Channel{
		Name:   "\u8fd9\u662f\u8ba2\u9605\u7ebf\u8def",
		Status: common.ChannelStatusEnabled,
		Tag:    &tag,
	}

	assert.Equal(t, RoutingChannelRolePaygo, DetectRoutingChannelRole(channel))
}

func TestDetectRoutingChannelRole_NameFallback(t *testing.T) {
	resetRoutingRuntimeStateForTest(t)

	channel := &model.Channel{
		Name:   "VIP\u8ba2\u9605\u4e3b\u7ebf\u8def",
		Status: common.ChannelStatusEnabled,
	}

	assert.Equal(t, RoutingChannelRoleSubscription, DetectRoutingChannelRole(channel))
}

func TestDetectRoutingChannelRole_EnabledUnclassifiedFallsToPaygo(t *testing.T) {
	resetRoutingRuntimeStateForTest(t)

	channel := &model.Channel{
		Name:   "fallback-main",
		Status: common.ChannelStatusEnabled,
	}

	assert.Equal(t, RoutingChannelRolePaygo, DetectRoutingChannelRole(channel))
}

func TestDetectRoutingChannelRole_DisabledUnclassifiedIsOther(t *testing.T) {
	resetRoutingRuntimeStateForTest(t)

	channel := &model.Channel{
		Name:   "fallback-main",
		Status: common.ChannelStatusAutoDisabled,
	}

	assert.Equal(t, RoutingChannelRoleOther, DetectRoutingChannelRole(channel))
}

func TestRoutingStatusCodeMappingForSubscriptionChannel(t *testing.T) {
	resetRoutingRuntimeStateForTest(t)

	tag := "subscription"
	channel := &model.Channel{
		Name:   "sub-line",
		Status: common.ChannelStatusEnabled,
		Tag:    &tag,
	}

	mapping := RoutingStatusCodeMappingForChannel(channel)
	require.NotNil(t, mapping)
	assert.Equal(t, "500", mapping["429"])
	assert.Equal(t, "500", mapping["403"])
}

func TestChannelSupportsRequestPathForAdvancedCustom(t *testing.T) {
	resetRoutingRuntimeStateForTest(t)

	channel := &model.Channel{
		Type: constant.ChannelTypeAdvancedCustom,
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{
		AdvancedCustom: &dto.AdvancedCustomConfig{
			Routes: []dto.AdvancedCustomRoute{
				{IncomingPath: "/v1/chat/completions"},
			},
		},
	})

	assert.True(t, ChannelSupportsRequestPath(channel, "/v1/chat/completions"))
	assert.False(t, ChannelSupportsRequestPath(channel, "/v1/responses"))
	assert.True(t, ChannelSupportsRequestPath(channel, ""))
	assert.True(t, ChannelSupportsRequestPath(&model.Channel{Type: constant.ChannelTypeOpenAI}, "/v1/responses"))
}

func TestChannelRuntimeHealthCooldown(t *testing.T) {
	resetRoutingRuntimeStateForTest(t)

	MarkChannelRoutingFailure(101, 429, "rate_limit", time.Minute)
	decision := IsChannelRuntimeHealthy(101, time.Now())
	assert.False(t, decision.Healthy)
	assert.Equal(t, "rate_limit", decision.Reason)

	MarkChannelRoutingSuccess(101)
	decision = IsChannelRuntimeHealthy(101, time.Now())
	assert.True(t, decision.Healthy)
}

func TestRunRoutingAutomationOnceSummarizesSlowSubscriptionChannels(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origSlow := setting.SlowChannelPolicy
	origProbe := setting.ProbePolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.SlowChannelPolicy = origSlow
		setting.ProbePolicy = origProbe
	})

	setting.Mode = operation_setting.RoutingPolicyModeObserve
	setting.SlowChannelPolicy.SummaryEnabled = true
	setting.SlowChannelPolicy.ScanIntervalSeconds = 1
	setting.SlowChannelPolicy.WindowMinutes = 60
	setting.SlowChannelPolicy.ConfirmWindowMinutes = 60
	setting.SlowChannelPolicy.MinRequests = 3
	setting.SlowChannelPolicy.P95Seconds = 20
	setting.SlowChannelPolicy.SlowRequestSeconds = 15
	setting.SlowChannelPolicy.SlowRatioPercent = 50
	setting.SlowChannelPolicy.AffinityClearEnabled = true

	now := common.GetTimestamp()
	subTag := "subscription"
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3001,
		Name:   "\u8ba2\u9605\u6162\u7ebf\u8def",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)

	for i, useTime := range []int{25, 30, 40} {
		require.NoError(t, model.LOG_DB.Create(&model.Log{
			UserId:    1,
			CreatedAt: now - int64(i),
			Type:      model.LogTypeConsume,
			Content:   fmt.Sprintf("request-%d", i),
			ModelName: "gpt-5",
			ChannelId: 3001,
			UseTime:   useTime,
			Other:     `{"request_path":"/v1/chat/completions"}`,
		}).Error)
	}

	summary := RunRoutingAutomationOnce(context.Background())
	require.Equal(t, 1, summary.SlowChannelCount)
	require.Equal(t, []int{3001}, summary.SlowChannels)
	assert.Equal(t, "MIXED_DEGRADED", summary.RoutingMode)
	assert.Equal(t, "slow_subscription_channels", summary.ReasonCode)
	assert.GreaterOrEqual(t, summary.MaxSubscriptionP95, 25)
	assert.NotZero(t, summary.LastSlowScanAt)

	state := SnapshotRoutingState()[3001]
	assert.Equal(t, int64(0), state.LastAffinityClearAt)
}

func TestRunRoutingAutomationOnceAutoDisablesSlowSubscriptionChannelWhenEnabled(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origSlow := setting.SlowChannelPolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.SlowChannelPolicy = origSlow
	})

	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	setting.SlowChannelPolicy.SummaryEnabled = true
	setting.SlowChannelPolicy.ScanIntervalSeconds = 1
	setting.SlowChannelPolicy.WindowMinutes = 60
	setting.SlowChannelPolicy.MinRequests = 3
	setting.SlowChannelPolicy.P95Seconds = 20
	setting.SlowChannelPolicy.SlowRequestSeconds = 15
	setting.SlowChannelPolicy.SlowRatioPercent = 50
	setting.SlowChannelPolicy.AffinityClearEnabled = true
	setting.SlowChannelPolicy.AutoDisableEnabled = true
	setting.SlowChannelPolicy.AutoDisableMinP95Seconds = 20
	setting.SlowChannelPolicy.AutoDisableHoldSeconds = 120
	setting.SlowChannelPolicy.MinEnabledSubscriptions = 1
	setting.SlowChannelPolicy.AutoDisableMaxPerRun = 1
	setting.ProbePolicy.ActiveProbeEnabled = true
	RegisterRoutingPolicyProbeExecutor(func(ctx context.Context, channel *model.Channel, probe operation_setting.RoutingPolicyProbe) bool {
		return true
	})

	now := common.GetTimestamp()
	subTag := "subscription"
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3005,
		Name:   "订阅慢线路-auto-disable",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3006,
		Name:   "订阅正常线路-floor-guard",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)

	for i, useTime := range []int{25, 30, 40} {
		require.NoError(t, model.LOG_DB.Create(&model.Log{
			UserId:    1,
			CreatedAt: now - int64(i),
			Type:      model.LogTypeConsume,
			Content:   fmt.Sprintf("slow-request-%d", i),
			ModelName: "gpt-5",
			ChannelId: 3005,
			UseTime:   useTime,
			Other:     `{"request_path":"/v1/chat/completions"}`,
		}).Error)
	}

	summary := RunRoutingAutomationOnce(context.Background())
	require.Equal(t, 1, summary.SlowChannelCount)
	require.Equal(t, []int{3005}, summary.SlowChannels)
	assert.NotEmpty(t, summary.LastSubscriptionDisableAction)

	channelAfter, err := model.GetChannelById(3005, true)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusAutoDisabled, channelAfter.Status)

	state := SnapshotRoutingState()[3005]
	assert.Equal(t, "slow_channel", state.Reason)
	assert.Greater(t, state.CooldownUntil, now)
}

func TestRunRoutingAutomationOnceDoesNotDisableSlowChannelWhenProbeDisabled(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origSlow := setting.SlowChannelPolicy
	origProbe := setting.ProbePolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.SlowChannelPolicy = origSlow
		setting.ProbePolicy = origProbe
	})

	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	setting.SlowChannelPolicy.SummaryEnabled = true
	setting.SlowChannelPolicy.WindowMinutes = 60
	setting.SlowChannelPolicy.MinRequests = 3
	setting.SlowChannelPolicy.P95Seconds = 20
	setting.SlowChannelPolicy.SlowRequestSeconds = 15
	setting.SlowChannelPolicy.SlowRatioPercent = 50
	setting.SlowChannelPolicy.AffinityClearEnabled = true
	setting.SlowChannelPolicy.AutoDisableEnabled = true
	setting.SlowChannelPolicy.AutoDisableMinP95Seconds = 20
	setting.SlowChannelPolicy.AutoDisableHoldSeconds = 120
	setting.SlowChannelPolicy.MinEnabledSubscriptions = 1
	setting.SlowChannelPolicy.AutoDisableMaxPerRun = 1
	setting.ProbePolicy.ActiveProbeEnabled = false

	now := common.GetTimestamp()
	subTag := "subscription"
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3008,
		Name:   "slow-probe-disabled",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3009,
		Name:   "slow-probe-disabled-floor",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)

	for i, useTime := range []int{25, 30, 40} {
		require.NoError(t, model.LOG_DB.Create(&model.Log{
			UserId:    1,
			CreatedAt: now - int64(i),
			Type:      model.LogTypeConsume,
			Content:   fmt.Sprintf("slow-probe-disabled-%d", i),
			ModelName: "gpt-5",
			ChannelId: 3008,
			UseTime:   useTime,
			Other:     `{"request_path":"/v1/chat/completions"}`,
		}).Error)
	}

	summary := RunRoutingAutomationOnce(context.Background())
	require.Equal(t, 1, summary.SlowChannelCount)
	assert.Empty(t, summary.LastSubscriptionDisableAction)

	channelAfter, err := model.GetChannelById(3008, true)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusEnabled, channelAfter.Status)

	state := SnapshotRoutingState()[3008]
	assert.Equal(t, int64(0), state.CooldownUntil)
}

func TestRunRoutingAutomationOnceTemporarilyDisablesPaygoHardFailureChannels(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origHard := setting.PaygoHardFailurePolicy
	origProbe := setting.ProbePolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.PaygoHardFailurePolicy = origHard
		setting.ProbePolicy = origProbe
	})

	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	setting.PaygoHardFailurePolicy.Enabled = true
	setting.PaygoHardFailurePolicy.WindowMinutes = 60
	setting.PaygoHardFailurePolicy.Threshold = 2
	setting.PaygoHardFailurePolicy.MaxPerRun = 1
	setting.PaygoHardFailurePolicy.RetrySeconds = 120
	setting.PaygoHardFailurePolicy.RestorePriority = -5
	setting.PaygoHardFailurePolicy.RestoreWeight = 1
	setting.ProbePolicy.ActiveProbeEnabled = true
	RegisterRoutingPolicyProbeExecutor(func(ctx context.Context, channel *model.Channel, probe operation_setting.RoutingPolicyProbe) bool {
		return true
	})

	now := common.GetTimestamp()
	paygoTag := "paygo"
	priority := int64(10)
	weight := uint(20)
	channel := &model.Channel{
		Id:       3002,
		Name:     "paygo-hard-failure",
		Status:   common.ChannelStatusEnabled,
		Tag:      &paygoTag,
		Priority: &priority,
		Weight:   &weight,
		Group:    "default",
		Models:   "gpt-5",
	}
	require.NoError(t, model.DB.Create(channel).Error)
	require.NoError(t, channel.AddAbilities(nil))

	for i := 0; i < 2; i++ {
		require.NoError(t, model.LOG_DB.Create(&model.Log{
			UserId:    1,
			CreatedAt: now - int64(i),
			Type:      model.LogTypeError,
			Content:   "auth failed",
			ModelName: "gpt-5",
			ChannelId: 3002,
			Other:     `{"error_code":"channel:invalid_key","status_code":401,"request_path":"/v1/chat/completions"}`,
		}).Error)
	}

	summary := RunRoutingAutomationOnce(context.Background())
	require.Equal(t, 1, summary.PaygoHardFailureCount)
	require.Equal(t, []int{3002}, summary.PaygoHardFailureChannels)

	channelAfter, err := model.GetChannelById(3002, true)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusAutoDisabled, channelAfter.Status)

	state := SnapshotRoutingState()[3002]
	assert.Equal(t, "paygo_hard_failure", state.Reason)
	assert.Greater(t, state.CooldownUntil, now)
}

func TestRunRoutingAutomationOnceObserveModeDoesNotDisablePaygoHardFailureChannels(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origHard := setting.PaygoHardFailurePolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.PaygoHardFailurePolicy = origHard
	})

	setting.Mode = operation_setting.RoutingPolicyModeObserve
	setting.PaygoHardFailurePolicy.Enabled = true
	setting.PaygoHardFailurePolicy.WindowMinutes = 60
	setting.PaygoHardFailurePolicy.Threshold = 1
	setting.PaygoHardFailurePolicy.MaxPerRun = 1
	setting.PaygoHardFailurePolicy.RetrySeconds = 120

	now := common.GetTimestamp()
	paygoTag := "paygo"
	channel := &model.Channel{
		Id:     3004,
		Name:   "paygo-observe-only",
		Status: common.ChannelStatusEnabled,
		Tag:    &paygoTag,
	}
	require.NoError(t, model.DB.Create(channel).Error)

	require.NoError(t, model.LOG_DB.Create(&model.Log{
		UserId:    1,
		CreatedAt: now,
		Type:      model.LogTypeError,
		Content:   "auth failed",
		ModelName: "gpt-5",
		ChannelId: 3004,
		Other:     `{"error_code":"channel:invalid_key","status_code":401,"request_path":"/v1/chat/completions"}`,
	}).Error)

	summary := RunRoutingAutomationOnce(context.Background())
	require.Equal(t, 1, summary.PaygoHardFailureCount)
	require.Equal(t, []int{3004}, summary.PaygoHardFailureChannels)
	assert.Empty(t, summary.LastSubscriptionDisableAction)

	channelAfter, err := model.GetChannelById(3004, true)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusEnabled, channelAfter.Status)

	state := SnapshotRoutingState()[3004]
	assert.Equal(t, int64(0), state.CooldownUntil)
	assert.Empty(t, state.Reason)
}

func TestRunRoutingAutomationOnceDoesNotDisablePaygoHardFailureWhenProbeDisabled(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origHard := setting.PaygoHardFailurePolicy
	origProbe := setting.ProbePolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.PaygoHardFailurePolicy = origHard
		setting.ProbePolicy = origProbe
	})

	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	setting.PaygoHardFailurePolicy.Enabled = true
	setting.PaygoHardFailurePolicy.WindowMinutes = 60
	setting.PaygoHardFailurePolicy.Threshold = 1
	setting.PaygoHardFailurePolicy.MaxPerRun = 1
	setting.PaygoHardFailurePolicy.RetrySeconds = 120
	setting.ProbePolicy.ActiveProbeEnabled = false

	now := common.GetTimestamp()
	paygoTag := "paygo"
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3007,
		Name:   "paygo-probe-disabled",
		Status: common.ChannelStatusEnabled,
		Tag:    &paygoTag,
	}).Error)
	require.NoError(t, model.LOG_DB.Create(&model.Log{
		UserId:    1,
		CreatedAt: now,
		Type:      model.LogTypeError,
		Content:   "auth failed",
		ModelName: "gpt-5",
		ChannelId: 3007,
		Other:     `{"error_code":"channel:invalid_key","status_code":401,"request_path":"/v1/chat/completions"}`,
	}).Error)

	summary := RunRoutingAutomationOnce(context.Background())
	require.Equal(t, 1, summary.PaygoHardFailureCount)
	require.Equal(t, []int{3007}, summary.PaygoHardFailureChannels)

	channelAfter, err := model.GetChannelById(3007, true)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusEnabled, channelAfter.Status)

	state := SnapshotRoutingState()[3007]
	assert.Equal(t, int64(0), state.CooldownUntil)
	assert.Empty(t, state.Reason)
}

func TestRunRoutingAutomationOnceRestoresPaygoHardFailureChannels(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origHard := setting.PaygoHardFailurePolicy
	origProbe := setting.ProbePolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.PaygoHardFailurePolicy = origHard
		setting.ProbePolicy = origProbe
	})

	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	setting.PaygoHardFailurePolicy.Enabled = true
	setting.PaygoHardFailurePolicy.WindowMinutes = 60
	setting.PaygoHardFailurePolicy.Threshold = 1
	setting.PaygoHardFailurePolicy.RetrySeconds = 1
	setting.PaygoHardFailurePolicy.RestorePriority = -7
	setting.PaygoHardFailurePolicy.RestoreWeight = 2
	setting.ProbePolicy.ActiveProbeEnabled = true
	setting.ProbePolicy.ProbeRetrySeconds = 1
	setting.ProbePolicy.ActiveProbeIntervalSeconds = 1

	now := common.GetTimestamp()
	paygoTag := "paygo"
	priority := int64(10)
	weight := uint(20)
	channel := &model.Channel{
		Id:       3003,
		Name:     "paygo-restore",
		Status:   common.ChannelStatusEnabled,
		Tag:      &paygoTag,
		Priority: &priority,
		Weight:   &weight,
		Group:    "default",
		Models:   "gpt-5",
	}
	require.NoError(t, model.DB.Create(channel).Error)
	require.NoError(t, channel.AddAbilities(nil))

	require.NoError(t, model.LOG_DB.Create(&model.Log{
		UserId:    1,
		CreatedAt: now,
		Type:      model.LogTypeError,
		Content:   "auth failed",
		ModelName: "gpt-5",
		ChannelId: 3003,
		Other:     `{"error_code":"channel:invalid_key","status_code":401,"request_path":"/v1/chat/completions"}`,
	}).Error)

	firstSummary := RunRoutingAutomationOnce(context.Background())
	assert.Equal(t, 1, firstSummary.PaygoHardFailureCount)
	assert.Equal(t, []int{3003}, firstSummary.PaygoHardFailureChannels)
	firstState := SnapshotRoutingState()[3003]

	require.NoError(t, model.LOG_DB.Where("channel_id = ?", 3003).Delete(&model.Log{}).Error)
	RegisterRoutingPolicyProbeExecutor(func(ctx context.Context, channel *model.Channel, probe operation_setting.RoutingPolicyProbe) bool {
		return channel != nil && channel.Id == 3003
	})

	waitForRoutingHoldUntil(t, firstState.RestoreHoldUntil)
	summary := RunRoutingAutomationOnce(context.Background())

	channelAfter, err := model.GetChannelById(3003, true)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusEnabled, channelAfter.Status)
	require.NotNil(t, channelAfter.Priority)
	require.NotNil(t, channelAfter.Weight)
	assert.Equal(t, int64(-7), *channelAfter.Priority)
	assert.Equal(t, uint(2), *channelAfter.Weight)
	assert.NotEmpty(t, summary.LastPaygoRestoreAction)
	assert.NotZero(t, summary.LastProbeAt)
}

func TestRunRoutingAutomationOnceStartsPaygoNudgeWithoutProbeRecovery(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origNudge := setting.PaygoNudgePolicy
	origProbe := setting.ProbePolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.PaygoNudgePolicy = origNudge
		setting.ProbePolicy = origProbe
	})

	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	setting.PaygoNudgePolicy.Enabled = true
	setting.PaygoNudgePolicy.HoldSeconds = 60
	setting.PaygoNudgePolicy.CooldownSeconds = 1
	setting.PaygoNudgePolicy.RecentWindowMinutes = 60
	setting.PaygoNudgePolicy.MinSubscriptions = 1
	setting.PaygoNudgePolicy.MinRecentSubscriptionSuccess = 1
	setting.PaygoNudgePolicy.MaxRecentErrors = 0
	setting.PaygoNudgePolicy.MaxTransientErrors = 0
	setting.PaygoNudgePolicy.RequireNoSlowChannels = false
	setting.ProbePolicy.ActiveProbeEnabled = false

	now := common.GetTimestamp()
	subTag := "subscription"
	paygoTag := "paygo"

	subChannel := &model.Channel{Id: 3010, Name: "sub-ok", Status: common.ChannelStatusEnabled, Tag: &subTag}
	paygoChannel := &model.Channel{Id: 3011, Name: "paygo-hold", Status: common.ChannelStatusEnabled, Tag: &paygoTag}
	require.NoError(t, model.DB.Create(subChannel).Error)
	require.NoError(t, model.DB.Create(paygoChannel).Error)
	require.NoError(t, model.LOG_DB.Create(&model.Log{
		UserId:    1,
		CreatedAt: now,
		Type:      model.LogTypeConsume,
		Content:   "subscription success",
		ModelName: "gpt-5",
		ChannelId: 3010,
		UseTime:   2,
		Other:     `{"request_path":"/v1/chat/completions"}`,
	}).Error)

	summary := RunRoutingAutomationOnce(context.Background())
	assert.NotZero(t, summary.LastNudgeAt)

	channelAfter, err := model.GetChannelById(3011, true)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusAutoDisabled, channelAfter.Status)
}

func TestRunRoutingAutomationOnceTemporarilyDisablesSubscriptionTransientChannels(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origSubscription := setting.SubscriptionPolicy
	origProbe := setting.ProbePolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.SubscriptionPolicy = origSubscription
		setting.ProbePolicy = origProbe
	})

	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	setting.SubscriptionPolicy.ErrorWindowMinutes = 60
	setting.SubscriptionPolicy.TransientUpstreamAffinityClearEnabled = true
	setting.SubscriptionPolicy.TransientUpstreamDisableEnabled = true
	setting.SubscriptionPolicy.TransientUpstreamDisableThreshold = 2
	setting.SubscriptionPolicy.TransientUpstreamMinEnabledSubscriptions = 1
	setting.ProbePolicy.ActiveProbeEnabled = true
	setting.ProbePolicy.ProbeRetrySeconds = 60
	RegisterRoutingPolicyProbeExecutor(func(ctx context.Context, channel *model.Channel, probe operation_setting.RoutingPolicyProbe) bool {
		return true
	})

	now := common.GetTimestamp()
	subTag := "subscription"
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3020,
		Name:   "subscription-transient-bad",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3021,
		Name:   "subscription-transient-floor-guard",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)

	for i := 0; i < 2; i++ {
		require.NoError(t, model.LOG_DB.Create(&model.Log{
			UserId:    1,
			CreatedAt: now - int64(i),
			Type:      model.LogTypeError,
			Content:   "upstream timeout",
			ModelName: "gpt-5",
			ChannelId: 3020,
			Other:     `{"error_code":"upstream_error","status_code":504,"request_path":"/v1/chat/completions"}`,
		}).Error)
	}

	summary := RunRoutingAutomationOnce(context.Background())
	assert.NotEmpty(t, summary.LastSubscriptionDisableAction)

	channelAfter, err := model.GetChannelById(3020, true)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusAutoDisabled, channelAfter.Status)

	state := SnapshotRoutingState()[3020]
	assert.Equal(t, "subscription_transient", state.Reason)
	assert.Greater(t, state.CooldownUntil, now)
}

func TestRunRoutingAutomationOnceObserveModeDoesNotDisableSubscriptionTransientChannels(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origSubscription := setting.SubscriptionPolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.SubscriptionPolicy = origSubscription
	})

	setting.Mode = operation_setting.RoutingPolicyModeObserve
	setting.SubscriptionPolicy.ErrorWindowMinutes = 60
	setting.SubscriptionPolicy.TransientUpstreamAffinityClearEnabled = true
	setting.SubscriptionPolicy.TransientUpstreamDisableEnabled = true
	setting.SubscriptionPolicy.TransientUpstreamDisableThreshold = 1
	setting.SubscriptionPolicy.TransientUpstreamMinEnabledSubscriptions = 1

	now := common.GetTimestamp()
	subTag := "subscription"
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3022,
		Name:   "subscription-transient-observe",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)

	require.NoError(t, model.LOG_DB.Create(&model.Log{
		UserId:    1,
		CreatedAt: now,
		Type:      model.LogTypeError,
		Content:   "gateway timeout",
		ModelName: "gpt-5",
		ChannelId: 3022,
		Other:     `{"error_code":"timeout","status_code":504,"request_path":"/v1/chat/completions"}`,
	}).Error)

	summary := RunRoutingAutomationOnce(context.Background())
	assert.Empty(t, summary.LastSubscriptionDisableAction)

	channelAfter, err := model.GetChannelById(3022, true)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusEnabled, channelAfter.Status)
}

func TestRunRoutingAutomationOnceDoesNotDisableSubscriptionTransientWhenProbeDisabled(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origSubscription := setting.SubscriptionPolicy
	origProbe := setting.ProbePolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.SubscriptionPolicy = origSubscription
		setting.ProbePolicy = origProbe
	})

	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	setting.SubscriptionPolicy.ErrorWindowMinutes = 60
	setting.SubscriptionPolicy.TransientUpstreamAffinityClearEnabled = true
	setting.SubscriptionPolicy.TransientUpstreamDisableEnabled = true
	setting.SubscriptionPolicy.TransientUpstreamDisableThreshold = 1
	setting.SubscriptionPolicy.TransientUpstreamMinEnabledSubscriptions = 1
	setting.ProbePolicy.ActiveProbeEnabled = false

	now := common.GetTimestamp()
	subTag := "subscription"
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3026,
		Name:   "subscription-transient-probe-disabled",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3027,
		Name:   "subscription-transient-probe-disabled-floor",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)
	require.NoError(t, model.LOG_DB.Create(&model.Log{
		UserId:    1,
		CreatedAt: now,
		Type:      model.LogTypeError,
		Content:   "gateway timeout",
		ModelName: "gpt-5",
		ChannelId: 3026,
		Other:     `{"error_code":"timeout","status_code":504,"request_path":"/v1/chat/completions"}`,
	}).Error)

	summary := RunRoutingAutomationOnce(context.Background())
	assert.Empty(t, summary.LastSubscriptionDisableAction)

	channelAfter, err := model.GetChannelById(3026, true)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusEnabled, channelAfter.Status)

	state := SnapshotRoutingState()[3026]
	assert.Equal(t, "subscription_transient", state.Reason)
	assert.Equal(t, int64(0), state.CooldownUntil)
}

func TestRunRoutingAutomationOnceProbeRestoresSubscriptionTransientChannels(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origSubscription := setting.SubscriptionPolicy
	origProbe := setting.ProbePolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.SubscriptionPolicy = origSubscription
		setting.ProbePolicy = origProbe
	})

	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	setting.SubscriptionPolicy.ErrorWindowMinutes = 60
	setting.SubscriptionPolicy.TransientUpstreamAffinityClearEnabled = true
	setting.SubscriptionPolicy.TransientUpstreamDisableEnabled = true
	setting.SubscriptionPolicy.TransientUpstreamDisableThreshold = 1
	setting.SubscriptionPolicy.TransientUpstreamMinEnabledSubscriptions = 1
	setting.ProbePolicy.ActiveProbeEnabled = true
	setting.ProbePolicy.ProbeRetrySeconds = 1
	setting.ProbePolicy.ActiveProbeIntervalSeconds = 1

	now := common.GetTimestamp()
	subTag := "subscription"
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3023,
		Name:   "subscription-transient-restore",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3024,
		Name:   "subscription-transient-restore-floor",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)

	require.NoError(t, model.LOG_DB.Create(&model.Log{
		UserId:    1,
		CreatedAt: now,
		Type:      model.LogTypeError,
		Content:   "gateway timeout",
		ModelName: "gpt-5",
		ChannelId: 3023,
		Other:     `{"error_code":"timeout","status_code":504,"request_path":"/v1/chat/completions"}`,
	}).Error)

	firstSummary := RunRoutingAutomationOnce(context.Background())
	assert.NotEmpty(t, firstSummary.LastSubscriptionDisableAction)
	firstState := SnapshotRoutingState()[3023]

	RegisterRoutingPolicyProbeExecutor(func(ctx context.Context, channel *model.Channel, probe operation_setting.RoutingPolicyProbe) bool {
		return channel != nil && channel.Id == 3023
	})

	waitForRoutingHoldUntil(t, firstState.RestoreHoldUntil)
	summary := RunRoutingAutomationOnce(context.Background())

	channelAfter, err := model.GetChannelById(3023, true)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusEnabled, channelAfter.Status)
	assert.NotZero(t, summary.LastProbeAt)

	state := SnapshotRoutingState()[3023]
	assert.Empty(t, state.Reason)
	assert.Equal(t, int64(0), state.CooldownUntil)
}

func TestRunRoutingAutomationOnceProbeFailureKeepsDisabledChannel(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origHard := setting.PaygoHardFailurePolicy
	origProbe := setting.ProbePolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.PaygoHardFailurePolicy = origHard
		setting.ProbePolicy = origProbe
	})

	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	setting.PaygoHardFailurePolicy.Enabled = true
	setting.PaygoHardFailurePolicy.WindowMinutes = 60
	setting.PaygoHardFailurePolicy.Threshold = 1
	setting.PaygoHardFailurePolicy.RetrySeconds = 1
	setting.ProbePolicy.ActiveProbeEnabled = true
	setting.ProbePolicy.ProbeRetrySeconds = 1
	setting.ProbePolicy.ActiveProbeIntervalSeconds = 1

	now := common.GetTimestamp()
	paygoTag := "paygo"
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3025,
		Name:   "paygo-probe-fail",
		Status: common.ChannelStatusEnabled,
		Tag:    &paygoTag,
		Group:  "default",
		Models: "gpt-5",
	}).Error)

	require.NoError(t, model.LOG_DB.Create(&model.Log{
		UserId:    1,
		CreatedAt: now,
		Type:      model.LogTypeError,
		Content:   "auth failed",
		ModelName: "gpt-5",
		ChannelId: 3025,
		Other:     `{"error_code":"channel:invalid_key","status_code":401,"request_path":"/v1/chat/completions"}`,
	}).Error)

	firstSummary := RunRoutingAutomationOnce(context.Background())
	require.Equal(t, 1, firstSummary.PaygoHardFailureCount)
	require.Equal(t, []int{3025}, firstSummary.PaygoHardFailureChannels)
	firstState := SnapshotRoutingState()[3025]

	RegisterRoutingPolicyProbeExecutor(func(ctx context.Context, channel *model.Channel, probe operation_setting.RoutingPolicyProbe) bool {
		return false
	})

	waitForRoutingHoldUntil(t, firstState.RestoreHoldUntil)
	summary := RunRoutingAutomationOnce(context.Background())

	channelAfter, err := model.GetChannelById(3025, true)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusAutoDisabled, channelAfter.Status)
	assert.Empty(t, summary.LastPaygoRestoreAction)
	assert.NotZero(t, summary.LastProbeAt)
}

func TestRunRoutingAutomationOnceClearsPerRunActionSummary(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	t.Cleanup(func() {
		setting.Mode = origMode
	})

	setting.Mode = operation_setting.RoutingPolicyModeObserve
	UpdateRoutingSummary(routingpolicy.Summary{
		RoutingMode:                   "MIXED_DEGRADED",
		LastSubscriptionDisableAction: "channel=1 reason=old",
		LastPaygoRestoreAction:        "channel=2 priority=0 weight=1",
	})

	summary := RunRoutingAutomationOnce(context.Background())

	assert.Empty(t, summary.LastSubscriptionDisableAction)
	assert.Empty(t, summary.LastPaygoRestoreAction)
}

func TestRunRoutingAutomationOnceProbeFailureMarksPaygoAsProbeFailedHold(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origHard := setting.PaygoHardFailurePolicy
	origProbe := setting.ProbePolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.PaygoHardFailurePolicy = origHard
		setting.ProbePolicy = origProbe
	})

	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	setting.PaygoHardFailurePolicy.Enabled = true
	setting.PaygoHardFailurePolicy.WindowMinutes = 60
	setting.PaygoHardFailurePolicy.Threshold = 1
	setting.PaygoHardFailurePolicy.RetrySeconds = 1
	setting.ProbePolicy.ActiveProbeEnabled = true
	setting.ProbePolicy.ProbeRetrySeconds = 1
	setting.ProbePolicy.ActiveProbeIntervalSeconds = 2

	now := common.GetTimestamp()
	paygoTag := "paygo"
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3030,
		Name:   "paygo-probe-failed-hold",
		Status: common.ChannelStatusEnabled,
		Tag:    &paygoTag,
		Group:  "default",
		Models: "gpt-5",
	}).Error)
	require.NoError(t, model.LOG_DB.Create(&model.Log{
		UserId:    1,
		CreatedAt: now,
		Type:      model.LogTypeError,
		Content:   "auth failed",
		ModelName: "gpt-5",
		ChannelId: 3030,
		Other:     `{"error_code":"channel:invalid_key","status_code":401,"request_path":"/v1/chat/completions"}`,
	}).Error)

	RegisterRoutingPolicyProbeExecutor(func(ctx context.Context, channel *model.Channel, probe operation_setting.RoutingPolicyProbe) bool {
		return false
	})

	firstSummary := RunRoutingAutomationOnce(context.Background())
	require.Equal(t, 1, firstSummary.PaygoHardFailureCount)

	firstState := SnapshotRoutingState()[3030]
	assert.Equal(t, "waiting_probe", firstState.RestorePhase)
	assert.Equal(t, "paygo_hard_failure", firstState.RestoreReason)
	assert.Greater(t, firstState.RestoreHoldUntil, firstState.LastErrorAt)

	waitForRoutingHoldUntil(t, firstState.RestoreHoldUntil)
	summary := RunRoutingAutomationOnce(context.Background())

	channelAfter, err := model.GetChannelById(3030, true)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusAutoDisabled, channelAfter.Status)
	assert.NotZero(t, summary.LastProbeAt)
	assert.NotZero(t, summary.NextProbeAt)
	assert.Contains(t, summary.LastProbeAction, "probe_failed_hold")

	state := SnapshotRoutingState()[3030]
	assert.Equal(t, "probe_failed_hold", state.RestorePhase)
	assert.Equal(t, "paygo_hard_failure", state.RestoreReason)
	assert.Equal(t, "failed", state.LastProbeResult)
	assert.NotZero(t, state.LastProbeAt)
	assert.GreaterOrEqual(t, state.RestoreHoldUntil-state.LastProbeAt, int64(2))
	assert.Equal(t, state.RestoreHoldUntil, summary.NextProbeAt)
}

func TestRunRoutingAutomationOnceProbeSuccessMarksPaygoAsRestored(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origHard := setting.PaygoHardFailurePolicy
	origProbe := setting.ProbePolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.PaygoHardFailurePolicy = origHard
		setting.ProbePolicy = origProbe
	})

	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	setting.PaygoHardFailurePolicy.Enabled = true
	setting.PaygoHardFailurePolicy.WindowMinutes = 60
	setting.PaygoHardFailurePolicy.Threshold = 1
	setting.PaygoHardFailurePolicy.RetrySeconds = 1
	setting.PaygoHardFailurePolicy.RestorePriority = -7
	setting.PaygoHardFailurePolicy.RestoreWeight = 2
	setting.ProbePolicy.ActiveProbeEnabled = true
	setting.ProbePolicy.ProbeRetrySeconds = 1
	setting.ProbePolicy.ActiveProbeIntervalSeconds = 1

	now := common.GetTimestamp()
	paygoTag := "paygo"
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3031,
		Name:   "paygo-probe-restored-state",
		Status: common.ChannelStatusEnabled,
		Tag:    &paygoTag,
		Group:  "default",
		Models: "gpt-5",
	}).Error)
	require.NoError(t, model.LOG_DB.Create(&model.Log{
		UserId:    1,
		CreatedAt: now,
		Type:      model.LogTypeError,
		Content:   "auth failed",
		ModelName: "gpt-5",
		ChannelId: 3031,
		Other:     `{"error_code":"channel:invalid_key","status_code":401,"request_path":"/v1/chat/completions"}`,
	}).Error)

	RegisterRoutingPolicyProbeExecutor(func(ctx context.Context, channel *model.Channel, probe operation_setting.RoutingPolicyProbe) bool {
		return true
	})

	firstSummary := RunRoutingAutomationOnce(context.Background())
	require.Equal(t, 1, firstSummary.PaygoHardFailureCount)

	firstState := SnapshotRoutingState()[3031]
	waitForRoutingHoldUntil(t, firstState.RestoreHoldUntil)
	summary := RunRoutingAutomationOnce(context.Background())

	channelAfter, err := model.GetChannelById(3031, true)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusEnabled, channelAfter.Status)
	assert.NotEmpty(t, summary.LastPaygoRestoreAction)
	assert.NotZero(t, summary.LastProbeAt)
	assert.Contains(t, summary.LastProbeAction, "restored")

	state := SnapshotRoutingState()[3031]
	assert.Equal(t, "restored", state.RestorePhase)
	assert.Equal(t, "paygo_hard_failure", state.RestoreReason)
	assert.Equal(t, "success", state.LastProbeResult)
	assert.NotZero(t, state.LastProbeAt)
	assert.Equal(t, int64(0), state.RestoreHoldUntil)
	assert.Equal(t, int64(0), state.CooldownUntil)
}

func TestRunRoutingAutomationOnceProbeFailureMarksSubscriptionAsProbeFailedHold(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origSubscription := setting.SubscriptionPolicy
	origProbe := setting.ProbePolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.SubscriptionPolicy = origSubscription
		setting.ProbePolicy = origProbe
	})

	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	setting.SubscriptionPolicy.ErrorWindowMinutes = 60
	setting.SubscriptionPolicy.TransientUpstreamAffinityClearEnabled = true
	setting.SubscriptionPolicy.TransientUpstreamDisableEnabled = true
	setting.SubscriptionPolicy.TransientUpstreamDisableThreshold = 1
	setting.SubscriptionPolicy.TransientUpstreamMinEnabledSubscriptions = 1
	setting.ProbePolicy.ActiveProbeEnabled = true
	setting.ProbePolicy.ProbeRetrySeconds = 1
	setting.ProbePolicy.ActiveProbeIntervalSeconds = 2

	now := common.GetTimestamp()
	subTag := "subscription"
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3032,
		Name:   "subscription-transient-probe-failed-hold",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3033,
		Name:   "subscription-transient-probe-failed-floor",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)
	require.NoError(t, model.LOG_DB.Create(&model.Log{
		UserId:    1,
		CreatedAt: now,
		Type:      model.LogTypeError,
		Content:   "gateway timeout",
		ModelName: "gpt-5",
		ChannelId: 3032,
		Other:     `{"error_code":"timeout","status_code":504,"request_path":"/v1/chat/completions"}`,
	}).Error)

	RegisterRoutingPolicyProbeExecutor(func(ctx context.Context, channel *model.Channel, probe operation_setting.RoutingPolicyProbe) bool {
		return false
	})

	firstSummary := RunRoutingAutomationOnce(context.Background())
	assert.NotEmpty(t, firstSummary.LastSubscriptionDisableAction)

	firstState := SnapshotRoutingState()[3032]
	assert.Equal(t, "waiting_probe", firstState.RestorePhase)
	assert.Equal(t, "subscription_transient", firstState.RestoreReason)
	assert.Greater(t, firstState.RestoreHoldUntil, firstState.LastErrorAt)

	waitForRoutingHoldUntil(t, firstState.RestoreHoldUntil)
	summary := RunRoutingAutomationOnce(context.Background())

	channelAfter, err := model.GetChannelById(3032, true)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusAutoDisabled, channelAfter.Status)
	assert.NotZero(t, summary.LastProbeAt)
	assert.NotZero(t, summary.NextProbeAt)
	assert.Contains(t, summary.LastProbeAction, "probe_failed_hold")

	state := SnapshotRoutingState()[3032]
	assert.Equal(t, "probe_failed_hold", state.RestorePhase)
	assert.Equal(t, "subscription_transient", state.RestoreReason)
	assert.Equal(t, "failed", state.LastProbeResult)
	assert.NotZero(t, state.LastProbeAt)
	assert.GreaterOrEqual(t, state.RestoreHoldUntil-state.LastProbeAt, int64(2))
}

func TestRunRoutingAutomationOnceProbeRestoresSlowAutoDisabledChannel(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origSlow := setting.SlowChannelPolicy
	origProbe := setting.ProbePolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.SlowChannelPolicy = origSlow
		setting.ProbePolicy = origProbe
	})

	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	setting.SlowChannelPolicy.SummaryEnabled = true
	setting.SlowChannelPolicy.WindowMinutes = 60
	setting.SlowChannelPolicy.MinRequests = 3
	setting.SlowChannelPolicy.P95Seconds = 20
	setting.SlowChannelPolicy.SlowRequestSeconds = 15
	setting.SlowChannelPolicy.SlowRatioPercent = 50
	setting.SlowChannelPolicy.AffinityClearEnabled = true
	setting.SlowChannelPolicy.AutoDisableEnabled = true
	setting.SlowChannelPolicy.AutoDisableMinP95Seconds = 20
	setting.SlowChannelPolicy.AutoDisableHoldSeconds = 1
	setting.SlowChannelPolicy.MinEnabledSubscriptions = 1
	setting.SlowChannelPolicy.AutoDisableMaxPerRun = 1
	setting.ProbePolicy.ActiveProbeEnabled = true
	setting.ProbePolicy.ProbeRetrySeconds = 1
	setting.ProbePolicy.ActiveProbeIntervalSeconds = 1

	RegisterRoutingPolicyProbeExecutor(func(ctx context.Context, channel *model.Channel, probe operation_setting.RoutingPolicyProbe) bool {
		return true
	})

	now := common.GetTimestamp()
	subTag := "subscription"
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3034,
		Name:   "slow-channel-probe-restore",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3035,
		Name:   "slow-channel-probe-restore-floor",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)
	for i, useTime := range []int{25, 30, 40} {
		require.NoError(t, model.LOG_DB.Create(&model.Log{
			UserId:    1,
			CreatedAt: now - int64(i),
			Type:      model.LogTypeConsume,
			Content:   fmt.Sprintf("slow-restore-%d", i),
			ModelName: "gpt-5",
			ChannelId: 3034,
			UseTime:   useTime,
			Other:     `{"request_path":"/v1/chat/completions"}`,
		}).Error)
	}

	firstSummary := RunRoutingAutomationOnce(context.Background())
	require.Equal(t, []int{3034}, firstSummary.SlowChannels)
	assert.NotEmpty(t, firstSummary.LastSubscriptionDisableAction)
	assert.NotZero(t, firstSummary.NextProbeAt)

	firstState := SnapshotRoutingState()[3034]
	assert.Equal(t, "waiting_probe", firstState.RestorePhase)
	assert.Equal(t, "slow_channel", firstState.RestoreReason)
	assert.Greater(t, firstState.RestoreHoldUntil, firstState.LastErrorAt)

	waitForRoutingHoldUntil(t, firstState.RestoreHoldUntil)
	summary := RunRoutingAutomationOnce(context.Background())

	channelAfter, err := model.GetChannelById(3034, true)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusEnabled, channelAfter.Status)
	assert.NotZero(t, summary.LastProbeAt)
	assert.Contains(t, summary.LastProbeAction, "restored")
	assert.Empty(t, summary.LastSubscriptionDisableAction)

	state := SnapshotRoutingState()[3034]
	assert.Equal(t, "restored", state.RestorePhase)
	assert.Equal(t, "slow_channel", state.RestoreReason)
	assert.Equal(t, "success", state.LastProbeResult)
	assert.Equal(t, int64(0), state.RestoreHoldUntil)
	assert.Equal(t, int64(0), state.CooldownUntil)
}

func TestRunRoutingAutomationOnceProbeFailureMarksSlowChannelAsProbeFailedHold(t *testing.T) {
	truncate(t)
	resetRoutingRuntimeStateForTest(t)

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	origSlow := setting.SlowChannelPolicy
	origProbe := setting.ProbePolicy
	t.Cleanup(func() {
		setting.Mode = origMode
		setting.SlowChannelPolicy = origSlow
		setting.ProbePolicy = origProbe
	})

	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	setting.SlowChannelPolicy.SummaryEnabled = true
	setting.SlowChannelPolicy.WindowMinutes = 60
	setting.SlowChannelPolicy.MinRequests = 3
	setting.SlowChannelPolicy.P95Seconds = 20
	setting.SlowChannelPolicy.SlowRequestSeconds = 15
	setting.SlowChannelPolicy.SlowRatioPercent = 50
	setting.SlowChannelPolicy.AffinityClearEnabled = true
	setting.SlowChannelPolicy.AutoDisableEnabled = true
	setting.SlowChannelPolicy.AutoDisableMinP95Seconds = 20
	setting.SlowChannelPolicy.AutoDisableHoldSeconds = 1
	setting.SlowChannelPolicy.MinEnabledSubscriptions = 1
	setting.SlowChannelPolicy.AutoDisableMaxPerRun = 1
	setting.ProbePolicy.ActiveProbeEnabled = true
	setting.ProbePolicy.ProbeRetrySeconds = 1
	setting.ProbePolicy.ActiveProbeIntervalSeconds = 2

	RegisterRoutingPolicyProbeExecutor(func(ctx context.Context, channel *model.Channel, probe operation_setting.RoutingPolicyProbe) bool {
		return false
	})

	now := common.GetTimestamp()
	subTag := "subscription"
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3036,
		Name:   "slow-channel-probe-failed-hold",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:     3037,
		Name:   "slow-channel-probe-failed-floor",
		Status: common.ChannelStatusEnabled,
		Tag:    &subTag,
	}).Error)
	for i, useTime := range []int{25, 30, 40} {
		require.NoError(t, model.LOG_DB.Create(&model.Log{
			UserId:    1,
			CreatedAt: now - int64(i),
			Type:      model.LogTypeConsume,
			Content:   fmt.Sprintf("slow-failed-%d", i),
			ModelName: "gpt-5",
			ChannelId: 3036,
			UseTime:   useTime,
			Other:     `{"request_path":"/v1/chat/completions"}`,
		}).Error)
	}

	firstSummary := RunRoutingAutomationOnce(context.Background())
	require.Equal(t, []int{3036}, firstSummary.SlowChannels)
	assert.NotEmpty(t, firstSummary.LastSubscriptionDisableAction)
	assert.NotZero(t, firstSummary.NextProbeAt)

	firstState := SnapshotRoutingState()[3036]
	assert.Equal(t, "waiting_probe", firstState.RestorePhase)
	assert.Equal(t, "slow_channel", firstState.RestoreReason)
	assert.Greater(t, firstState.RestoreHoldUntil, firstState.LastErrorAt)

	waitForRoutingHoldUntil(t, firstState.RestoreHoldUntil)
	summary := RunRoutingAutomationOnce(context.Background())

	channelAfter, err := model.GetChannelById(3036, true)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusAutoDisabled, channelAfter.Status)
	assert.NotZero(t, summary.LastProbeAt)
	assert.NotZero(t, summary.NextProbeAt)
	assert.Contains(t, summary.LastProbeAction, "probe_failed_hold")

	state := SnapshotRoutingState()[3036]
	assert.Equal(t, "probe_failed_hold", state.RestorePhase)
	assert.Equal(t, "slow_channel", state.RestoreReason)
	assert.Equal(t, "failed", state.LastProbeResult)
	assert.NotZero(t, state.LastProbeAt)
	assert.GreaterOrEqual(t, state.RestoreHoldUntil-state.LastProbeAt, int64(2))
	assert.Equal(t, state.RestoreHoldUntil, summary.NextProbeAt)
}
