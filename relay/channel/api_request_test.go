package channel

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestProcessHeaderOverride_ChannelTestSkipsPassthroughRules(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"*": "",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Empty(t, headers)
}

func TestProcessHeaderOverride_ChannelTestSkipsClientHeaderPlaceholder(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"X-Upstream-Trace": "{client_header:X-Trace-Id}",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	_, ok := headers["x-upstream-trace"]
	require.False(t, ok)
}

func TestProcessHeaderOverride_NonTestKeepsClientHeaderPlaceholder(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: false,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"X-Upstream-Trace": "{client_header:X-Trace-Id}",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "trace-123", headers["x-upstream-trace"])
}

func TestProcessHeaderOverride_RuntimeOverrideIsFinalHeaderMap(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	info := &relaycommon.RelayInfo{
		IsChannelTest:             false,
		UseRuntimeHeadersOverride: true,
		RuntimeHeadersOverride: map[string]any{
			"x-static":  "runtime-value",
			"x-runtime": "runtime-only",
		},
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"X-Static": "legacy-value",
				"X-Legacy": "legacy-only",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "runtime-value", headers["x-static"])
	require.Equal(t, "runtime-only", headers["x-runtime"])
	_, exists := headers["x-legacy"]
	require.False(t, exists)
}

func TestProcessHeaderOverride_PassthroughSkipsAcceptEncoding(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")
	ctx.Request.Header.Set("Accept-Encoding", "gzip")

	info := &relaycommon.RelayInfo{
		IsChannelTest: false,
		ChannelMeta: &relaycommon.ChannelMeta{
			HeadersOverride: map[string]any{
				"*": "",
			},
		},
	}

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "trace-123", headers["x-trace-id"])

	_, hasAcceptEncoding := headers["accept-encoding"]
	require.False(t, hasAcceptEncoding)
}

func TestProcessHeaderOverride_PassHeadersTemplateSetsRuntimeHeaders(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	ctx.Request.Header.Set("Originator", "Codex CLI")
	ctx.Request.Header.Set("Session_id", "sess-123")

	info := &relaycommon.RelayInfo{
		IsChannelTest: false,
		RequestHeaders: map[string]string{
			"Originator": "Codex CLI",
			"Session_id": "sess-123",
		},
		ChannelMeta: &relaycommon.ChannelMeta{
			ParamOverride: map[string]any{
				"operations": []any{
					map[string]any{
						"mode":  "pass_headers",
						"value": []any{"Originator", "Session_id", "X-Codex-Beta-Features"},
					},
				},
			},
			HeadersOverride: map[string]any{
				"X-Static": "legacy-value",
			},
		},
	}

	_, err := relaycommon.ApplyParamOverrideWithRelayInfo([]byte(`{"model":"gpt-4.1"}`), info)
	require.NoError(t, err)
	require.True(t, info.UseRuntimeHeadersOverride)
	require.Equal(t, "Codex CLI", info.RuntimeHeadersOverride["originator"])
	require.Equal(t, "sess-123", info.RuntimeHeadersOverride["session_id"])
	_, exists := info.RuntimeHeadersOverride["x-codex-beta-features"]
	require.False(t, exists)
	require.Equal(t, "legacy-value", info.RuntimeHeadersOverride["x-static"])

	headers, err := processHeaderOverride(info, ctx)
	require.NoError(t, err)
	require.Equal(t, "Codex CLI", headers["originator"])
	require.Equal(t, "sess-123", headers["session_id"])
	_, exists = headers["x-codex-beta-features"]
	require.False(t, exists)

	upstreamReq := httptest.NewRequest(http.MethodPost, "https://example.com/v1/responses", nil)
	applyHeaderOverrideToRequest(upstreamReq, headers)
	require.Equal(t, "Codex CLI", upstreamReq.Header.Get("Originator"))
	require.Equal(t, "sess-123", upstreamReq.Header.Get("Session_id"))
	require.Empty(t, upstreamReq.Header.Get("X-Codex-Beta-Features"))
}

