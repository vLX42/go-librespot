package daemon

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/devgianlu/go-librespot/spclient"
)

// searchTimeout bounds one search, Web API round trip and token renewal
// included.
const searchTimeout = 15 * time.Second

const (
	searchDefaultTypes = "track,album,artist,playlist"
	searchDefaultLimit = 5
	// The Web API takes at most 10 results per type, from an offset of at
	// most 1000.
	searchMaxLimit  = 10
	searchMaxOffset = 1000
)

// parseSearchTypes reads the comma separated type parameter of a search. Every
// type must be one the Web API knows; repeats are dropped.
func parseSearchTypes(param string) ([]string, bool) {
	var types []string
	for _, typ := range strings.Split(param, ",") {
		typ = strings.ToLower(strings.TrimSpace(typ))
		if !slices.Contains(spclient.WebSearchTypes, typ) {
			return nil, false
		}
		if !slices.Contains(types, typ) {
			types = append(types, typ)
		}
	}
	return types, len(types) > 0
}

// searchError maps a failed search onto the status the API answers with: a
// rate limited search is a 429, and a query the Web API refuses a 400.
func searchError(err error) error {
	var rateLimited *spclient.RateLimitedError
	if errors.As(err, &rateLimited) {
		return fmt.Errorf("search: %w: %w", ErrTooManyRequests, err)
	}

	var status *spclient.StatusError
	if errors.As(err, &status) && status.StatusCode == 400 {
		return fmt.Errorf("search: %w: %w", ErrBadRequest, err)
	}

	return fmt.Errorf("search: %w", err)
}

// searchResults trims the Web API answer to what picking an item to play
// needs, keeping the types it answered for.
func searchResults(res *spclient.WebSearchResponse) *ApiSearchResults {
	return &ApiSearchResults{
		Tracks:     searchPage(res.Tracks),
		Albums:     searchPage(res.Albums),
		Artists:    searchPage(res.Artists),
		Playlists:  searchPage(res.Playlists),
		Shows:      searchPage(res.Shows),
		Episodes:   searchPage(res.Episodes),
		Audiobooks: searchPage(res.Audiobooks),
	}
}

func searchPage(page *spclient.WebSearchPage) *ApiSearchPage {
	if page == nil {
		return nil
	}

	out := &ApiSearchPage{Total: page.Total, Items: make([]ApiSearchItem, 0, len(page.Items))}
	for _, item := range page.Items {
		// The Web API lists what it cannot show, such as a removed playlist,
		// as null, and there is nothing to play without a uri.
		if item == nil || item.Uri == "" {
			continue
		}
		out.Items = append(out.Items, searchItem(item))
	}
	return out
}

func searchItem(item *spclient.WebSearchItem) ApiSearchItem {
	out := ApiSearchItem{Uri: item.Uri, Type: item.Type, Name: item.Name}

	names := make([]string, 0, len(item.Artists)+len(item.Authors))
	for _, named := range append(item.Artists, item.Authors...) {
		names = append(names, named.Name)
	}
	if len(names) > 0 {
		out.ArtistNames = &names
	}

	images := item.Images
	if album := item.Album; album != nil {
		out.AlbumName = nonEmpty(album.Name)
		out.AlbumUri = nonEmpty(album.Uri)
		if len(images) == 0 {
			images = album.Images
		}
	}
	if owner := item.Owner; owner != nil {
		out.OwnerName = nonEmpty(owner.DisplayName)
	}
	out.Publisher = nonEmpty(item.Publisher)
	out.ReleaseDate = nonEmpty(item.ReleaseDate)
	out.ImageUrl = nonEmpty(largestImage(images))
	out.Explicit = item.Explicit
	if item.DurationMs > 0 {
		out.DurationMs = &item.DurationMs
	}

	return out
}

// largestImage picks the widest image, which the Web API usually lists first
// but does not promise to.
func largestImage(images []spclient.WebImage) string {
	var best *spclient.WebImage
	for i := range images {
		if best == nil || images[i].Width > best.Width {
			best = &images[i]
		}
	}
	if best == nil {
		return ""
	}
	return best.Url
}
