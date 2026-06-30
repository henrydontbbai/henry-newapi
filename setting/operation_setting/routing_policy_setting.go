package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

const (
	RoutingPolicyModeDisabled = "disabled"
	RoutingPolicyModeObserve  = "observe"
	RoutingPolicyModeEnforce  = "enforce"
)

type RoutingPolicyRoleDetection struct {
	SubscriptionTags         []string `json:"subscription_tags"`
	PaygoTags                []string `json:"paygo_tags"`
	SubscriptionNamePatterns []string `json:"subscription_name_patterns"`
}

type RoutingPolicySubscription struct {
	StatusCodeMapping                        map[string]string `json:"status_code_mapping"`
	ErrorWindowMinutes                       int               `json:"error_window_minutes"`
	RateLimitDisableThreshold                int               `json:"rate_limit_disable_threshold"`
	TransientUpstreamAffinityClearEnabled    bool              `json:"transient_upstream_affinity_clear_enabled"`
	TransientUpstreamDisableEnabled          bool              `json:"transient_upstream_disable_enabled"`
	TransientUpstreamDisableThreshold        int               `json:"transient_upstream_disable_threshold"`
	TransientUpstreamMinEnabledSubscriptions int               `json:"transient_upstream_min_enabled_subscriptions"`
}

type RoutingPolicyProbe struct {
	ActiveProbeEnabled         bool   `json:"active_probe_enabled"`
	ActiveProbeIntervalSeconds int    `json:"active_probe_interval_seconds"`
	ProbeModel                 string `json:"probe_model"`
	ProbeEndpointType          string `json:"probe_endpoint_type"`
	ProbeConnectTimeoutSeconds int    `json:"probe_connect_timeout_seconds"`
	ProbeMaxTimeSeconds        int    `json:"probe_max_time_seconds"`
	ProbeRetrySeconds          int    `json:"probe_retry_seconds"`
}

type RoutingPolicyPaygoNudge struct {
	Enabled                      bool `json:"enabled"`
	HoldSeconds                  int  `json:"hold_seconds"`
	CooldownSeconds              int  `json:"cooldown_seconds"`
	RecentWindowMinutes          int  `json:"recent_window_minutes"`
	MinSubscriptions             int  `json:"min_subscriptions"`
	MinRecentSubscriptionSuccess int  `json:"min_recent_subscription_success"`
	MaxRecentErrors              int  `json:"max_recent_errors"`
	MaxTransientErrors           int  `json:"max_transient_errors"`
	RequireNoSlowChannels        bool `json:"require_no_slow_channels"`
}

type RoutingPolicyPaygoHardFailure struct {
	Enabled         bool  `json:"enabled"`
	WindowMinutes   int   `json:"window_minutes"`
	Threshold       int   `json:"threshold"`
	ForceThreshold  int   `json:"force_threshold"`
	MaxPerRun       int   `json:"max_per_run"`
	RetrySeconds    int   `json:"retry_seconds"`
	RestorePriority int64 `json:"restore_priority"`
	RestoreWeight   uint  `json:"restore_weight"`
}

type RoutingPolicySlowChannel struct {
	SummaryEnabled               bool `json:"summary_enabled"`
	ScanIntervalSeconds          int  `json:"scan_interval_seconds"`
	WindowMinutes                int  `json:"window_minutes"`
	ConfirmWindowMinutes         int  `json:"confirm_window_minutes"`
	MinRequests                  int  `json:"min_requests"`
	P95Seconds                   int  `json:"p95_seconds"`
	SlowRequestSeconds           int  `json:"slow_request_seconds"`
	SlowRatioPercent             int  `json:"slow_ratio_percent"`
	AffinityClearEnabled         bool `json:"affinity_clear_enabled"`
	AffinityClearCooldownSeconds int  `json:"affinity_clear_cooldown_seconds"`
	WeightDegradeEnabled         bool `json:"weight_degrade_enabled"`
	AutoDisableEnabled           bool `json:"auto_disable_enabled"`
	AutoDisableMinP95Seconds     int  `json:"auto_disable_min_p95_seconds"`
	AutoDisableHoldSeconds       int  `json:"auto_disable_hold_seconds"`
	MinEnabledSubscriptions      int  `json:"min_enabled_subscriptions"`
	AutoDisableMaxPerRun         int  `json:"auto_disable_max_per_run"`
	AutoDisableCooldownSeconds   int  `json:"auto_disable_cooldown_seconds"`
}

