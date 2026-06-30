package service

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetHttpClientForContextReturnsBaseClientWithoutTimeout(t *testing.T) {
	origClient := httpClient
	t.Cleanup(func() {
		httpClient = origClient
	})
	baseTransport := &http.Transport{}
	httpClient = &http.Client{Transport: baseTransport}

	client, err := GetHttpClientForContext(context.Background(), "")
	require.NoError(t, err)
	require.Same(t, httpClient, client)
}

func TestGetHttpClientForContextClonesTransportWithConnectTimeout(t *testing.T) {
	origClient := httpClient
	t.Cleanup(func() {
		httpClient = origClient
	})
	baseTransport := &http.Transport{}
	httpClient = &http.Client{Transport: baseTransport}

	ctx := WithHTTPClientConnectTimeout(context.Background(), 2*time.Second)
	client, err := GetHttpClientForContext(ctx, "")
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotSame(t, httpClient, client)

	clonedTransport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotSame(t, baseTransport, clonedTransport)
	require.NotNil(t, clonedTransport.DialContext)
	require.Equal(t, 2*time.Second, clonedTransport.TLSHandshakeTimeout)
	require.Nil(t, baseTransport.DialContext)
}

func TestCloneHTTPClientWithConnectTimeoutPreservesExistingDialContext(t *testing.T) {
	expectedErr := errors.New("proxy dialer used")
	connectTimeout := 2 * time.Second
	deadlineSeen := false
	var deadlineRemaining time.Duration
	baseTransport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			deadline, ok := ctx.Deadline()
			deadlineSeen = ok
			deadlineRemaining = time.Until(deadline)
			return nil, expectedErr
		},
	}
	baseClient := &http.Client{Transport: baseTransport}

	client := cloneHTTPClientWithConnectTimeout(baseClient, connectTimeout)
	clonedTransport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotSame(t, baseTransport, clonedTransport)

	_, err := clonedTransport.DialContext(context.Background(), "proxy-test", "example.com:443")
	require.ErrorIs(t, err, expectedErr)
	require.True(t, deadlineSeen)
	require.Greater(t, deadlineRemaining, time.Duration(0))
	require.LessOrEqual(t, deadlineRemaining, connectTimeout)
	require.Equal(t, connectTimeout, clonedTransport.TLSHandshakeTimeout)
}
