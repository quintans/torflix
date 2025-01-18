package extractor_test

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/quintans/torflix/internal/lib/extractor"
	"github.com/stretchr/testify/require"
)

func TestApiExtractor(t *testing.T) {
	tests := []struct {
		name    string
		results []extractor.Result
	}{
		{
			name: "tpb",
			results: []extractor.Result{
				{
					Name:   "SAS Rogue Heroes S02E01 1080p HEVC x265-MeGusta",
					Magnet: "magnet:?xt=urn:btih:991B63685C6BB91E2A199D8495ECE6AA605A161C&dn=SAS%20Rogue%20Heroes%20S02E01%201080p%20HEVC%20x265-MeGusta",
					Size:   "364 MB",
					Seeds:  "389",
				},
				{
					Name:   "SAS Rogue Heroes S02E02 1080p HEVC x265-MeGusta",
					Magnet: "magnet:?xt=urn:btih:2864855F08CF34BD80FC2439A44BC31D8F3CD788&dn=SAS%20Rogue%20Heroes%20S02E02%201080p%20HEVC%20x265-MeGusta",
					Size:   "490 MB",
					Seeds:  "352",
				},
			},
		},
	}

	scraper, err := extractor.NewApi(apiCfg)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &http.Server{
				Addr: ":1234",
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err = w.Write(tpb)
					require.NoError(t, err)
				}),
			}

			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					slog.Error("ListenAndServe()", "error", err)
				}
			}()

			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := server.Shutdown(ctx); err != nil {
					slog.Error("Server Shutdown Failed.", "error", err)
				}
			}()

			results, err := scraper.Extract(tt.name, "something with spaces")
			require.NoError(t, err)

			for i := range results {
				results[i].Name = removeExtraSpaces(results[i].Name)
			}

			require.Equal(t, tt.results, results)
		})
	}
}

var apiCfg = []byte(`{
	"tpb": {
		"url": "http://localhost:1234/q.php?q={{.query}}&cat=",
		"result": {
			"name": "name",
			"hash": "info_hash",
			"ssize": "size",
			"seeds": "seeders"
		}
	}
}`)

var tpb = []byte(`[
    {
        "id": "77721055",
        "name": "SAS Rogue Heroes S02E01 1080p HEVC x265-MeGusta",
        "info_hash": "991B63685C6BB91E2A199D8495ECE6AA605A161C",
        "leechers": "193",
        "seeders": "389",
        "num_files": "0",
        "size": "363836994",
        "username": "jajaja",
        "added": "1735743901",
        "status": "vip",
        "category": "208",
        "imdb": "tt10405370"
    },
    {
        "id": "77721255",
        "name": "SAS Rogue Heroes S02E02 1080p HEVC x265-MeGusta",
        "info_hash": "2864855F08CF34BD80FC2439A44BC31D8F3CD788",
        "leechers": "81",
        "seeders": "352",
        "num_files": "0",
        "size": "490349168",
        "username": "jajaja",
        "added": "1735745701",
        "status": "vip",
        "category": "208",
        "imdb": "tt10405370"
    }
]`)