type RoutingPolicySetting struct {
	Mode                   string                    `json:"mode"`
	RoleDetection          RoutingPolicyRoleDetection `json:"role_detection"`
	SubscriptionPolicy     RoutingPolicySubscription `json:"subscription_policy"`
	ProbePolicy            RoutingPolicyProbe        `json:"probe_policy"`
	PaygoNudgePolicy       RoutingPolicyPaygoNudge   `json:"paygo_nudge_policy"`
	PaygoHardFailurePolicy RoutingPolicyPaygoHardFailure `json:"paygo_hard_failure_policy"`
	SlowChannelPolicy      RoutingPolicySlowChannel  `json:"slow_channel_policy"`
}

var routingPolicySetting = RoutingPolicySetting{
	Mode: RoutingPolicyModeObserve,
	RoleDetection: RoutingPolicyRoleDetection{
		SubscriptionTags:         []string{"subscription"},
		PaygoTags:                []string{"paygo"},
		SubscriptionNamePatterns: []string{"\u8ba2\u9605"},
	},
	SubscriptionPolicy: RoutingPolicySubscription{
		StatusCodeMapping: map[string]string{
			"429": "500",
			"403": "500",
		},
		ErrorWindowMinutes:                       10,
		RateLimitDisableThreshold:                2,
		TransientUpstreamAffinityClearEnabled:    true,
		TransientUpstreamDisableEnabled:          true,
		TransientUpstreamDisableThreshold:        3,
		TransientUpstreamMinEnabledSubscriptions: 4,
	},
	ProbePolicy: RoutingPolicyProbe{
		ActiveProbeEnabled:         false,
		ActiveProbeIntervalSeconds: 900,
		ProbeModel:                 "gpt-5.5",
		ProbeEndpointType:          "openai-response-compact",
		ProbeConnectTimeoutSeconds: 3,
		ProbeMaxTimeSeconds:        12,
		ProbeRetrySeconds:          600,
	},
	PaygoNudgePolicy: RoutingPolicyPaygoNudge{
		Enabled:                      true,
		HoldSeconds:                  60,
		CooldownSeconds:              900,
		RecentWindowMinutes:          10,
		MinSubscriptions:             6,
		MinRecentSubscriptionSuccess: 1,
		MaxRecentErrors:              0,
		MaxTransientErrors:           0,
		RequireNoSlowChannels:        true,
	},
	PaygoHardFailurePolicy: RoutingPolicyPaygoHardFailure{
		Enabled:         true,
		WindowMinutes:   5,
		Threshold:       3,
		ForceThreshold:  20,
		MaxPerRun:       1,
		RetrySeconds:    1800,
		RestorePriority: 0,
		RestoreWeight:   1,
	},
	SlowChannelPolicy: RoutingPolicySlowChannel{
		SummaryEnabled:               true,
		ScanIntervalSeconds:          600,
		WindowMinutes:                10,
		ConfirmWindowMinutes:         30,
		MinRequests:                  20,
		P95Seconds:                   45,
		SlowRequestSeconds:           30,
		SlowRatioPercent:             20,
		AffinityClearEnabled:         true,
		AffinityClearCooldownSeconds: 60,
		WeightDegradeEnabled:         false,
		AutoDisableEnabled:           false,
		AutoDisableMinP95Seconds:     120,
		AutoDisableHoldSeconds:       900,
		MinEnabledSubscriptions:      4,
		AutoDisableMaxPerRun:         1,
		AutoDisableCooldownSeconds:   1800,
	},
}

func init() {
	config.GlobalConfig.Register("routing_policy_setting", &routingPolicySetting)
}

