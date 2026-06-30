package controller

import (
	"context"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

func init() {
	service.RegisterRoutingPolicyProbeExecutor(runRoutingPolicyProbe)
}

func runRoutingPolicyProbe(ctx context.Context, channel *model.Channel, probe operation_setting.RoutingPolicyProbe) bool {
	if channel == nil {
		return false
	}
	testUserID, err := resolveChannelTestUserID(nil)
	if err != nil {
		return false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := time.Duration(probe.ProbeMaxTimeSeconds) * time.Second
	if timeout <= 0 {
		timeout = 12 * time.Second
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result := testChannel(probeCtx, channel, testUserID, probe.ProbeModel, probe.ProbeEndpointType, false)
	return result.localErr == nil && result.newAPIError == nil
}
