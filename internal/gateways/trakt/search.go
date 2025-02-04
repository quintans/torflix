package trakt

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/https"
	"github.com/quintans/torflix/internal/lib/retry"
)

func (t *Trakt) Search(query string) ([]app.SearchResult, error) {
	uri := fmt.Sprintf("/search/movie,show?fields=title,aliases&query=%s", url.QueryEscape(query))
	var results []app.SearchResult
	err := t.request(http.MethodGet, uri, &results, nil)
	if err != nil {
		return nil, fmt.Errorf("searching for '%s': %w", query, err)
	}
	return results, nil
}

func (t *Trakt) request(method, uri string, request any, response any) error {
	return retry.Do(func() error {
		err := t.client.Request(method, uri, request, response, nil)
		if err != nil {
			return fmt.Errorf("requesting %s: %w", uri, err)
		}

		return nil
	}, retry.WithDelayFunc(https.DelayFunc))
}
