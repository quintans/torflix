package controller

import (
	"cmp"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/components"
	"github.com/quintans/torflix/internal/lib/extractor"
	"github.com/quintans/torflix/internal/lib/magnet"
	"github.com/quintans/torflix/internal/lib/safe"
	"github.com/quintans/torflix/internal/model"
)

type SearchView interface {
	Show(*model.Search, []string)
}

type Search struct {
	view           SearchView
	nav            app.Navigator
	repo           Repository
	searchScraper  app.Scraper
	detailsScraper app.Scraper
	eventBus       app.EventBus
	providers      []string
}

func NewSearch(
	view SearchView, nav app.Navigator, repo Repository,
	searchScraper app.Scraper, detailsScraper app.Scraper,
	providers []string,
	eventBus app.EventBus,
) (Search, error) {
	model, err := repo.LoadSearch()
	if err != nil {
		return Search{}, fmt.Errorf("loading search: %w", err)
	}
	selectedProviders := model.SelectedProviders()
	changed := false
	for k := range selectedProviders {
		if !slices.Contains(providers, k) {
			delete(selectedProviders, k)
			changed = true
		}
	}
	if changed {
		model.SetSelectedProviders(selectedProviders)
		repo.SaveSearch(model)
	}

	return Search{
		view:           view,
		nav:            nav,
		repo:           repo,
		searchScraper:  searchScraper,
		detailsScraper: detailsScraper,
		eventBus:       eventBus,
		providers:      providers,
	}, nil
}

func (c Search) OnEnter() {
	providers := c.providers

	model, err := c.repo.LoadSearch()
	if err != nil {
		c.eventBus.Publish(app.NewNotifyError("Failed to load search: %s", err))
		slog.Error("Failed to load search", "error", err)
		return
	}

	c.view.Show(model, providers)
}

type Result struct {
	Provider string
	Name     string
	Magnet   string
	Size     string
	Seeds    int
	Quality  int
	Follow   string
	Hash     string
}

func (c Search) SearchModel() (*model.Search, error) {
	return c.repo.LoadSearch()
}

func (c Search) Search(query string, selectedProviders []string) ([]components.MagnetItem, error) {
	query = strings.TrimSpace(query)
	model, err := c.repo.LoadSearch()
	if err != nil {
		return nil, fmt.Errorf("loading search: %w", err)
	}

	err = model.SetQuery(query)
	if err != nil {
		return nil, fmt.Errorf("setting query: %w", err)
	}

	if strings.HasPrefix(query, "magnet:") {
		mag, err := magnet.Parse(query)
		if err != nil {
			return nil, fmt.Errorf("parsing magnet: %w", err)
		}
		dn := mag.DisplayName
		c.Download(cleanTorrentName(dn), query)
		return nil, nil
	}

	selection := make(map[string]bool, len(c.providers))
	for _, p := range c.providers {
		selection[p] = slices.Contains(selectedProviders, p)
	}
	model.SetSelectedProviders(selection)
	c.repo.SaveSearch(model)

	settings, err := c.repo.LoadSettings()
	if err != nil {
		return nil, fmt.Errorf("loading settings: %w", err)
	}
	qualities := settings.Qualities()
	safeRes := safe.New([]Result{})
	wg := sync.WaitGroup{}
	for _, slug := range selectedProviders {
		wg.Add(1)
		go func(slug string) {
			defer wg.Done()

			res, err := c.searchScraper.ScrapeQuery(slug, query)
			if err != nil {
				c.eventBus.Publish(app.NewNotifyError("Failed scraping: %s", err))
				slog.Error("Failed scraping", "error", err)
				return
			}

			// retrieve magnet if follow link is set
			for k, r := range res {
				if r.Follow != "" {
					magnet, err := c.follow(slug, r.Follow)
					if err != nil {
						c.eventBus.Publish(app.NewNotifyWarn("Failed following: %s", err))
						slog.Error("Failed following", "error", err)
						continue
					}
					res[k].Magnet = magnet
				}
			}

			r, err := c.transformToMyResult(slug, res, qualities)
			if err != nil {
				c.eventBus.Publish(app.NewNotifyError("Failed converting seeds (slug=%s): %s", slug, err))
				slog.Error("Failed converting seeds.", "slug", slug, "error", err)
				return
			}
			safeRes.Update(func(v []Result) []Result {
				return append(v, r...)
			})
		}(slug)
	}
	wg.Wait()

	results := safeRes.Get()
	if len(results) == 0 {
		c.eventBus.Publish(app.NewNotifyInfo("No results found for query"))
		return nil, nil
	}
	results = slices.DeleteFunc(results, func(r Result) bool {
		return r.Seeds == 0
	})

	results, err = c.collapseByHash(results)
	if err != nil {
		return nil, fmt.Errorf("collapsing by hash: %w", err)
	}

	slices.SortFunc(results, func(b, a Result) int {
		if a.Quality != b.Quality {
			return cmp.Compare(a.Quality, b.Quality)
		}
		return cmp.Compare(a.Seeds, b.Seeds)
	})

	items := make([]components.MagnetItem, len(results))
	for i, r := range results {
		items[i] = components.MagnetItem{
			Provider: r.Provider,
			Name:     r.Name,
			Size:     r.Size,
			Seeds:    strconv.Itoa(r.Seeds),
			Magnet:   r.Magnet,
			Follow:   r.Follow,
		}
		if r.Quality != 0 {
			items[i].Quality = qualities[r.Quality-1]
		}
	}

	return items, nil
}

