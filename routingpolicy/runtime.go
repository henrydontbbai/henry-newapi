package routingpolicy

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/cachex"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/samber/hot"
)

type Decision struct {
	Healthy bool
	Reason  string
}

type HealthState struct {
	CooldownUntil       int64  `json:"cooldown_until"`
	Reason              string `json:"reason,omitempty"`
	LastErrorAt         int64  `json:"last_error_at,omitempty"`
	LastSuccessAt       int64  `json:"last_success_at,omitempty"`
	FailureCount        int    `json:"failure_count,omitempty"`
	SuccessCount        int    `json:"success_count,omitempty"`
	LastStatusCode      int    `json:"last_status_code,omitempty"`
	LastAffinityClearAt int64  `json:"last_affinity_clear_at,omitempty"`
}

type Summary struct {
	RoutingMode                   string `json:"routing_mode"`
	ReasonCode                    string `json:"reason_code"`
	SlowChannelCount              int    `json:"slow_channel_count"`
	SlowChannels                  []int  `json:"slow_channels"`
	MaxSubscriptionP95            int    `json:"max_subscription_p95"`
	PaygoHardFailureCount         int    `json:"paygo_hard_failure_count"`
	PaygoHardFailureChannels      []int  `json:"paygo_hard_failure_channels"`
	LastProbeAt                   int64  `json:"last_probe_at"`
	LastNudgeAt                   int64  `json:"last_nudge_at"`
	LastSlowScanAt                int64  `json:"last_slow_scan_at"`
	LastSubscriptionDisableAction string `json:"last_subscription_disable_action"`
	LastPaygoRestoreAction        string `json:"last_paygo_restore_action"`
}

type ChannelRole string

const (
	ChannelRoleSubscription ChannelRole = "subscription"
	ChannelRolePaygo        ChannelRole = "paygo"
	ChannelRoleOther        ChannelRole = "other"
)

const (
	routingPolicyStateNamespace         = "new-api:routing_policy_state:v1"
	routingPolicySummaryNamespace       = "new-api:routing_policy_summary:v1"
	routingPolicySummaryKey             = "summary"
	routingPolicySnapshotNamespace      = "new-api:routing_policy_snapshot:v1"
	routingPolicyStateTTL               = 48 * time.Hour
	routingPolicySummaryTTL             = 48 * time.Hour
	routingPolicySnapshotTTL            = 48 * time.Hour
	snapshotKeyPaygoNudge               = "paygo_nudge_snapshot"
	snapshotKeyPaygoHardFailureRestore  = "paygo_hard_failure_restore"
	snapshotKeySubscriptionTransientRestore = "subscription_transient_restore"
)

type roleDetector func(*model.Channel) ChannelRole

type AutomationHooks struct {
	ClearAffinityByChannel func(channelID int) int
	ProbeChannel           func(ctx context.Context, channel *model.Channel, probe operation_setting.RoutingPolicyProbe) bool
}

type slowChannelSample struct {
	ChannelID    int
	RequestCount int
	P95Seconds   int
	SlowRatio    int
}

type channelSnapshot struct {
	ChannelID    int   `json:"channel_id"`
	Status       int   `json:"status"`
	Priority     int64 `json:"priority"`
	Weight       uint  `json:"weight"`
	SnapshotAt   int64 `json:"snapshot_at"`
	HoldUntil    int64 `json:"hold_until"`
	LastActionAt int64 `json:"last_action_at"`
}

var (
	stateCacheOnce    sync.Once
	stateCache        *cachex.HybridCache[HealthState]
	summaryCacheOnce  sync.Once
	summaryCache      *cachex.HybridCache[Summary]
	snapshotCacheOnce sync.Once
	snapshotCache     *cachex.HybridCache[map[string]channelSnapshot]

	summaryMu            sync.RWMutex
	inMemorySummaryState = Summary{RoutingMode: "SUBSCRIPTION_PRIMARY"}
)

func getStateCache() *cachex.HybridCache[HealthState] {
	stateCacheOnce.Do(func() {
		stateCache = cachex.NewHybridCache[HealthState](cachex.HybridCacheConfig[HealthState]{
			Namespace: cachex.Namespace(routingPolicyStateNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[HealthState]{},
			Memory: func() *hot.HotCache[string, HealthState] {
				return hot.NewHotCache[string, HealthState](hot.LRU, 100_000).
					WithTTL(routingPolicyStateTTL).
					WithJanitor().
					Build()
			},
		})
	})
	return stateCache
}

func getSummaryCache() *cachex.HybridCache[Summary] {
	summaryCacheOnce.Do(func() {
		summaryCache = cachex.NewHybridCache[Summary](cachex.HybridCacheConfig[Summary]{
			Namespace: cachex.Namespace(routingPolicySummaryNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[Summary]{},
			Memory: func() *hot.HotCache[string, Summary] {
				return hot.NewHotCache[string, Summary](hot.LRU, 16).
					WithTTL(routingPolicySummaryTTL).
					WithJanitor().
					Build()
			},
		})
	})
	return summaryCache
}

func getSnapshotCache() *cachex.HybridCache[map[string]channelSnapshot] {
	snapshotCacheOnce.Do(func() {
		snapshotCache = cachex.NewHybridCache[map[string]channelSnapshot](cachex.HybridCacheConfig[map[string]channelSnapshot]{
			Namespace: cachex.Namespace(routingPolicySnapshotNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[map[string]channelSnapshot]{},
			Memory: func() *hot.HotCache[string, map[string]channelSnapshot] {
				return hot.NewHotCache[string, map[string]channelSnapshot](hot.LRU, 32).
					WithTTL(routingPolicySnapshotTTL).
					WithJanitor().
					Build()
			},
		})
	})
	return snapshotCache
}

func cacheKeyForChannel(channelID int) string {
	if channelID <= 0 {
		return ""
	}
	return fmt.Sprintf("channel:%d", channelID)
}

func IsEnabled() bool {
	return operation_setting.GetRoutingPolicySetting().Mode != operation_setting.RoutingPolicyModeDisabled
}

func IsEnforceMode() bool {
	return operation_setting.GetRoutingPolicySetting().Mode == operation_setting.RoutingPolicyModeEnforce
}

func loadState(channelID int) (HealthState, bool) {
	key := cacheKeyForChannel(channelID)
	if key == "" {
		return HealthState{}, false
	}
	state, found, err := getStateCache().Get(key)
	if err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("routing policy state load failed: channel=%d err=%v", channelID, err))
		return HealthState{}, false
	}
	return state, found
}

func storeState(channelID int, state HealthState) {
	key := cacheKeyForChannel(channelID)
	if key == "" {
		return
	}
	if err := getStateCache().SetWithTTL(key, state, routingPolicyStateTTL); err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("routing policy state store failed: channel=%d err=%v", channelID, err))
	}
}

