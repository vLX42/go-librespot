//go:build test_unit

package spclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	librespot "github.com/devgianlu/go-librespot"
	"github.com/stretchr/testify/require"
)

// webSearchClient points the Web API at a stub server for one test.
func webSearchClient(t *testing.T, handler http.HandlerFunc) *Spclient {
	t.Helper()

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	base, err := url.Parse(server.URL + "/v1/")
	require.NoError(t, err)

	saved := webApiBaseUrl
	webApiBaseUrl = *base
	t.Cleanup(func() { webApiBaseUrl = saved })

	client := server.Client()
	client.Timeout = 5 * time.Second

	return &Spclient{
		log:         &librespot.NullLogger{},
		client:      client,
		accessToken: func(context.Context, bool) (string, error) { return "token", nil },
	}
}

func TestWebSearchSendsTheQuery(t *testing.T) {
	var got *http.Request
	c := webSearchClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(context.Background())
		_, _ = w.Write([]byte(`{"tracks":{"total":1,"items":[null,{"uri":"spotify:track:x","type":"track","name":"X",
			"artists":[{"uri":"spotify:artist:a","name":"A"}],"album":{"uri":"spotify:album:b","name":"B","images":[]},
			"duration_ms":1000,"explicit":false}]}}`))
	})

	res, err := c.WebSearch(t.Context(), "daft punk", []string{"track", "album"}, 3, 6)
	require.NoError(t, err)

	require.Equal(t, "/v1/search", got.URL.Path)
	require.Equal(t, url.Values{
		"q":      {"daft punk"},
		"type":   {"track,album"},
		"limit":  {"3"},
		"offset": {"6"},
	}, got.URL.Query())
	require.Equal(t, "Bearer token", got.Header.Get("Authorization"))

	require.Nil(t, res.Albums)
	require.Equal(t, 1, res.Tracks.Total)
	require.Len(t, res.Tracks.Items, 2)
	require.Nil(t, res.Tracks.Items[0])
	require.Equal(t, "spotify:track:x", res.Tracks.Items[1].Uri)
	require.Equal(t, "B", res.Tracks.Items[1].Album.Name)
}

func TestWebSearchRateLimited(t *testing.T) {
	calls := 0
	c := webSearchClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := c.WebSearch(t.Context(), "x", []string{"track"}, 5, 0)

	var rateLimited *RateLimitedError
	require.True(t, errors.As(err, &rateLimited))
	require.Equal(t, 7*time.Second, rateLimited.RetryAfter)
	require.Equal(t, 1, calls)
}

func TestWebSearchStatusError(t *testing.T) {
	c := webSearchClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	_, err := c.WebSearch(t.Context(), "x", []string{"track"}, 5, 0)

	var status *StatusError
	require.True(t, errors.As(err, &status))
	require.Equal(t, http.StatusBadRequest, status.StatusCode)
}