func TestDoRequestUsesRequestContextConnectTimeoutForTLSHandshake(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("ALL_PROXY", "")
	t.Setenv("NO_PROXY", "*")

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	done := make(chan struct{})
	var mu sync.Mutex
	var accepted []net.Conn
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			accepted = append(accepted, conn)
			mu.Unlock()
			go func(conn net.Conn) {
				<-done
				_ = conn.Close()
			}(conn)
		}
	}()
	t.Cleanup(func() {
		close(done)
		_ = listener.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, conn := range accepted {
			_ = conn.Close()
		}
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader("{}"))

	req, err := http.NewRequest(http.MethodPost, "https://"+listener.Addr().String()+"/v1/chat/completions", strings.NewReader("{}"))
	require.NoError(t, err)
	requestCtx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
	defer cancel()
	req = req.WithContext(service.WithHTTPClientConnectTimeout(requestCtx, 50*time.Millisecond))

	start := time.Now()
	resp, err := DoRequest(ctx, req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}})
	elapsed := time.Since(start)

	require.Error(t, err)
	require.Nil(t, resp)
	require.Less(t, elapsed, 2*time.Second)
}

func TestDoRequestUsesConfiguredHTTPProxyClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("ALL_PROXY", "")
	t.Setenv("NO_PROXY", "*")
	service.ResetProxyClientCache()
	t.Cleanup(service.ResetProxyClientCache)

	var proxyHits atomic.Int32
	var proxyMu sync.Mutex
	var proxiedURLs []string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHits.Add(1)
		proxyMu.Lock()
		proxiedURLs = append(proxiedURLs, r.URL.String())
		proxyMu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer proxy.Close()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", strings.NewReader(""))

	req, err := http.NewRequest(http.MethodGet, "http://upstream.example/v1/models", strings.NewReader(""))
	require.NoError(t, err)
	req = req.WithContext(service.WithHTTPClientConnectTimeout(req.Context(), time.Second))
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelSetting: dto.ChannelSettings{
				Proxy: proxy.URL,
			},
		},
	}

	resp, err := DoRequest(ctx, req, info)
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Equal(t, int32(1), proxyHits.Load())
	proxyMu.Lock()
	gotProxiedURLs := append([]string(nil), proxiedURLs...)
	proxyMu.Unlock()
	require.Equal(t, []string{"http://upstream.example/v1/models"}, gotProxiedURLs)
}

func TestDoRequestUsesRequestContextConnectTimeoutForSOCKSProxyHandshake(t *testing.T) {
	for _, scheme := range []string{"socks5", "socks5h"} {
		t.Run(scheme, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			t.Setenv("HTTP_PROXY", "")
			t.Setenv("HTTPS_PROXY", "")
			t.Setenv("ALL_PROXY", "")
			t.Setenv("NO_PROXY", "*")
			service.ResetProxyClientCache()
			t.Cleanup(service.ResetProxyClientCache)

			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)

			done := make(chan struct{})
			var mu sync.Mutex
			var accepted []net.Conn
			go func() {
				for {
					conn, err := listener.Accept()
					if err != nil {
						return
					}
					mu.Lock()
					accepted = append(accepted, conn)
					mu.Unlock()
					go func(conn net.Conn) {
						<-done
						_ = conn.Close()
					}(conn)
				}
			}()
			t.Cleanup(func() {
				close(done)
				_ = listener.Close()
				mu.Lock()
				defer mu.Unlock()
				for _, conn := range accepted {
					_ = conn.Close()
				}
			})

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", strings.NewReader(""))

			req, err := http.NewRequest(http.MethodGet, "http://upstream.example/v1/models", strings.NewReader(""))
			require.NoError(t, err)
			requestCtx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
			defer cancel()
			req = req.WithContext(service.WithHTTPClientConnectTimeout(requestCtx, 50*time.Millisecond))
			info := &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelSetting: dto.ChannelSettings{
						Proxy: scheme + "://" + listener.Addr().String(),
					},
				},
			}

			start := time.Now()
			resp, err := DoRequest(ctx, req, info)
			elapsed := time.Since(start)

			require.Error(t, err)
			require.Nil(t, resp)
			require.Less(t, elapsed, 2*time.Second)
		})
	}
}