func IsChannelHealthy(channelID int, now time.Time) Decision {
	if !IsEnabled() {
		return Decision{Healthy: true}
	}
	state, ok := loadState(channelID)
	if !ok {
		return Decision{Healthy: true}
	}
	if state.CooldownUntil > now.Unix() {
		return Decision{Healthy: false, Reason: state.Reason}
	}
	return Decision{Healthy: true}
}

func MarkFailure(channelID int, statusCode int, reason string, cooldown time.Duration) {
	if channelID <= 0 {
		return
	}
	now := time.Now().Unix()
	state, _ := loadState(channelID)
	state.Reason = strings.TrimSpace(reason)
	state.LastErrorAt = now
	state.LastStatusCode = statusCode
	state.FailureCount++
	if cooldown > 0 {
		state.CooldownUntil = now + int64(cooldown.Seconds())
	}
	storeState(channelID, state)
}

func MarkSuccess(channelID int) {
	if channelID <= 0 {
		return
	}
	now := time.Now().Unix()
	state, _ := loadState(channelID)
	state.LastSuccessAt = now
	state.SuccessCount++
	state.CooldownUntil = 0
	state.Reason = ""
	state.LastStatusCode = 0
	storeState(channelID, state)
}

func LogSkipDecision(ctx context.Context, channelID int, reason string, enforced bool) {
	mode := "would-skip"
	if enforced {
		mode = "did-skip"
	}
	logger.LogInfo(ctx, fmt.Sprintf("routing policy %s channel=%d reason=%s", mode, channelID, reason))
}

func RecordFailOpen(ctx context.Context, candidateCount int) {
	logger.LogWarn(ctx, fmt.Sprintf("routing policy fail-open candidates=%d", candidateCount))
}

func GetSummary() Summary {
	summary, found, err := getSummaryCache().Get(routingPolicySummaryKey)
	if err == nil && found {
		return summary
	}
	summaryMu.RLock()
	defer summaryMu.RUnlock()
	return inMemorySummaryState
}

func UpdateSummary(summary Summary) {
	summaryMu.Lock()
	inMemorySummaryState = summary
	summaryMu.Unlock()
	if err := getSummaryCache().SetWithTTL(routingPolicySummaryKey, summary, routingPolicySummaryTTL); err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("routing policy summary store failed: err=%v", err))
	}
}

func RunAutomationOnce(ctx context.Context, detectRole roleDetector, hooks AutomationHooks) Summary {
	now := time.Now()
	summary := GetSummary()
	summary.LastSubscriptionDisableAction = ""
	summary.LastPaygoRestoreAction = ""
	if !IsEnabled() || detectRole == nil {
		UpdateSummary(summary)
		return summary
	}

	channels, err := model.GetAllChannels(0, 0, true, true)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("routing policy automation failed to list channels: %v", err))
		UpdateSummary(summary)
		return summary
	}

	channelMap := make(map[int]*model.Channel, len(channels))
	subscriptionChannels := make([]*model.Channel, 0, len(channels))
	paygoChannels := make([]*model.Channel, 0, len(channels))
	for _, channel := range channels {
		if channel == nil {
			continue
		}
		channelMap[channel.Id] = channel
		switch detectRole(channel) {
		case ChannelRoleSubscription:
			subscriptionChannels = append(subscriptionChannels, channel)
		case ChannelRolePaygo:
			paygoChannels = append(paygoChannels, channel)
		}
	}

	summary.LastSlowScanAt = common.GetTimestamp()

	restoreSubscriptionTransientChannels(ctx, &summary, channelMap, now, hooks)
	restorePaygoHardFailureChannels(ctx, &summary, channelMap, now, hooks)
	restorePaygoNudgeSnapshot(ctx, &summary, channelMap, now)

	slowSamples := []slowChannelSample{}
	if operation_setting.GetRoutingPolicySetting().SlowChannelPolicy.SummaryEnabled {
		slowSamples = scanSlowSubscriptionChannels(ctx, detectRole, channels, now)
	}
	applySlowChannelSummary(ctx, &summary, slowSamples, subscriptionChannels, now, hooks)
	applySubscriptionTransientIsolation(ctx, &summary, detectRole, channels, now, hooks)
	applyPaygoHardFailureIsolation(ctx, &summary, detectRole, channels, now, hooks)
	applyPaygoNudge(ctx, &summary, subscriptionChannels, paygoChannels, now, hooks)
	updateRoutingMode(&summary, subscriptionChannels, paygoChannels)
	UpdateSummary(summary)
	return summary
}

func DescribeState(channelID int) string {
	state, ok := loadState(channelID)
	if !ok {
		return ""
	}
	return fmt.Sprintf("cooldown_until=%d reason=%s failures=%d", state.CooldownUntil, state.Reason, state.FailureCount)
}

