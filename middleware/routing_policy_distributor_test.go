package middleware

import (
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
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSpecificChannelRejectsUnsupportedRequestPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	channel := &model.Channel{
		Id:     7201,
		Type:   constant.ChannelTypeAdvancedCustom,
		Status: common.ChannelStatusEnabled,
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{
		AdvancedCustom: &dto.AdvancedCustomConfig{
			Routes: []dto.AdvancedCustomRoute{{IncomingPath: "/v1/chat/completions"}},
		},
	})

	require.False(t, isSpecificChannelUsable(c, channel))
}

func TestSpecificChannelRejectsUnhealthyInEnforceMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	routingpolicy.ResetForTest()

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	t.Cleanup(func() {
		setting.Mode = origMode
		routingpolicy.ResetForTest()
	})
	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	routingpolicy.MarkFailure(7202, http.StatusBadGateway, "runtime_cooldown", time.Minute)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	channel := &model.Channel{
		Id:     7202,
		Type:   constant.ChannelTypeOpenAI,
		Status: common.ChannelStatusEnabled,
	}

	require.False(t, isSpecificChannelUsable(c, channel))
}

func TestSpecificChannelAllowsUnhealthyInObserveMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	routingpolicy.ResetForTest()

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	t.Cleanup(func() {
		setting.Mode = origMode
		routingpolicy.ResetForTest()
	})
	setting.Mode = operation_setting.RoutingPolicyModeObserve
	routingpolicy.MarkFailure(7203, http.StatusBadGateway, "runtime_cooldown", time.Minute)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	channel := &model.Channel{
		Id:     7203,
		Type:   constant.ChannelTypeOpenAI,
		Status: common.ChannelStatusEnabled,
	}

	require.True(t, isSpecificChannelUsable(c, channel))
}

func TestAffinityChannelRejectsUnsupportedRequestPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	channel := &model.Channel{
		Id:     7301,
		Type:   constant.ChannelTypeAdvancedCustom,
		Status: common.ChannelStatusEnabled,
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{
		AdvancedCustom: &dto.AdvancedCustomConfig{
			Routes: []dto.AdvancedCustomRoute{{IncomingPath: "/v1/chat/completions"}},
		},
	})

	usability := isAffinityChannelUsable(c, channel)

	require.False(t, usability.usable)
}

func TestAffinityChannelRejectsUnhealthyInEnforceMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	routingpolicy.ResetForTest()

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	t.Cleanup(func() {
		setting.Mode = origMode
		routingpolicy.ResetForTest()
	})
	setting.Mode = operation_setting.RoutingPolicyModeEnforce
	routingpolicy.MarkFailure(7302, http.StatusBadGateway, "runtime_cooldown", time.Minute)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	channel := &model.Channel{
		Id:     7302,
		Type:   constant.ChannelTypeOpenAI,
		Status: common.ChannelStatusEnabled,
	}

	usability := isAffinityChannelUsable(c, channel)

	require.False(t, usability.healthy)
	require.False(t, usability.usable)
}

func TestAffinityChannelAllowsUnhealthyInObserveMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	routingpolicy.ResetForTest()

	setting := operation_setting.GetRoutingPolicySetting()
	origMode := setting.Mode
	t.Cleanup(func() {
		setting.Mode = origMode
		routingpolicy.ResetForTest()
	})
	setting.Mode = operation_setting.RoutingPolicyModeObserve
	routingpolicy.MarkFailure(7303, http.StatusBadGateway, "runtime_cooldown", time.Minute)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	channel := &model.Channel{
		Id:     7303,
		Type:   constant.ChannelTypeOpenAI,
		Status: common.ChannelStatusEnabled,
	}

	usability := isAffinityChannelUsable(c, channel)

	require.False(t, usability.healthy)
	require.True(t, usability.usable)
}
