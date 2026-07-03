package routingpolicy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkSuccessClearsPendingRestoreState(t *testing.T) {
	ResetForTest()

	channelID := 123
	storeState(channelID, HealthState{
		CooldownUntil:    999,
		Reason:           "paygo_hard_failure",
		LastSuccessAt:     100,
		SuccessCount:      2,
		LastStatusCode:    500,
		RestorePhase:      restorePhaseProbeFailedHold,
		RestoreReason:     "paygo_hard_failure",
		RestoreHoldUntil:  200,
		LastProbeAt:       55,
		LastProbeResult:   probeResultFailed,
		LastRestoreAt:     77,
	})

	MarkSuccess(channelID)

	state, ok := loadState(channelID)
	require.True(t, ok)

	assert.Equal(t, int64(0), state.CooldownUntil)
	assert.Empty(t, state.Reason)
	assert.Equal(t, 0, state.LastStatusCode)
	assert.Empty(t, state.RestorePhase)
	assert.Empty(t, state.RestoreReason)
	assert.Equal(t, int64(0), state.RestoreHoldUntil)
	assert.Empty(t, state.LastProbeResult)
	assert.Equal(t, int64(55), state.LastProbeAt)
	assert.Equal(t, int64(77), state.LastRestoreAt)
	assert.Greater(t, state.LastSuccessAt, int64(100))
	assert.Equal(t, 3, state.SuccessCount)
}

func TestNoteSummaryNextProbeAtKeepsEarliestHold(t *testing.T) {
	var summary Summary

	noteSummaryNextProbeAt(&summary, 300)
	noteSummaryNextProbeAt(&summary, 180)
	noteSummaryNextProbeAt(&summary, 260)
	noteSummaryNextProbeAt(&summary, 0)

	assert.Equal(t, int64(180), summary.NextProbeAt)
}