func SnapshotState() map[int]HealthState {
	keys, err := getStateCache().Keys()
	if err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("routing policy state key scan failed: %v", err))
		return map[int]HealthState{}
	}
	result := make(map[int]HealthState, len(keys))
	ids := make([]int, 0, len(keys))
	for _, key := range keys {
		channelID := parseChannelIDFromKey(key)
		if channelID <= 0 {
			continue
		}
		state, ok := loadState(channelID)
		if !ok {
			continue
		}
		ids = append(ids, channelID)
		result[channelID] = state
	}
	sort.Ints(ids)
	ordered := make(map[int]HealthState, len(ids))
	for _, id := range ids {
		ordered[id] = result[id]
	}
	return ordered
}

func ResetForTest() {
	stateCacheOnce = sync.Once{}
	stateCache = nil
	summaryCacheOnce = sync.Once{}
	summaryCache = nil
	snapshotCacheOnce = sync.Once{}
	snapshotCache = nil
	summaryMu.Lock()
	inMemorySummaryState = Summary{RoutingMode: "SUBSCRIPTION_PRIMARY"}
	summaryMu.Unlock()
}

func parseChannelIDFromKey(key string) int {
	key = strings.TrimSpace(key)
	if idx := strings.LastIndex(key, "channel:"); idx >= 0 {
		key = key[idx+len("channel:"):]
	}
	return common.String2Int(key)
}

func applySlowChannelSummary(ctx context.Context, summary *Summary, samples []slowChannelSample, subscriptionChannels []*model.Channel, now time.Time, hooks AutomationHooks) {
	if summary == nil {
		return
	}
	setting := operation_setting.GetRoutingPolicySetting().SlowChannelPolicy
	probeSetting := operation_setting.GetRoutingPolicySetting().ProbePolicy
	summary.SlowChannels = summary.SlowChannels[:0]
	summary.SlowChannelCount = 0
	summary.MaxSubscriptionP95 = 0
	if len(samples) == 0 {
		return
	}

	enabledCount := countEnabledChannels(subscriptionChannels)
	autoDisabledCount := 0
	sort.Slice(samples, func(i, j int) bool { return samples[i].ChannelID < samples[j].ChannelID })
	for _, sample := range samples {
		summary.SlowChannels = append(summary.SlowChannels, sample.ChannelID)
		if sample.P95Seconds > summary.MaxSubscriptionP95 {
			summary.MaxSubscriptionP95 = sample.P95Seconds
		}
		if setting.AffinityClearEnabled {
			clearAffinityByChannelIfNeeded(sample.ChannelID, setting.AffinityClearCooldownSeconds, "slow_channel", ctx, hooks)
		}
		logger.LogInfo(ctx, fmt.Sprintf("routing policy slow-channel-confirmed channel=%d p95=%ds slow_ratio=%d%% requests=%d",
			sample.ChannelID, sample.P95Seconds, sample.SlowRatio, sample.RequestCount))
		if !setting.AutoDisableEnabled || !IsEnforceMode() {
			continue
		}
		if !probeSetting.ActiveProbeEnabled || hooks.ProbeChannel == nil {
			logger.LogWarn(ctx, fmt.Sprintf("routing policy skip-disable slow subscription channel=%d reason=probe_disabled", sample.ChannelID))
			continue
		}
		if sample.P95Seconds < setting.AutoDisableMinP95Seconds {
			continue
		}
		if enabledCount-autoDisabledCount <= setting.MinEnabledSubscriptions {
			continue
		}
		if autoDisabledCount >= setting.AutoDisableMaxPerRun {
			continue
		}
		state, _ := loadState(sample.ChannelID)
		if state.CooldownUntil > now.Unix() && state.Reason == "slow_channel" {
			continue
		}
		channel, err := model.GetChannelById(sample.ChannelID, true)
		if err != nil || channel == nil || channel.Status != common.ChannelStatusEnabled {
			continue
		}
		action := applyTemporaryChannelDisable(channel, "routing_policy_slow_channel", setting.AutoDisableHoldSeconds, "slow_channel")
		if action == "" {
			continue
		}
		syncChannelStatusInSlice(subscriptionChannels, sample.ChannelID, common.ChannelStatusAutoDisabled)
		summary.LastSubscriptionDisableAction = action
		autoDisabledCount++
		logger.LogWarn(ctx, fmt.Sprintf("routing policy disable slow subscription channel=%d p95=%d", sample.ChannelID, sample.P95Seconds))
	}
	summary.SlowChannelCount = len(summary.SlowChannels)
}

func clearAffinityByChannelIfNeeded(channelID int, cooldownSeconds int, reason string, ctx context.Context, hooks AutomationHooks) bool {
	if channelID <= 0 {
		return false
	}
	if !IsEnforceMode() {
		logger.LogInfo(ctx, fmt.Sprintf("routing policy would-clear affinity channel=%d reason=%s", channelID, reason))
		return false
	}
	state, _ := loadState(channelID)
	now := time.Now().Unix()
	if cooldownSeconds > 0 && state.LastAffinityClearAt > 0 && now-state.LastAffinityClearAt < int64(cooldownSeconds) {
		return false
	}
	deleted := 0
	if hooks.ClearAffinityByChannel != nil {
		deleted = hooks.ClearAffinityByChannel(channelID)
	}
	state.LastAffinityClearAt = now
	if state.Reason == "" {
		state.Reason = reason
	}
	storeState(channelID, state)
	if deleted > 0 {
		logger.LogInfo(ctx, fmt.Sprintf("routing policy affinity-clear channel=%d reason=%s deleted=%d", channelID, reason, deleted))
	}
	return deleted > 0
}

func syncChannelStatusInSlice(channels []*model.Channel, channelID int, status int) {
	for _, channel := range channels {
		if channel != nil && channel.Id == channelID {
			channel.Status = status
			return
		}
	}
}

