package extractor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/jpillora/scraper/scraper"
)

type Result struct {
	Name   string `json:"name"`
	Magnet string `json:"magnet"`
	Size   string `json:"size"`
	Seeds  string `json:"seeds"`
	Follow string `json:"follow"`
}

type Endpoint struct {
	QueryInPath bool `json:"queryInPath"`
}

type Scraper struct {
	Handler  *scraper.Handler
	scrapers map[string]Endpoint
}

func NewScraper(searchCfg []byte) (*Scraper, error) {
	cfg := slices.Clone(searchCfg)

	scrapers := map[string]Endpoint{}
	err := json.Unmarshal(cfg, &scrapers)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal search config: %w", err)
	}

	search := &scraper.Handler{Log: false}
	err = search.LoadConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load search config: %w", err)
	}

	return &Scraper{
		Handler:  search,
		scrapers: scrapers,
	}, nil
}

func (s *Scraper) ScrapeQuery(slug string, query string) ([]Result, error) {
	cfg := s.scrapers[slug]
	if cfg.QueryInPath {
		query = url.PathEscape(query)
	} else {
		query = url.QueryEscape(query)
	}

	return s.scrape(slug, map[string]string{
		"query": query,
	})
}

func (s *Scraper) ScrapeLink(slug string, link string) ([]Result, error) {
	return s.scrape(slug, map[string]string{
		"link": link,
	})
}

func (s *Scraper) scrape(slug string, values map[string]string) ([]Result, error) {
	endpoint := s.Handler.Endpoint(slug)
	if endpoint == nil {
		return nil, fmt.Errorf("endpoint not found: %s", slug)
	}

	http.DefaultClient.Timeout = 10 * time.Second
	res, err := endpoint.Execute(values)
	if err != nil {
		return nil, fmt.Errorf("failed to execute endpoint: %w", err)
	}

	// encode as JSON
	b, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results: %w", err)
	}

	results := make([]Result, 0, len(res))
	err = json.Unmarshal(b, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal results: %w", err)
	}

	return results, nil
}
