package controller

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/require"
)

func TestRunRoutingPolicyProbeUsesConnectTimeoutForTLSHandshake(t *testing.T) {
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("ALL_PROXY", "")
	t.Setenv("NO_PROXY", "*")

	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       9001,
		Username: "routing-probe-root",
		Password: "password123",
		Role:     common.RoleRootUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}).Error)

	listener, accepted, cleanup := newStalledProbeListener(t)
	defer cleanup()

	baseURL := "https://" + listener.Addr().String()
	channel := &model.Channel{
		Id:      9901,
		Name:    "routing-probe-timeout",
		Type:    constant.ChannelTypeOpenAI,
		Status:  common.ChannelStatusEnabled,
		Key:     "test-key",
		BaseURL: &baseURL,
		Group:   "default",
	}

	start := time.Now()
	ok := runRoutingPolicyProbe(context.Background(), channel, operation_setting.RoutingPolicyProbe{
		ProbeModel:                 "gpt-4o-mini",
		ProbeConnectTimeoutSeconds: 1,
		ProbeMaxTimeSeconds:        5,
	})
	elapsed := time.Since(start)

	require.False(t, ok)
	require.Less(t, elapsed, 3*time.Second)
	require.GreaterOrEqual(t, accepted.Load(), int32(1))
}

func TestRunRoutingPolicyProbeUsesConnectTimeoutForSOCKSHandshake(t *testing.T) {
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("ALL_PROXY", "")
	t.Setenv("NO_PROXY", "*")

	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       9002,
		Username: "routing-probe-proxy-root",
		Password: "password123",
		Role:     common.RoleRootUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}).Error)

	listener, accepted, cleanup := newStalledProbeListener(t)
	defer cleanup()

	baseURL := "https://upstream.example"
	channel := &model.Channel{
		Id:      9902,
		Name:    "routing-probe-socks-timeout",
		Type:    constant.ChannelTypeOpenAI,
		Status:  common.ChannelStatusEnabled,
		Key:     "test-key",
		BaseURL: &baseURL,
		Group:   "default",
	}
	channel.SetSetting(dto.ChannelSettings{
		Proxy: fmt.Sprintf("socks5://%s", listener.Addr().String()),
	})

	start := time.Now()
	ok := runRoutingPolicyProbe(context.Background(), channel, operation_setting.RoutingPolicyProbe{
		ProbeModel:                 "gpt-4o-mini",
		ProbeConnectTimeoutSeconds: 1,
		ProbeMaxTimeSeconds:        5,
	})
	elapsed := time.Since(start)

	require.False(t, ok)
	require.Less(t, elapsed, 3*time.Second)
	require.GreaterOrEqual(t, accepted.Load(), int32(1))
}

func newStalledProbeListener(t *testing.T) (net.Listener, *atomic.Int32, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	done := make(chan struct{})
	var mu sync.Mutex
	accepted := make([]net.Conn, 0, 4)
	var acceptedCount atomic.Int32
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			acceptedCount.Add(1)
			mu.Lock()
			accepted = append(accepted, conn)
			mu.Unlock()
			go func(conn net.Conn) {
				<-done
				_ = conn.Close()
			}(conn)
		}
	}()

	cleanup := func() {
		close(done)
		_ = listener.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, conn := range accepted {
			_ = conn.Close()
		}
	}

	return listener, &acceptedCount, cleanup
}