func scanSlowSubscriptionChannels(ctx context.Context, detectRole roleDetector, channels []*model.Channel, now time.Time) []slowChannelSample {
	_ = ctx
	setting := operation_setting.GetRoutingPolicySetting().SlowChannelPolicy
	if len(channels) == 0 {
		return nil
	}

	candidateSet := make(map[int]struct{})
	for _, channel := range channels {
		if channel == nil || channel.Status != common.ChannelStatusEnabled {
			continue
		}
		if detectRole(channel) == ChannelRoleSubscription {
			candidateSet[channel.Id] = struct{}{}
		}
	}
	if len(candidateSet) == 0 {
		return nil
	}

	windowStart := now.Add(-time.Duration(setting.WindowMinutes) * time.Minute).Unix()
	var logs []model.Log
	query := model.LOG_DB.Model(&model.Log{}).
		Where("type = ?", model.LogTypeConsume).
		Where("created_at >= ?", windowStart).
		Where("channel_id IN ?", keysFromSet(candidateSet)).
		Order("created_at desc")
	if err := query.Find(&logs).Error; err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("routing policy slow scan failed to query logs: %v", err))
		return nil
	}

	type agg struct {
		durations []int
		slowCount int
	}
	aggregates := make(map[int]*agg)
	for _, entry := range logs {
		if entry.ChannelId <= 0 || entry.UseTime <= 0 {
			continue
		}
		if !isRelayRequestLog(entry.Other) || isModelTestLog(entry.Content) {
			continue
		}
		current := aggregates[entry.ChannelId]
		if current == nil {
			current = &agg{}
			aggregates[entry.ChannelId] = current
		}
		current.durations = append(current.durations, entry.UseTime)
		if entry.UseTime >= setting.SlowRequestSeconds {
			current.slowCount++
		}
	}

	results := make([]slowChannelSample, 0, len(aggregates))
	for channelID, aggregate := range aggregates {
		if len(aggregate.durations) < setting.MinRequests {
			continue
		}
		p95 := percentile95(aggregate.durations)
		slowRatio := aggregate.slowCount * 100 / len(aggregate.durations)
		if p95 < setting.P95Seconds && slowRatio < setting.SlowRatioPercent {
			continue
		}
		results = append(results, slowChannelSample{
			ChannelID:    channelID,
			RequestCount: len(aggregate.durations),
			P95Seconds:   p95,
			SlowRatio:    slowRatio,
		})
	}
	return results
}

func keysFromSet(items map[int]struct{}) []int {
	result := make([]int, 0, len(items))
	for id := range items {
		result = append(result, id)
	}
	sort.Ints(result)
	return result
}

func percentile95(values []int) int {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int(nil), values...)
	sort.Ints(sorted)
	index := (len(sorted)*95 + 99) / 100
	if index <= 0 {
		index = 1
	}
	if index > len(sorted) {
		index = len(sorted)
	}
	return sorted[index-1]
}

func isRelayRequestLog(otherRaw string) bool {
	otherMap, err := common.StrToMap(otherRaw)
	if err != nil || len(otherMap) == 0 {
		return true
	}
	requestPath := fmt.Sprintf("%v", otherMap["request_path"])
	return strings.HasPrefix(requestPath, "/v1/")
}

func isModelTestLog(content string) bool {
	content = strings.TrimSpace(strings.ToLower(content))
	return strings.Contains(content, "\u6a21\u578b\u6d4b\u8bd5") || strings.Contains(content, "channel test")
}

func applyPaygoHardFailureIsolation(ctx context.Context, summary *Summary, detectRole roleDetector, channels []*model.Channel, now time.Time, hooks AutomationHooks) {
	if summary == nil {
		return
	}
	setting := operation_setting.GetRoutingPolicySetting().PaygoHardFailurePolicy
	probeSetting := operation_setting.GetRoutingPolicySetting().ProbePolicy
	summary.PaygoHardFailureChannels = summary.PaygoHardFailureChannels[:0]
	summary.PaygoHardFailureCount = 0
	if !setting.Enabled {
		return
	}

	channelMap := make(map[int]*model.Channel, len(channels))
	candidates := make([]int, 0)
	for _, channel := range channels {
		if channel == nil {
			continue
		}
		channelMap[channel.Id] = channel
		if channel.Status == common.ChannelStatusEnabled && detectRole(channel) == ChannelRolePaygo {
			candidates = append(candidates, channel.Id)
		}
	}
	if len(candidates) == 0 {
		return
	}

	windowStart := now.Add(-time.Duration(setting.WindowMinutes) * time.Minute).Unix()
	var logs []model.Log
	if err := model.LOG_DB.Model(&model.Log{}).
		Where("type = ?", model.LogTypeError).
		Where("created_at >= ?", windowStart).
		Where("channel_id IN ?", candidates).
		Order("created_at desc").
		Find(&logs).Error; err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("routing policy paygo hard-failure scan failed: %v", err))
		return
	}

	hits := make(map[int]int)
	stateByChannel := make(map[int]HealthState, len(candidates))
	for _, channelID := range candidates {
		state, _ := loadState(channelID)
		stateByChannel[channelID] = state
	}
	for _, entry := range logs {
		state := stateByChannel[entry.ChannelId]
		if state.LastSuccessAt > 0 && entry.CreatedAt <= state.LastSuccessAt {
			continue
		}
		if isPaygoHardFailureLog(entry) {
			hits[entry.ChannelId]++
		}
	}

	restoreSnapshots, _ := loadChannelSnapshots(snapshotKeyPaygoHardFailureRestore)
	if restoreSnapshots == nil {
		restoreSnapshots = map[string]channelSnapshot{}
	}
	disabled := 0
	for _, channelID := range candidates {
		count := hits[channelID]
		if count < setting.Threshold {
			continue
		}
		summary.PaygoHardFailureChannels = append(summary.PaygoHardFailureChannels, channelID)
		if disabled >= setting.MaxPerRun {
			continue
		}
		channel := channelMap[channelID]
		if channel == nil {
			continue
		}
		if !IsEnforceMode() {
			logger.LogInfo(ctx, fmt.Sprintf("routing policy would-disable paygo hard-failure channel=%d hits=%d", channelID, count))
			continue
		}
		if !probeSetting.ActiveProbeEnabled || hooks.ProbeChannel == nil {
			logger.LogWarn(ctx, fmt.Sprintf("routing policy skip-disable paygo hard-failure channel=%d reason=probe_disabled", channelID))
			continue
		}
		action := applyTemporaryChannelDisable(channel, "routing_policy_paygo_hard_failure", setting.RetrySeconds, "paygo_hard_failure")
		if action == "" {
			continue
		}
		snapKey := snapshotKey(channelID)
		if _, exists := restoreSnapshots[snapKey]; !exists {
			restoreSnapshots[snapKey] = channelSnapshot{
				ChannelID:    channelID,
				Status:       channel.Status,
				Priority:     channel.GetPriority(),
				Weight:       uint(channel.GetWeight()),
				SnapshotAt:   now.Unix(),
				HoldUntil:    now.Unix() + int64(setting.RetrySeconds),
				LastActionAt: now.Unix(),
			}
		}
		disabled++
		clearAffinityByChannelIfNeeded(channelID, setting.RetrySeconds, "paygo_hard_failure", ctx, hooks)
		logger.LogWarn(ctx, fmt.Sprintf("routing policy disable channel=%d reason=paygo_hard_failure hits=%d", channelID, count))
	}
	if len(restoreSnapshots) > 0 {
		storeChannelSnapshots(snapshotKeyPaygoHardFailureRestore, restoreSnapshots)
	}
	summary.PaygoHardFailureCount = len(summary.PaygoHardFailureChannels)
}

