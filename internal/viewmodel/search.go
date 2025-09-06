package viewmodel

import (
	"cmp"
	"fmt"
	"net/http"
	"net/url"
	gslices "slices"
	"sort"
	"strings"
	"time"

	"github.com/quintans/faults"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/bind"
	"github.com/quintans/torflix/internal/lib/magnet"
	"github.com/quintans/torflix/internal/lib/timer"
	"github.com/quintans/torflix/internal/model"
)

type SearchService interface {
	LoadSearch() (*app.SearchSettings, error)
	SaveSearch(model *model.Search) error
	Search(query string, providers []string) ([]*SearchResult, error)
}

type Search struct {
	shared            *Shared
	searchService     SearchService
	downloadService   DownloadService
	OriginalQuery     string
	Providers         []string
	Query             bind.Setter[string]
	MediaName         bind.Setter[string]
	SelectedProviders bind.Setter[map[string]bool]
	DownloadSubtitles bind.Setter[bool]
	SearchResults     bind.Notifier[[]*SearchData]
}

type SearchResult struct {
	Data  []*SearchData
	Error error
}

type SearchData struct {
	Provider    string
	Name        string
	Magnet      string
	Size        string
	Seeds       int
	Quality     int
	QualityName string
	Hash        string
	Cached      bool
}

func NewSearch(shared *Shared, searchService SearchService, downloadService DownloadService) *Search {
	s := &Search{
		shared:          shared,
		searchService:   searchService,
		downloadService: downloadService,
		SearchResults:   bind.NewNotifier[[]*SearchData](),
	}

	s.mount()

	return s
}

func (s *Search) mount() {
	data, err := s.searchService.LoadSearch()
	if err != nil {
		s.shared.Error(err, "Failed to load search data")
		return
	}
	s.Query = bind.New[string](data.Model.Query())
	s.MediaName = bind.New[string](data.Model.MediaName())

	if len(data.Providers) == 0 {
		s.shared.Error(nil, "No providers available for search")
		return
	}

	s.DownloadSubtitles = bind.New[bool](data.Model.Subtitles())
	s.DownloadSubtitles.Listen(func(subtitles bool) {
		data.Model.SetSubtitles(subtitles)
		err := s.searchService.SaveSearch(data.Model)
		if err != nil {
			s.shared.Error(err, "Failed to save search data")
		}
	})

	s.Providers = data.Providers
	// the following code must come after setting the providers
	selected := make(map[string]bool, len(data.Providers))
	oldSelection := data.Model.SelectedProviders()
	for _, v := range data.Providers {
		if oldSelection[v] {
			selected[v] = true
		}
	}
	s.SelectedProviders = bind.NewMap[string, bool](selected)
}

func (s *Search) Unmount() {
	s.Query.UnbindAll()
	s.MediaName.UnbindAll()
	s.SelectedProviders.UnbindAll()
	s.DownloadSubtitles.UnbindAll()
	s.SearchResults.UnbindAll()
}

func IsTorrentResource(link string) bool {
	return strings.HasPrefix(link, "magnet:") ||
		strings.HasPrefix(link, "http:") ||
		strings.HasPrefix(link, "https:") ||
		strings.HasSuffix(link, ".torrent")
}

func (s *Search) SearchAsync() bool {
	query := strings.TrimSpace(s.Query.Get())
	mediaName := strings.TrimSpace(s.MediaName.Get())

	model := model.NewSearch()
	model.SetSubtitles(s.DownloadSubtitles.Get())
	model.SetQuery(query)
	model.SetMediaName(mediaName)

	err := model.SetQuery(query)
	if err != nil {
		s.shared.Error(err, "Failed to set query")
		return false
	}

	selectedProviders := s.SelectedProviders.Get()
	isTorrent := IsTorrentResource(query)

	if !isTorrent && len(selectedProviders) == 0 {
		s.shared.Warn("Please select at least one provider")
		return false
	}

	model.SetSelectedProviders(selectedProviders)

	err = s.searchService.SaveSearch(model)
	if err != nil {
		s.shared.Error(err, "Failed to save search")
		return false
	}

	if isTorrent {
		mn := s.MediaName.Get()
		if mn == "" {
			s.shared.Warn("Media name is required when providing a link")
			return false
		}

		s.OriginalQuery = mn
		return download(s.shared, s.downloadService, s.OriginalQuery, query, s.DownloadSubtitles.Get())
	}

	d := timer.New(time.Second, func() {
		s.shared.Publish(app.Loading{
			Text: "Searching torrents",
			Show: true,
		})
	})

	defer func() {
		d.Stop()
		s.shared.Publish(app.Loading{}) // hide spinner
	}()

	s.OriginalQuery = query

	providers := []string{}
	for k, v := range selectedProviders {
		if v {
			providers = append(providers, k)
		}
	}

	results, err := s.searchService.Search(query, providers)
	if err != nil {
		s.shared.Error(err, "Failed to search")
		return false
	}

	data := []*SearchData{}
	for _, r := range results {
		if r.Error != nil {
			s.shared.Error(r.Error, "Failed to search")
			continue
		}
		data = append(data, r.Data...)
	}
	// results may be >= 0 but the data may be empty (where seeds = 0)
	if len(data) == 0 {
		s.shared.Info("No results found for query")
		s.SearchResults.NotifyAsync(data)

		return false
	}

	data, err = s.collapseByHash(data)
	if err != nil {
		s.shared.Error(err, "Failed to collapse by hash")
		return false
	}

	gslices.SortFunc(data, func(b, a *SearchData) int {
		if a.Quality != b.Quality {
			return cmp.Compare(a.Quality, b.Quality)
		}
		return cmp.Compare(a.Seeds, b.Seeds)
	})

	s.SearchResults.NotifyAsync(data)

	return true
}

