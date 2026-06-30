package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/routingpolicy"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestShouldRetryDoesNotRetrySpecificChannelChannelError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("specific_channel_id", 123)

	err := types.NewError(errors.New("upstream failed"), types.ErrorCodeChannelInvalidKey)

	require.False(t, shouldRetry(c, err, 1))
}

func TestValidateLockedTaskChannelRejectsUnsupportedRequestPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil)

	channel := &model.Channel{
		Id:     7101,
		Type:   constant.ChannelTypeAdvancedCustom,
		Status: common.ChannelStatusEnabled,
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{
		AdvancedCustom: &dto.AdvancedCustomConfig{
			Routes: []dto.AdvancedCustomRoute{{IncomingPath: "/v1/chat/completions"}},
		},
	})

	taskErr := validateLockedTaskChannel(c, channel)

	require.NotNil(t, taskErr)
	require.Equal(t, "locked_channel_path_unsupported", taskErr.Code)
	require.Equal(t, http.StatusServiceUnavailable, taskErr.StatusCode)
}

func TestValidateLockedTaskChannelRejectsUnhealthyInEnforceMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	routingpolicy.ResetForTest()

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	t.Cleanup(func() {
		setting.Mode = origMode
		routingpolicy.ResetForTest()
	})
	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	routingpolicy.MarkFailure(7102, http.StatusBadGateway, "runtime_cooldown", time.Minute)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	channel := &model.Channel{
		Id:     7102,
		Type:   constant.ChannelTypeOpenAI,
		Status: common.ChannelStatusEnabled,
	}

	taskErr := validateLockedTaskChannel(c, channel)

	require.NotNil(t, taskErr)
	require.Equal(t, "locked_channel_unhealthy", taskErr.Code)
	require.Equal(t, http.StatusServiceUnavailable, taskErr.StatusCode)
}

func TestValidateLockedTaskChannelAllowsUnhealthyInObserveMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	routingpolicy.ResetForTest()

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	t.Cleanup(func() {
		setting.Mode = origMode
		routingpolicy.ResetForTest()
	})
	setting.Mode = operation_setting.RoutingPolicyModeObserve
	routingpolicy.MarkFailure(7103, http.StatusBadGateway, "runtime_cooldown", time.Minute)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	channel := &model.Channel{
		Id:     7103,
		Type:   constant.ChannelTypeOpenAI,
		Status: common.ChannelStatusEnabled,
	}

	require.Nil(t, validateLockedTaskChannel(c, channel))
}