func isPaygoHardFailureLog(entry model.Log) bool {
	otherMap, err := common.StrToMap(entry.Other)
	if err != nil || len(otherMap) == 0 {
		return false
	}
	errorCode := strings.ToLower(fmt.Sprintf("%v", otherMap["error_code"]))
	statusCode := common.String2Int(fmt.Sprintf("%v", otherMap["status_code"]))
	if strings.Contains(errorCode, "invalid_key") || strings.Contains(errorCode, "access_denied") {
		return true
	}
	if statusCode == 401 {
		return true
	}
	return statusCode == 403 && strings.Contains(strings.ToLower(entry.Content), "key")
}

func restorePaygoHardFailureChannels(ctx context.Context, summary *Summary, channelMap map[int]*model.Channel, now time.Time, hooks AutomationHooks) {
	setting := operation_setting.GetRoutingPolicySetting().PaygoHardFailurePolicy
	probeSetting := operation_setting.GetRoutingPolicySetting().ProbePolicy
	if !setting.Enabled || !probeSetting.ActiveProbeEnabled || hooks.ProbeChannel == nil {
		return
	}
	snapshots, _ := loadChannelSnapshots(snapshotKeyPaygoHardFailureRestore)
	if len(snapshots) == 0 {
		return
	}

	changed := false
	for key, snapshot := range snapshots {
		if snapshot.HoldUntil > now.Unix() {
			continue
		}
		channel := channelMap[snapshot.ChannelID]
		if channel == nil {
			delete(snapshots, key)
			changed = true
			continue
		}
		summary.LastProbeAt = now.Unix()
		if !hooks.ProbeChannel(ctx, channel, probeSetting) {
			if IsEnforceMode() {
				snapshot.HoldUntil = now.Unix() + int64(probeRetryDelaySeconds(probeSetting))
				snapshot.LastActionAt = now.Unix()
				snapshots[key] = snapshot
				changed = true
			}
			logger.LogInfo(ctx, fmt.Sprintf("routing policy probe-failed paygo channel=%d", snapshot.ChannelID))
			continue
		}
		if !IsEnforceMode() {
			logger.LogInfo(ctx, fmt.Sprintf("routing policy would-restore paygo channel=%d", snapshot.ChannelID))
			continue
		}
		if channel.Status != common.ChannelStatusEnabled && !model.UpdateChannelStatus(snapshot.ChannelID, "", common.ChannelStatusEnabled, "") {
			delete(snapshots, key)
			changed = true
			continue
		}
		channel.Status = common.ChannelStatusEnabled
		if channel.Priority == nil {
			channel.Priority = common.GetPointer[int64](setting.RestorePriority)
		} else {
			*channel.Priority = setting.RestorePriority
		}
		if channel.Weight == nil {
			channel.Weight = common.GetPointer[uint](setting.RestoreWeight)
		} else {
			*channel.Weight = setting.RestoreWeight
		}
		if err := channel.SaveWithoutKey(); err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("routing policy restore channel save failed: channel=%d err=%v", snapshot.ChannelID, err))
		}
		if err := channel.UpdateAbilities(nil); err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("routing policy restore channel abilities failed: channel=%d err=%v", snapshot.ChannelID, err))
		}
		MarkSuccess(snapshot.ChannelID)
		action := fmt.Sprintf("channel=%d priority=%d weight=%d", snapshot.ChannelID, setting.RestorePriority, setting.RestoreWeight)
		summary.LastPaygoRestoreAction = action
		logger.LogInfo(ctx, fmt.Sprintf("routing policy probe-success restore paygo channel=%d priority=%d weight=%d", snapshot.ChannelID, setting.RestorePriority, setting.RestoreWeight))
		delete(snapshots, key)
		changed = true
	}
	if changed {
		storeChannelSnapshots(snapshotKeyPaygoHardFailureRestore, snapshots)
	}
}

