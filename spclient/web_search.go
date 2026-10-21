package spclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// webApiBaseUrl is the public Web API, which takes the session's own token.
var webApiBaseUrl = url.URL{Scheme: "https", Host: "api.spotify.com", Path: "/v1/"}

// WebSearchTypes are the item types the Web API search knows.
var WebSearchTypes = []string{"track", "album", "artist", "playlist", "show", "episode", "audiobook"}

// WebImage is an image as the Web API lists it.
type WebImage struct {
	Url    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// WebNamed is the name and URI of a referenced item, like a track's artist.
type WebNamed struct {
	Uri  string `json:"uri"`
	Name string `json:"name"`
}

// WebSearchItem is one search result of any type. The Web API gives each type
// fields of its own; only the ones a result list needs are read here.
type WebSearchItem struct {
	Uri         string     `json:"uri"`
	Type        string     `json:"type"`
	Name        string     `json:"name"`
	Artists     []WebNamed `json:"artists"`
	Authors     []WebNamed `json:"authors"`
	Album       *WebAlbum  `json:"album"`
	Owner       *WebOwner  `json:"owner"`
	Publisher   string     `json:"publisher"`
	DurationMs  int        `json:"duration_ms"`
	ReleaseDate string     `json:"release_date"`
	Explicit    *bool      `json:"explicit"`
	Images      []WebImage `json:"images"`
}

// WebAlbum is the album a track belongs to.
type WebAlbum struct {
	Uri    string     `json:"uri"`
	Name   string     `json:"name"`
	Images []WebImage `json:"images"`
}

// WebOwner is the owner of a playlist.
type WebOwner struct {
	DisplayName string `json:"display_name"`
}

// WebSearchPage is the results of one type. The Web API lists an item it
// cannot show, such as a removed playlist, as null.
type WebSearchPage struct {
	Total int              `json:"total"`
	Items []*WebSearchItem `json:"items"`
}

// WebSearchResponse is the answer of the Web API search, a page per type asked
// for.
type WebSearchResponse struct {
	Tracks     *WebSearchPage `json:"tracks"`
	Albums     *WebSearchPage `json:"albums"`
	Artists    *WebSearchPage `json:"artists"`
	Playlists  *WebSearchPage `json:"playlists"`
	Shows      *WebSearchPage `json:"shows"`
	Episodes   *WebSearchPage `json:"episodes"`
	Audiobooks *WebSearchPage `json:"audiobooks"`
}

// WebSearch searches the catalog through the public Web API's /v1/search,
// with the session's token, so the market is the user's own. It is sent once:
// a 429 comes back as a RateLimitedError, since repeating a rate limited
// search only makes it worse, and any other failure as a StatusError.
func (c *Spclient) WebSearch(ctx context.Context, query string, types []string, limit, offset int) (*WebSearchResponse, error) {
	reqUrl := webApiBaseUrl.JoinPath("search")
	params := url.Values{
		"q":      []string{query},
		"type":   []string{strings.Join(types, ",")},
		"limit":  []string{strconv.Itoa(limit)},
		"offset": []string{strconv.Itoa(offset)},
	}
	header := http.Header{"Accept": []string{"application/json"}}

	resp, err := c.innerRequestWith(ctx, c.client, http.MethodGet, reqUrl, params, header, nil, false)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusTooManyRequests:
		return nil, &RateLimitedError{
			RetryAfter: parseRetryAfter(resp.Header),
			err:        fmt.Errorf("web api search rate limited"),
		}
	default:
		return nil, &StatusError{Op: "web api search", StatusCode: resp.StatusCode}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading search response: %w", err)
	}

	var res WebSearchResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("failed unmarshalling search response: %w", err)
	}

	return &res, nil
}