func (c Search) follow(provider, link string) (string, error) {
	if link == "" {
		return "", fmt.Errorf("follow link not set")
	}

	results, err := c.detailsScraper.ScrapeLink(provider, link)
	if err != nil {
		return "", fmt.Errorf("failed to scrape follow link (%s): %w", link, err)
	}

	if len(results) == 0 {
		return "", fmt.Errorf("no results found for follow link")
	}

	if results[0].Magnet == "" {
		return "", fmt.Errorf("no magnet link found for follow link")
	}

	return results[0].Magnet, nil
}

var reHash = regexp.MustCompile(`urn:btih:([a-fA-F0-9]+)`)

func (c Search) transformToMyResult(slug string, r []extractor.Result, qualities []string) ([]Result, error) {
	var results []Result
	for _, r := range r {
		rep := strings.NewReplacer(",", "", ".", "")
		seeds, err := strconv.Atoi(rep.Replace(r.Seeds))
		if err != nil {
			return nil, fmt.Errorf("converting seeds '%s' for '%s': %s", r.Seeds, r.Name, err)
		}

		var hash string
		match := reHash.FindStringSubmatch(r.Magnet)
		if len(match) > 1 {
			hash = match[1]
		}

		result := Result{
			Provider: slug,
			Name:     r.Name,
			Magnet:   r.Magnet,
			Size:     r.Size,
			Seeds:    seeds,
			Follow:   r.Follow,
			Hash:     hash,
		}

		for i, q := range qualities {
			name := strings.ToLower(r.Name)
			if strings.Contains(name, q) {
				result.Quality = i + 1
				break
			}
		}

		results = append(results, result)
	}
	return results, nil
}

func (c Search) collapseByHash(results []Result) ([]Result, error) {
	groups := map[string][]Result{}
	for _, r := range results {
		h, ok := groups[r.Hash]
		if !ok {
			groups[r.Hash] = []Result{r}
		} else {
			groups[r.Hash] = append(h, r)
		}
	}

	var merged []Result
	for _, group := range groups {
		if len(group) == 1 {
			merged = append(merged, group[0])
		} else {
			magnets := make([]string, 0, len(group))
			for _, r := range group {
				if r.Magnet == "" {
					c.eventBus.Publish(app.NewNotifyWarn("Empty magnet link. name=%s, provider=%s, follow=%s", r.Name, r.Provider, r.Follow))
					continue
				}
				magnets = append(magnets, r.Magnet)
			}

			magnet, dn, err := mergeMagnetLinks(magnets)
			if err != nil {
				return nil, fmt.Errorf("merging magnet links: %w", err)
			}

			providers := make([]string, 0, len(group))
			for _, r := range group {
				providers = append(providers, r.Provider)
			}
			sort.Strings(providers)

			maxSeeded := slices.MaxFunc(group, func(a, b Result) int {
				return cmp.Compare(a.Seeds, b.Seeds)
			})

			merged = append(merged, Result{
				Provider: strings.Join(providers, ","),
				Name:     dn,
				Magnet:   magnet,
				Size:     maxSeeded.Size,
				Seeds:    maxSeeded.Seeds,
				Quality:  maxSeeded.Quality,
				Hash:     maxSeeded.Hash,
			})
		}
	}

	return merged, nil
}