func applyPaygoNudge(ctx context.Context, summary *Summary, subscriptionChannels []*model.Channel, paygoChannels []*model.Channel, now time.Time, hooks AutomationHooks) {
	setting := operation_setting.GetRoutingPolicySetting().PaygoNudgePolicy
	if !setting.Enabled || len(paygoChannels) == 0 {
		return
	}
	if setting.RequireNoSlowChannels && summary.SlowChannelCount > 0 {
		return
	}
	if countEnabledChannels(subscriptionChannels) < setting.MinSubscriptions {
		return
	}

	windowStart := now.Add(-time.Duration(setting.RecentWindowMinutes) * time.Minute).Unix()
	successCount := countRecentConsumeLogs(subscriptionChannels, windowStart)
	if successCount < setting.MinRecentSubscriptionSuccess {
		return
	}
	errorCount := countRecentErrorLogs(subscriptionChannels, windowStart)
	if errorCount > setting.MaxRecentErrors {
		return
	}
	transientErrors := countTransientSubscriptionErrors(subscriptionChannels, windowStart)
	if transientErrors > setting.MaxTransientErrors {
		return
	}

	nudgeSnapshots, _ := loadChannelSnapshots(snapshotKeyPaygoNudge)
	if len(nudgeSnapshots) > 0 {
		return
	}
	if summary.LastNudgeAt > 0 && now.Unix()-summary.LastNudgeAt < int64(setting.CooldownSeconds) {
		return
	}

	for _, channel := range paygoChannels {
		if channel == nil || channel.Status != common.ChannelStatusEnabled {
			continue
		}
		clearAffinityByChannelIfNeeded(channel.Id, setting.CooldownSeconds, "paygo_nudge", ctx, hooks)
	}

	if !IsEnforceMode() {
		logger.LogInfo(ctx, "routing policy would-start paygo nudge")
		summary.LastNudgeAt = now.Unix()
		return
	}

	snapshots := make(map[string]channelSnapshot)
	for _, channel := range paygoChannels {
		if channel == nil || channel.Status != common.ChannelStatusEnabled {
			continue
		}
		if model.UpdateChannelStatus(channel.Id, "", common.ChannelStatusAutoDisabled, "routing_policy_paygo_nudge_hold") {
			channel.Status = common.ChannelStatusAutoDisabled
			snapshots[snapshotKey(channel.Id)] = channelSnapshot{
				ChannelID:    channel.Id,
				Status:       channel.Status,
				Priority:     channel.GetPriority(),
				Weight:       uint(channel.GetWeight()),
				SnapshotAt:   now.Unix(),
				HoldUntil:    now.Unix() + int64(setting.HoldSeconds),
				LastActionAt: now.Unix(),
			}
			logger.LogInfo(ctx, fmt.Sprintf("routing policy nudge-hold-disable paygo channel=%d hold_seconds=%d", channel.Id, setting.HoldSeconds))
		}
	}
	if len(snapshots) == 0 {
		return
	}
	storeChannelSnapshots(snapshotKeyPaygoNudge, snapshots)
	summary.LastNudgeAt = now.Unix()
}

func restorePaygoNudgeSnapshot(ctx context.Context, summary *Summary, channelMap map[int]*model.Channel, now time.Time) {
	snapshots, _ := loadChannelSnapshots(snapshotKeyPaygoNudge)
	if len(snapshots) == 0 {
		return
	}
	hardFailureSnapshots, _ := loadChannelSnapshots(snapshotKeyPaygoHardFailureRestore)
	changed := false
	for key, snapshot := range snapshots {
		if snapshot.HoldUntil > now.Unix() {
			continue
		}
		if _, isolated := hardFailureSnapshots[snapshotKey(snapshot.ChannelID)]; isolated {
			delete(snapshots, key)
			changed = true
			continue
		}
		channel := channelMap[snapshot.ChannelID]
		if channel == nil {
			delete(snapshots, key)
			changed = true
			continue
		}
		if !IsEnforceMode() {
			logger.LogInfo(ctx, fmt.Sprintf("routing policy would-restore paygo nudge channel=%d", snapshot.ChannelID))
			continue
		}
		if model.UpdateChannelStatus(snapshot.ChannelID, "", common.ChannelStatusEnabled, "") {
			channel.Status = common.ChannelStatusEnabled
			logger.LogInfo(ctx, fmt.Sprintf("routing policy nudge-restore paygo channel=%d", snapshot.ChannelID))
		}
		delete(snapshots, key)
		changed = true
	}
	if changed {
		storeChannelSnapshots(snapshotKeyPaygoNudge, snapshots)
		summary.LastPaygoRestoreAction = "paygo_nudge_restore"
	}
}