func (s *Search) Download(magnetLink string) bool {
	return download(s.shared, s.downloadService, s.OriginalQuery, magnetLink, s.DownloadSubtitles.Get())
}

func (s *Search) collapseByHash(results []*SearchData) ([]*SearchData, error) {
	groups := map[string][]*SearchData{}
	for k, r := range results {
		hash := r.Hash
		if hash == "" {
			hash = fmt.Sprintf("no_hash_%d", k)
		}
		h, ok := groups[hash]
		if !ok {
			groups[hash] = []*SearchData{r}
		} else {
			groups[hash] = append(h, r)
		}
	}

	var merged []*SearchData
	for _, group := range groups {
		if len(group) == 1 {
			merged = append(merged, group[0])
		} else {
			magnets := make([]string, 0, len(group))
			for _, r := range group {
				if r.Magnet == "" {
					s.shared.Warn("Empty magnet link. name=%s, provider=%s", r.Name, r.Provider)
					continue
				}
				magnets = append(magnets, r.Magnet)
			}

			magnet, dn, err := mergeMagnetLinks(magnets)
			if err != nil {
				return nil, faults.Errorf("merging magnet links: %w", err)
			}

			providers := make([]string, 0, len(group))
			for _, r := range group {
				providers = append(providers, r.Provider)
			}
			sort.Strings(providers)

			maxSeeded := gslices.MaxFunc(group, func(a, b *SearchData) int {
				return cmp.Compare(a.Seeds, b.Seeds)
			})

			merged = append(merged, &SearchData{
				Provider:    strings.Join(providers, ","),
				Name:        dn,
				Magnet:      magnet,
				Size:        maxSeeded.Size,
				Seeds:       maxSeeded.Seeds,
				Quality:     maxSeeded.Quality,
				QualityName: maxSeeded.QualityName,
				Hash:        maxSeeded.Hash,
			})
		}
	}

	return merged, nil
}

func mergeMagnetLinks(links []string) (string, string, error) {
	if len(links) == 0 {
		return "", "", faults.Errorf("no magnet links provided")
	}

	// Maps to store unique values for each component
	var hash string
	var dns []string
	trackers := make(map[string]struct{})
	webSeeds := make(map[string]struct{})

	for _, link := range links {
		u, err := url.Parse(link)
		if err != nil {
			return "", "", faults.Errorf("failed to parse link (%s): %w", link, err)
		}

		if strings.HasPrefix(u.Scheme, "http") {
			// stop redirect
			checkRedirect := http.DefaultClient.CheckRedirect
			http.DefaultClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}
			defer func() {
				http.DefaultClient.CheckRedirect = checkRedirect
			}()

			res, err := http.Get(link)
			if err != nil {
				return "", "", faults.Errorf("failed to fetch link (%s): %w", link, err)
			}
			defer res.Body.Close()
			if res.StatusCode != http.StatusTemporaryRedirect &&
				res.StatusCode != http.StatusPermanentRedirect {
				return "", "", faults.Errorf("don't know how to handle non-magnet link: %s; Status Code: %d", link, res.StatusCode)
			}

			link = res.Header.Get("Location")
		}

		mag, err := magnet.Parse(link)
		if err != nil {
			return "", "", faults.Errorf("failed to parse magnet link (%s): %w", link, err)
		}

		dns = append(dns, mag.DisplayName)

		if hash == "" {
			hash = mag.Hash
		} else if hash != mag.Hash {
			return "", "", faults.Errorf("different hashes found when merging: %s and %s", hash, mag.Hash)
		}

		for _, value := range mag.Trackers {
			trackers[value] = struct{}{}
		}

		for _, value := range mag.WebSeeds {
			webSeeds[value] = struct{}{}
		}
	}

	if hash == "" {
		return "", "", faults.Errorf("no hash (xt) found in magnet links")
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
