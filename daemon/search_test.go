//go:build test_unit

package daemon

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/devgianlu/go-librespot/spclient"
	"github.com/stretchr/testify/require"
)

func TestParseSearchTypes(t *testing.T) {
	types, ok := parseSearchTypes(" Track,album ,track")
	require.True(t, ok)
	require.Equal(t, []string{"track", "album"}, types)

	for _, param := range []string{"", "track,", "song", "track,,album"} {
		_, ok := parseSearchTypes(param)
		require.False(t, ok, param)
	}
}

func TestSearchError(t *testing.T) {
	require.ErrorIs(t, searchError(&spclient.RateLimitedError{RetryAfter: time.Second}), ErrTooManyRequests)
	require.ErrorIs(t, searchError(&spclient.StatusError{StatusCode: http.StatusBadRequest}), ErrBadRequest)

	err := searchError(&spclient.StatusError{StatusCode: http.StatusBadGateway})
	require.False(t, errors.Is(err, ErrBadRequest))
	require.False(t, errors.Is(err, ErrTooManyRequests))
}

func TestSearchResults(t *testing.T) {
	explicit := true
	res := searchResults(&spclient.WebSearchResponse{
		Tracks: &spclient.WebSearchPage{Total: 42, Items: []*spclient.WebSearchItem{
			nil,
			{Type: "track", Name: "no uri"},
			{
				Uri: "spotify:track:t", Type: "track", Name: "Song",
				Artists:    []spclient.WebNamed{{Name: "A"}, {Name: "B"}},
				Album:      &spclient.WebAlbum{Uri: "spotify:album:a", Name: "Album", Images: []spclient.WebImage{{Url: "small", Width: 64}, {Url: "big", Width: 640}}},
				DurationMs: 1234,
				Explicit:   &explicit,
			},
		}},
		Playlists: &spclient.WebSearchPage{Total: 1, Items: []*spclient.WebSearchItem{{
			Uri: "spotify:playlist:p", Type: "playlist", Name: "Mix",
			Owner:  &spclient.WebOwner{DisplayName: "me"},
			Images: []spclient.WebImage{{Url: "cover"}},
		}}},
	})

	require.Nil(t, res.Albums)
	require.Equal(t, 42, res.Tracks.Total)
	require.Len(t, res.Tracks.Items, 1)

	track := res.Tracks.Items[0]
	require.Equal(t, "spotify:track:t", track.Uri)
	require.Equal(t, []string{"A", "B"}, *track.ArtistNames)
	require.Equal(t, "Album", *track.AlbumName)
	require.Equal(t, "spotify:album:a", *track.AlbumUri)
	require.Equal(t, "big", *track.ImageUrl)
	require.Equal(t, 1234, *track.DurationMs)
	require.True(t, *track.Explicit)
	require.Nil(t, track.OwnerName)

	playlist := res.Playlists.Items[0]
	require.Equal(t, "me", *playlist.OwnerName)
	require.Equal(t, "cover", *playlist.ImageUrl)
	require.Nil(t, playlist.ArtistNames)
	require.Nil(t, playlist.DurationMs)
}