func applySubscriptionTransientIsolation(ctx context.Context, summary *Summary, detectRole roleDetector, channels []*model.Channel, now time.Time, hooks AutomationHooks) {
	if summary == nil {
		return
	}
	setting := operation_setting.GetRoutingPolicySetting().SubscriptionPolicy
	probeSetting := operation_setting.GetRoutingPolicySetting().ProbePolicy
	if !setting.TransientUpstreamAffinityClearEnabled && !setting.TransientUpstreamDisableEnabled {
		return
	}

	channelMap := make(map[int]*model.Channel, len(channels))
	candidates := make([]int, 0)
	subscriptionChannels := make([]*model.Channel, 0)
	for _, channel := range channels {
		if channel == nil {
			continue
		}
		channelMap[channel.Id] = channel
		if channel.Status == common.ChannelStatusEnabled && detectRole(channel) == ChannelRoleSubscription {
			candidates = append(candidates, channel.Id)
			subscriptionChannels = append(subscriptionChannels, channel)
		}
	}
	if len(candidates) == 0 {
		return
	}

	windowStart := now.Add(-time.Duration(setting.ErrorWindowMinutes) * time.Minute).Unix()
	var logs []model.Log
	if err := model.LOG_DB.Model(&model.Log{}).
		Where("type = ?", model.LogTypeError).
		Where("created_at >= ?", windowStart).
		Where("channel_id IN ?", candidates).
		Order("created_at desc").
		Find(&logs).Error; err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("routing policy subscription transient scan failed: %v", err))
		return
	}

	stateByChannel := make(map[int]HealthState, len(candidates))
	for _, channelID := range candidates {
		state, _ := loadState(channelID)
		stateByChannel[channelID] = state
	}

	hits := make(map[int]int)
	for _, entry := range logs {
		state := stateByChannel[entry.ChannelId]
		if state.LastSuccessAt > 0 && entry.CreatedAt <= state.LastSuccessAt {
			continue
		}
		if isTransientUpstreamLog(entry) {
			hits[entry.ChannelId]++
		}
	}
	if len(hits) == 0 {
		return
	}

	restoreSnapshots, _ := loadChannelSnapshots(snapshotKeySubscriptionTransientRestore)
	if restoreSnapshots == nil {
		restoreSnapshots = map[string]channelSnapshot{}
	}
	enabledCount := countEnabledChannels(subscriptionChannels)
	disabledCount := 0
	nowUnix := now.Unix()
	sort.Ints(candidates)
	for _, channelID := range candidates {
		count := hits[channelID]
		if count == 0 {
			continue
		}
		if setting.TransientUpstreamAffinityClearEnabled {
			clearAffinityByChannelIfNeeded(channelID, 0, "subscription_transient", ctx, hooks)
		}
		state := stateByChannel[channelID]
		state.Reason = "subscription_transient"
		state.LastErrorAt = nowUnix
		if state.FailureCount < count {
			state.FailureCount = count
		}
		storeState(channelID, state)

		if !setting.TransientUpstreamDisableEnabled || count < setting.TransientUpstreamDisableThreshold {
			continue
		}
		if enabledCount-disabledCount <= setting.TransientUpstreamMinEnabledSubscriptions {
			logger.LogInfo(ctx, fmt.Sprintf("routing policy skip-disable subscription transient channel=%d floor=%d", channelID, setting.TransientUpstreamMinEnabledSubscriptions))
			continue
		}
		if !IsEnforceMode() {
			logger.LogInfo(ctx, fmt.Sprintf("routing policy would-disable subscription transient channel=%d hits=%d", channelID, count))
			continue
		}
		if !probeSetting.ActiveProbeEnabled || hooks.ProbeChannel == nil {
			logger.LogWarn(ctx, fmt.Sprintf("routing policy skip-disable subscription transient channel=%d reason=probe_disabled", channelID))
			continue
		}

		channel := channelMap[channelID]
		if channel == nil || channel.Status != common.ChannelStatusEnabled {
			continue
		}
		action := applyTemporaryChannelDisable(channel, "routing_policy_subscription_transient", probeRetryDelaySeconds(probeSetting), "subscription_transient")
		if action == "" {
			continue
		}
		restoreSnapshots[snapshotKey(channelID)] = channelSnapshot{
			ChannelID:    channelID,
			Status:       channel.Status,
			Priority:     channel.GetPriority(),
			Weight:       uint(channel.GetWeight()),
			SnapshotAt:   nowUnix,
			HoldUntil:    nowUnix + int64(probeRetryDelaySeconds(probeSetting)),
			LastActionAt: nowUnix,
		}
		summary.LastSubscriptionDisableAction = action
		disabledCount++
		logger.LogWarn(ctx, fmt.Sprintf("routing policy disable channel=%d reason=subscription_transient hits=%d", channelID, count))
	}

	if len(restoreSnapshots) > 0 {
		storeChannelSnapshots(snapshotKeySubscriptionTransientRestore, restoreSnapshots)
	}
}

func restoreSubscriptionTransientChannels(ctx context.Context, summary *Summary, channelMap map[int]*model.Channel, now time.Time, hooks AutomationHooks) {
	setting := operation_setting.GetRoutingPolicySetting().SubscriptionPolicy
	probeSetting := operation_setting.GetRoutingPolicySetting().ProbePolicy
	if !setting.TransientUpstreamDisableEnabled || !probeSetting.ActiveProbeEnabled || hooks.ProbeChannel == nil {
		return
	}
	snapshots, _ := loadChannelSnapshots(snapshotKeySubscriptionTransientRestore)
	if len(snapshots) == 0 {
		return
	}

	changed := false
	for key, snapshot := range snapshots {
		if snapshot.HoldUntil > now.Unix() {
			continue
		}
		channel := channelMap[snapshot.ChannelID]
		if channel == nil {
			delete(snapshots, key)
			changed = true
			continue
		}
		summary.LastProbeAt = now.Unix()
		if !hooks.ProbeChannel(ctx, channel, probeSetting) {
			if IsEnforceMode() {
				snapshot.HoldUntil = now.Unix() + int64(probeRetryDelaySeconds(probeSetting))
				snapshot.LastActionAt = now.Unix()
				snapshots[key] = snapshot
				changed = true
			}
			logger.LogInfo(ctx, fmt.Sprintf("routing policy probe-failed subscription channel=%d", snapshot.ChannelID))
			continue
		}
		if !IsEnforceMode() {
			logger.LogInfo(ctx, fmt.Sprintf("routing policy would-restore subscription transient channel=%d", snapshot.ChannelID))
			continue
		}
		if channel.Status != common.ChannelStatusEnabled && !model.UpdateChannelStatus(snapshot.ChannelID, "", common.ChannelStatusEnabled, "") {
			delete(snapshots, key)
			changed = true
			continue
		}
		channel.Status = common.ChannelStatusEnabled
		MarkSuccess(snapshot.ChannelID)
		logger.LogInfo(ctx, fmt.Sprintf("routing policy probe-success restore subscription channel=%d", snapshot.ChannelID))
		delete(snapshots, key)
		changed = true
	}
	if changed {
		storeChannelSnapshots(snapshotKeySubscriptionTransientRestore, snapshots)
	}
}