func (c Search) Download(originalQuery string, magnetLink string) error {
	model := c.repo.LoadDownload()
	err := model.SetQueryAndLink(originalQuery, magnetLink)
	if err != nil {
		return fmt.Errorf("setting link: %w", err)
	}
	c.repo.SaveDownload(model)

	c.nav.Go(DownloadNavigation)

	return nil
}

func mergeMagnetLinks(links []string) (string, string, error) {
	if len(links) == 0 {
		return "", "", fmt.Errorf("no magnet links provided")
	}

	// Maps to store unique values for each component
	var hash string
	var dns []string
	trackers := make(map[string]struct{})
	webSeeds := make(map[string]struct{})

	for _, link := range links {
		mag, err := magnet.Parse(link)
		if err != nil {
			return "", "", fmt.Errorf("failed to parse magnet link (%s): %w", link, err)
		}

		u, err := url.Parse(link)
		if err != nil {
			return "", "", fmt.Errorf("failed to parse magnet link (%s): %w", link, err)
		}
		if u.Scheme != "magnet" {
			return "", "", fmt.Errorf("invalid scheme for magnet (%s): %s", link, u.Scheme)
		}

		dns = append(dns, mag.DisplayName)

		if hash == "" {
			hash = mag.Hash
		} else if hash != mag.Hash {
			return "", "", fmt.Errorf("different hashes found when merging: %s and %s", hash, mag.Hash)
		}

		for _, value := range mag.Trackers {
			trackers[value] = struct{}{}
		}

		for _, value := range mag.WebSeeds {
			webSeeds[value] = struct{}{}
		}
	}

	if hash == "" {
		return "", "", fmt.Errorf("no hash (xt) found in magnet links")
	}

	// Determine the smallest `dn`
	var smallestDN string
	if len(dns) > 0 {
		sort.Strings(dns)
		smallestDN = dns[0]
	}

	// Build the merged magnet link
	mergedParams := url.Values{}
	mergedParams.Add("xt", hash)
	if smallestDN != "" {
		mergedParams.Add("dn", smallestDN)
	}
	for tracker := range trackers {
		mergedParams.Add("tr", tracker)
	}
	for webSeed := range webSeeds {
		mergedParams.Add("ws", webSeed)
	}

	return fmt.Sprintf("magnet:?%s", mergedParams.Encode()), smallestDN, nil
}

func cleanTorrentName(torrentName string) string {
	// Pattern to identify and retain special markers (Season/Episode info)
	importantPattern := regexp.MustCompile(`(?i)\b(S\d{2}E\d{2}|Season \d+)\b`)

	// Find the first occurrence of the pattern
	loc := importantPattern.FindStringIndex(torrentName)

	if loc != nil {
		// Keep everything up to and including the matched pattern
		torrentName = torrentName[:loc[1]]
	}

	// Remove common metadata from the trimmed name
	patterns := []string{
		`(?i)\b(720p|1080p|2160p|4k|8k)\b`,           // Resolutions
		`(?i)\b(x264|x265|h264|h265)\b`,              // Codecs
		`(?i)\b(WEBRip|BRRip|BluRay|HDTV|WEB-DL)\b`,  // Sources
		`(?i)\b(DTS|DD5\.1|AAC|Atmos|TrueHD|MP3)\b`,  // Audio formats
		`$begin:math:display$\\w+$end:math:display$`, // Text in square brackets
		`$begin:math:text$[^)]+$end:math:text$`,      // Text in parentheses
		`-.*$`,                                       // Trailing release group name
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		torrentName = re.ReplaceAllString(torrentName, "")
	}

	// Replace dots and underscores with spaces
	torrentName = strings.ReplaceAll(torrentName, ".", " ")
	torrentName = strings.ReplaceAll(torrentName, "_", " ")

	// Trim extra spaces
	torrentName = strings.TrimSpace(torrentName)

	return torrentName
}