func normalizeRoutingPolicySetting(setting *RoutingPolicySetting) {
	if setting == nil {
		return
	}
	slowChannelPolicyMissing := isRoutingPolicySlowChannelPolicyZero(setting.SlowChannelPolicy)

	switch setting.Mode {
	case RoutingPolicyModeDisabled, RoutingPolicyModeObserve, RoutingPolicyModeEnforce:
	default:
		setting.Mode = RoutingPolicyModeObserve
	}

	if len(setting.RoleDetection.SubscriptionTags) == 0 {
		setting.RoleDetection.SubscriptionTags = []string{"subscription"}
	}
	if len(setting.RoleDetection.PaygoTags) == 0 {
		setting.RoleDetection.PaygoTags = []string{"paygo"}
	}
	if len(setting.RoleDetection.SubscriptionNamePatterns) == 0 {
		setting.RoleDetection.SubscriptionNamePatterns = []string{"\u8ba2\u9605"}
	}
	if len(setting.SubscriptionPolicy.StatusCodeMapping) == 0 {
		setting.SubscriptionPolicy.StatusCodeMapping = map[string]string{"429": "500", "403": "500"}
	}
	if setting.SubscriptionPolicy.ErrorWindowMinutes <= 0 {
		setting.SubscriptionPolicy.ErrorWindowMinutes = 10
	}
	if setting.SubscriptionPolicy.RateLimitDisableThreshold <= 0 {
		setting.SubscriptionPolicy.RateLimitDisableThreshold = 2
	}
	if setting.SubscriptionPolicy.TransientUpstreamDisableThreshold <= 0 {
		setting.SubscriptionPolicy.TransientUpstreamDisableThreshold = 3
	}
	if setting.SubscriptionPolicy.TransientUpstreamMinEnabledSubscriptions <= 0 {
		setting.SubscriptionPolicy.TransientUpstreamMinEnabledSubscriptions = 4
	}
	if setting.ProbePolicy.ActiveProbeIntervalSeconds <= 0 {
		setting.ProbePolicy.ActiveProbeIntervalSeconds = 900
	}
	if setting.ProbePolicy.ProbeModel == "" {
		setting.ProbePolicy.ProbeModel = "gpt-5.5"
	}
	if setting.ProbePolicy.ProbeEndpointType == "" {
		setting.ProbePolicy.ProbeEndpointType = "openai-response-compact"
	}
	if setting.ProbePolicy.ProbeConnectTimeoutSeconds <= 0 {
		setting.ProbePolicy.ProbeConnectTimeoutSeconds = 3
	}
	if setting.ProbePolicy.ProbeMaxTimeSeconds <= 0 {
		setting.ProbePolicy.ProbeMaxTimeSeconds = 12
	}
	if setting.ProbePolicy.ProbeRetrySeconds <= 0 {
		setting.ProbePolicy.ProbeRetrySeconds = 600
	}
	if setting.PaygoNudgePolicy.HoldSeconds <= 0 {
		setting.PaygoNudgePolicy.HoldSeconds = 60
	}
	if setting.PaygoNudgePolicy.CooldownSeconds <= 0 {
		setting.PaygoNudgePolicy.CooldownSeconds = 900
	}
	if setting.PaygoNudgePolicy.RecentWindowMinutes <= 0 {
		setting.PaygoNudgePolicy.RecentWindowMinutes = 10
	}
	if setting.PaygoNudgePolicy.MinSubscriptions <= 0 {
		setting.PaygoNudgePolicy.MinSubscriptions = 6
	}
	if setting.PaygoNudgePolicy.MinRecentSubscriptionSuccess <= 0 {
		setting.PaygoNudgePolicy.MinRecentSubscriptionSuccess = 1
	}
	if setting.PaygoHardFailurePolicy.WindowMinutes <= 0 {
		setting.PaygoHardFailurePolicy.WindowMinutes = 5
	}
	if setting.PaygoHardFailurePolicy.Threshold <= 0 {
		setting.PaygoHardFailurePolicy.Threshold = 3
	}
	if setting.PaygoHardFailurePolicy.ForceThreshold <= 0 {
		setting.PaygoHardFailurePolicy.ForceThreshold = 20
	}
	if setting.PaygoHardFailurePolicy.MaxPerRun <= 0 {
		setting.PaygoHardFailurePolicy.MaxPerRun = 1
	}
	if setting.PaygoHardFailurePolicy.RetrySeconds <= 0 {
		setting.PaygoHardFailurePolicy.RetrySeconds = 1800
	}
	if setting.PaygoHardFailurePolicy.RestoreWeight == 0 {
		setting.PaygoHardFailurePolicy.RestoreWeight = 1
	}
	if setting.SlowChannelPolicy.ScanIntervalSeconds <= 0 {
		setting.SlowChannelPolicy.ScanIntervalSeconds = 600
	}
	if slowChannelPolicyMissing {
		setting.SlowChannelPolicy.SummaryEnabled = true
	}
	if setting.SlowChannelPolicy.WindowMinutes <= 0 {
		setting.SlowChannelPolicy.WindowMinutes = 10
	}
	if setting.SlowChannelPolicy.ConfirmWindowMinutes <= 0 {
		setting.SlowChannelPolicy.ConfirmWindowMinutes = 30
	}
	if setting.SlowChannelPolicy.MinRequests <= 0 {
		setting.SlowChannelPolicy.MinRequests = 20
	}
	if setting.SlowChannelPolicy.P95Seconds <= 0 {
		setting.SlowChannelPolicy.P95Seconds = 45
	}
	if setting.SlowChannelPolicy.SlowRequestSeconds <= 0 {
		setting.SlowChannelPolicy.SlowRequestSeconds = 30
	}
	if setting.SlowChannelPolicy.SlowRatioPercent <= 0 {
		setting.SlowChannelPolicy.SlowRatioPercent = 20
	}
	if setting.SlowChannelPolicy.AffinityClearCooldownSeconds <= 0 {
		setting.SlowChannelPolicy.AffinityClearCooldownSeconds = 60
	}
	if setting.SlowChannelPolicy.AutoDisableMinP95Seconds <= 0 {
		setting.SlowChannelPolicy.AutoDisableMinP95Seconds = 120
	}
	if setting.SlowChannelPolicy.AutoDisableHoldSeconds <= 0 {
		setting.SlowChannelPolicy.AutoDisableHoldSeconds = 900
	}
	if setting.SlowChannelPolicy.MinEnabledSubscriptions <= 0 {
		setting.SlowChannelPolicy.MinEnabledSubscriptions = 4
	}
	if setting.SlowChannelPolicy.AutoDisableMaxPerRun <= 0 {
		setting.SlowChannelPolicy.AutoDisableMaxPerRun = 1
	}
	if setting.SlowChannelPolicy.AutoDisableCooldownSeconds <= 0 {
		setting.SlowChannelPolicy.AutoDisableCooldownSeconds = 1800
	}
}

func isRoutingPolicySlowChannelPolicyZero(policy RoutingPolicySlowChannel) bool {
	return !policy.SummaryEnabled &&
		policy.ScanIntervalSeconds == 0 &&
		policy.WindowMinutes == 0 &&
		policy.ConfirmWindowMinutes == 0 &&
		policy.MinRequests == 0 &&
		policy.P95Seconds == 0 &&
		policy.SlowRequestSeconds == 0 &&
		policy.SlowRatioPercent == 0 &&
		!policy.AffinityClearEnabled &&
		policy.AffinityClearCooldownSeconds == 0 &&
		!policy.WeightDegradeEnabled &&
		!policy.AutoDisableEnabled &&
		policy.AutoDisableMinP95Seconds == 0 &&
		policy.AutoDisableHoldSeconds == 0 &&
		policy.MinEnabledSubscriptions == 0 &&
		policy.AutoDisableMaxPerRun == 0 &&
		policy.AutoDisableCooldownSeconds == 0
}

func GetRoutingPolicySetting() *RoutingPolicySetting {
	normalizeRoutingPolicySetting(&routingPolicySetting)
	return &routingPolicySetting
}