func countRecentConsumeLogs(channels []*model.Channel, windowStart int64) int {
	channelIDs := make([]int, 0, len(channels))
	for _, channel := range channels {
		if channel != nil && channel.Status == common.ChannelStatusEnabled {
			channelIDs = append(channelIDs, channel.Id)
		}
	}
	if len(channelIDs) == 0 {
		return 0
	}
	var count int64
	_ = model.LOG_DB.Model(&model.Log{}).
		Where("type = ?", model.LogTypeConsume).
		Where("created_at >= ?", windowStart).
		Where("channel_id IN ?", channelIDs).
		Count(&count).Error
	return int(count)
}

func countRecentErrorLogs(channels []*model.Channel, windowStart int64) int {
	channelIDs := make([]int, 0, len(channels))
	for _, channel := range channels {
		if channel != nil && channel.Status == common.ChannelStatusEnabled {
			channelIDs = append(channelIDs, channel.Id)
		}
	}
	if len(channelIDs) == 0 {
		return 0
	}
	var count int64
	_ = model.LOG_DB.Model(&model.Log{}).
		Where("type = ?", model.LogTypeError).
		Where("created_at >= ?", windowStart).
		Where("channel_id IN ?", channelIDs).
		Count(&count).Error
	return int(count)
}

func countTransientSubscriptionErrors(channels []*model.Channel, windowStart int64) int {
	channelIDs := make([]int, 0, len(channels))
	for _, channel := range channels {
		if channel != nil {
			channelIDs = append(channelIDs, channel.Id)
		}
	}
	if len(channelIDs) == 0 {
		return 0
	}
	var logs []model.Log
	if err := model.LOG_DB.Model(&model.Log{}).
		Where("type = ?", model.LogTypeError).
		Where("created_at >= ?", windowStart).
		Where("channel_id IN ?", channelIDs).
		Order("created_at desc").
		Find(&logs).Error; err != nil {
		return 0
	}
	count := 0
	for _, entry := range logs {
		if isTransientUpstreamLog(entry) {
			count++
		}
	}
	return count
}

func isTransientUpstreamLog(entry model.Log) bool {
	otherMap, err := common.StrToMap(entry.Other)
	if err != nil {
		return false
	}
	errorCode := strings.ToLower(fmt.Sprintf("%v", otherMap["error_code"]))
	statusCode := common.String2Int(fmt.Sprintf("%v", otherMap["status_code"]))
	switch statusCode {
	case 0, 502, 503, 504:
		return true
	}
	return strings.Contains(errorCode, "timeout") || strings.Contains(errorCode, "upstream_error")
}

func countEnabledChannels(channels []*model.Channel) int {
	count := 0
	for _, channel := range channels {
		if channel != nil && channel.Status == common.ChannelStatusEnabled {
			count++
		}
	}
	return count
}

func applyTemporaryChannelDisable(channel *model.Channel, reason string, retrySeconds int, stateReason string) string {
	if channel == nil {
		return ""
	}
	if changed := model.UpdateChannelStatus(channel.Id, "", common.ChannelStatusAutoDisabled, reason); !changed {
		return ""
	}
	channel.Status = common.ChannelStatusAutoDisabled
	state, _ := loadState(channel.Id)
	now := time.Now().Unix()
	state.Reason = stateReason
	state.LastErrorAt = now
	state.CooldownUntil = now + int64(retrySeconds)
	storeState(channel.Id, state)
	return fmt.Sprintf("channel=%d reason=%s hold_seconds=%d", channel.Id, stateReason, retrySeconds)
}

func probeRetryDelaySeconds(setting operation_setting.RoutingPolicyProbe) int {
	if setting.ProbeRetrySeconds > 0 {
		return setting.ProbeRetrySeconds
	}
	return 600
}

func updateRoutingMode(summary *Summary, subscriptionChannels []*model.Channel, paygoChannels []*model.Channel) {
	if summary == nil {
		return
	}
	enabledSubscriptions := countEnabledChannels(subscriptionChannels)
	enabledPaygo := countEnabledChannels(paygoChannels)
	summary.RoutingMode = "SUBSCRIPTION_PRIMARY"
	summary.ReasonCode = ""
	switch {
	case enabledSubscriptions == 0 && enabledPaygo == 0:
		summary.RoutingMode = "CRITICAL"
		summary.ReasonCode = "no_healthy_channels"
	case enabledSubscriptions == 0:
		summary.RoutingMode = "PAYGO_ONLY_TEMP"
		if summary.ReasonCode == "" {
			summary.ReasonCode = "subscription_unavailable"
		}
	case enabledPaygo == 0 && len(paygoChannels) > 0:
		summary.RoutingMode = "NO_HEALTHY_PAYGO"
		if summary.ReasonCode == "" {
			summary.ReasonCode = "paygo_unavailable"
		}
	}
	if summary.PaygoHardFailureCount > 0 {
		summary.RoutingMode = "MIXED_DEGRADED"
		summary.ReasonCode = "paygo_hard_failure"
	}
	if summary.SlowChannelCount > 0 {
		if summary.RoutingMode == "SUBSCRIPTION_PRIMARY" {
			summary.RoutingMode = "MIXED_DEGRADED"
		}
		if summary.ReasonCode == "" {
			summary.ReasonCode = "slow_subscription_channels"
		}
	}
}

func loadChannelSnapshots(key string) (map[string]channelSnapshot, bool) {
	value, found, err := getSnapshotCache().Get(key)
	if err != nil || !found {
		return nil, false
	}
	return value, true
}

func storeChannelSnapshots(key string, value map[string]channelSnapshot) {
	if value == nil {
		value = map[string]channelSnapshot{}
	}
	if err := getSnapshotCache().SetWithTTL(key, value, routingPolicySnapshotTTL); err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("routing policy snapshot store failed: key=%s err=%v", key, err))
	}
}

func snapshotKey(channelID int) string {
	return fmt.Sprintf("channel:%d", channelID)
}
