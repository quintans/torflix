package extractor

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/jpillora/scraper/scraper"
)

type Result struct {
	Name   string
	Magnet string
	Size   string
	Seeds  string
}

type HtmlResult struct {
	Name   string `json:"name"`
	Magnet string `json:"magnet"`
	Size   string `json:"size"`
	Seeds  string `json:"seeds"`
	Follow string `json:"follow"`
}

type HtmlEndpoint struct {
	QueryInPath bool   `json:"queryInPath"`
	Url         string `json:"url"`
}

type Scraper struct {
	QueryHandler  *scraper.Handler
	FollowHandler *scraper.Handler
	queryScrapers map[string]HtmlEndpoint
}

func NewScraper(searchCfg, followCfg []byte) (*Scraper, error) {
	cfg := slices.Clone(searchCfg)

	scrapers := map[string]HtmlEndpoint{}
	err := json.Unmarshal(cfg, &scrapers)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal search config: %w", err)
	}

	queryHandler, err := newHandler(searchCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create query handler: %w", err)
	}

	followHandler, err := newHandler(followCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create follow handler: %w", err)
	}

	return &Scraper{
		QueryHandler:  queryHandler,
		FollowHandler: followHandler,
		queryScrapers: scrapers,
	}, nil
}

func newHandler(scrapeCfg []byte) (*scraper.Handler, error) {
	cfg := slices.Clone(scrapeCfg)

	search := &scraper.Handler{
		Headers: map[string]string{
			"User-Agent":      "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/115.0",
			"Accept":          "*/*",
			"Accept-Encoding": "deflate",
			"Connection":      "keep-alive",
		},
		Log:   true,
		Debug: true,
	}
	err := search.LoadConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load search config: %w", err)
	}
	for _, v := range search.Config {
		replacer := strings.NewReplacer("{{query}}", "", "{{link}}", "")
		newUrl := replacer.Replace(v.URL)
		u, err := url.Parse(newUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to parse url: %w", err)
		}
		search.Headers["Host"] = u.Host
	}

	return search, nil
}

func (s *Scraper) Accept(slug string) bool {
	_, ok := s.queryScrapers[slug]
	return ok
}

func (s *Scraper) Slugs() []string {
	slugs := make([]string, 0, len(s.queryScrapers))
	for k := range s.queryScrapers {
		slugs = append(slugs, k)
	}
	return slugs
}

func (s *Scraper) Extract(slug string, query string) ([]Result, error) {
	cfg, ok := s.queryScrapers[slug]
	if !ok {
		return nil, fmt.Errorf("no scraper found for %s", slug)
	}

	if cfg.QueryInPath {
		query = url.PathEscape(query)
	} else {
		query = url.QueryEscape(query)
	}

	htmlRes, err := scrape(s.QueryHandler, slug, map[string]string{
		"query": query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scrape query: %w", err)
	}

	// retrieve magnet if follow link is set
	for k, r := range htmlRes {
		if r.Follow != "" {
			magnet, err := s.follow(slug, r.Follow)
			if err != nil {
				slog.Warn("Failed to follow link", "slug", slug, "link", r.Follow, "error", err)
				continue
			}
			htmlRes[k].Magnet = magnet
		}
	}

	res := make([]Result, 0, len(htmlRes))
	for _, r := range htmlRes {
		if r.Magnet == "" {
			continue
		}

		res = append(res, Result{
			Name:   r.Name,
			Magnet: r.Magnet,
			Size:   r.Size,
			Seeds:  r.Seeds,
		})
	}

	return res, nil
}

func (s *Scraper) follow(provider, link string) (string, error) {
	if link == "" {
		return "", fmt.Errorf("follow link not set")
	}

	results, err := s.scrapeLink(provider, link)
	if err != nil {
		return "", fmt.Errorf("failed to scrape follow link: %w", err)
	}

	if len(results) == 0 {
		return "", fmt.Errorf("no results found for follow link")
	}

	if results[0].Magnet == "" {
		return "", fmt.Errorf("no magnet link found for follow link")
	}

	return results[0].Magnet, nil
}

func (s *Scraper) scrapeLink(slug string, link string) ([]HtmlResult, error) {
	return scrape(s.FollowHandler, slug, map[string]string{
		"link": link,
	})
}

func scrape(handler *scraper.Handler, slug string, values map[string]string) ([]HtmlResult, error) {
	endpoint := handler.Endpoint(slug)
	if endpoint == nil {
		return nil, fmt.Errorf("endpoint not found: %s", slug)
	}

	fmt.Println("===> Scraping", endpoint)

	http.DefaultClient.Timeout = 10 * time.Second
	res, err := endpoint.Execute(values)
	if err != nil {
		return nil, fmt.Errorf("failed to execute endpoint: %w", err)
	}

	fmt.Println("===> Found", len(res), "results:", res)

	// encode as JSON
	b, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results: %w", err)
	}

	results := make([]HtmlResult, 0, len(res))
	err = json.Unmarshal(b, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal results: %w", err)
	}

	return results, nil
}
