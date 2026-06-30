package model

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetRoutingPolicyHooksForTest() {
	routingPolicyHooks = RoutingPolicyHooks{
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
}

func TestGetRandomSatisfiedChannelSkipsUnhealthyAndFailOpens(t *testing.T) {
	truncateTables(t)

	origMemory := common.MemoryCacheEnabled
	t.Cleanup(func() {
		common.MemoryCacheEnabled = origMemory
		resetRoutingPolicyHooksForTest()
	})

	common.MemoryCacheEnabled = true
	RegisterRoutingPolicyHooks(RoutingPolicyHooks{
		IsEnabled:     func() bool { return true },
		IsEnforceMode: func() bool { return true },
		IsChannelHealthy: func(channelID int, now time.Time) RoutingPolicyDecision {
			return RoutingPolicyDecision{Healthy: channelID != 4101, Reason: "cooldown"}
		},
	})

	priority := int64(10)
	weight := uint(10)
	tag := "subscription"
	require.NoError(t, DB.Create(&Channel{
		Id:       4101,
		Name:     "cache-unhealthy",
		Group:    "default",
		Models:   "gpt-5",
		Status:   common.ChannelStatusEnabled,
		Priority: &priority,
		Weight:   &weight,
		Tag:      &tag,
	}).Error)
	require.NoError(t, DB.Create(&Channel{
		Id:       4102,
		Name:     "cache-healthy",
		Group:    "default",
		Models:   "gpt-5",
		Status:   common.ChannelStatusEnabled,
		Priority: &priority,
		Weight:   &weight,
		Tag:      &tag,
	}).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     "default",
		Model:     "gpt-5",
		ChannelId: 4101,
		Enabled:   true,
		Priority:  &priority,
		Weight:    weight,
		Tag:       &tag,
	}).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     "default",
		Model:     "gpt-5",
		ChannelId: 4102,
		Enabled:   true,
		Priority:  &priority,
		Weight:    weight,
		Tag:       &tag,
	}).Error)

	InitChannelCache()

	channel, err := GetRandomSatisfiedChannel("default", "gpt-5", 0, "")
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, 4102, channel.Id)

	RegisterRoutingPolicyHooks(RoutingPolicyHooks{
		IsEnabled:     func() bool { return true },
		IsEnforceMode: func() bool { return true },
		IsChannelHealthy: func(channelID int, now time.Time) RoutingPolicyDecision {
			return RoutingPolicyDecision{Healthy: false, Reason: "cooldown"}
		},
	})

	channel, err = GetRandomSatisfiedChannel("default", "gpt-5", 0, "")
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Contains(t, []int{4101, 4102}, channel.Id)
}

func TestGetChannelSkipsUnhealthyAndFailOpens(t *testing.T) {
	truncateTables(t)

	origMemory := common.MemoryCacheEnabled
	t.Cleanup(func() {
		common.MemoryCacheEnabled = origMemory
		resetRoutingPolicyHooksForTest()
	})

	common.MemoryCacheEnabled = false
	RegisterRoutingPolicyHooks(RoutingPolicyHooks{
		IsEnabled:     func() bool { return true },
		IsEnforceMode: func() bool { return true },
		IsChannelHealthy: func(channelID int, now time.Time) RoutingPolicyDecision {
			return RoutingPolicyDecision{Healthy: channelID != 4201, Reason: "cooldown"}
		},
	})

	priority := int64(10)
	weight := uint(10)
	tag := "subscription"
	require.NoError(t, DB.Create(&Channel{
		Id:       4201,
		Name:     "db-unhealthy",
		Group:    "default",
		Models:   "gpt-5",
		Status:   common.ChannelStatusEnabled,
		Priority: &priority,
		Weight:   &weight,
		Tag:      &tag,
	}).Error)
	require.NoError(t, DB.Create(&Channel{
		Id:       4202,
		Name:     "db-healthy",
		Group:    "default",
		Models:   "gpt-5",
		Status:   common.ChannelStatusEnabled,
		Priority: &priority,
		Weight:   &weight,
		Tag:      &tag,
	}).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     "default",
		Model:     "gpt-5",
		ChannelId: 4201,
		Enabled:   true,
		Priority:  &priority,
		Weight:    weight,
		Tag:       &tag,
	}).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     "default",
		Model:     "gpt-5",
		ChannelId: 4202,
		Enabled:   true,
		Priority:  &priority,
		Weight:    weight,
		Tag:       &tag,
	}).Error)

	channel, err := GetChannel("default", "gpt-5", 0, "")
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, 4202, channel.Id)

	RegisterRoutingPolicyHooks(RoutingPolicyHooks{
		IsEnabled:     func() bool { return true },
		IsEnforceMode: func() bool { return true },
		IsChannelHealthy: func(channelID int, now time.Time) RoutingPolicyDecision {
			return RoutingPolicyDecision{Healthy: false, Reason: "cooldown"}
		},
	})

	channel, err = GetChannel("default", "gpt-5", 0, "")
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Contains(t, []int{4201, 4202}, channel.Id)
}
