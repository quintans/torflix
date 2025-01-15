package magnet

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/quintans/torflix/internal/lib/maps"
)

type Magnet struct {
	Hash        string
	DisplayName string
	Trackers    []string
	WebSeeds    []string
}

func Parse(link string) (Magnet, error) {
	u, err := url.Parse(link)
	if err != nil {
		return Magnet{}, fmt.Errorf("failed to parse magnet link: %w", err)
	}
	if u.Scheme != "magnet" {
		return Magnet{}, fmt.Errorf("invalid scheme for magnet: %s", u.Scheme)
	}

	// Maps to store unique values for each component
	var hash string
	var dn string
	trackers := make(map[string]struct{})
	webSeeds := make(map[string]struct{})

	// Parse query parameters
	queryParams := u.Query()
	for key, values := range queryParams {
		switch key {
		case "xt":
			// Extract hash
			for _, value := range values {
				if strings.HasPrefix(value, "urn:btih:") {
					if hash == "" {
						hash = value
					} else if hash != value {
						return Magnet{}, fmt.Errorf("different hashes found: %s and %s", hash, value)
					}
				}
			}
		case "dn":
			dn = values[0]
		case "tr":
			// Collect unique trackers
			for _, value := range values {
				trackers[value] = struct{}{}
			}
		case "ws":
			// Collect unique web seeds
			for _, value := range values {
				webSeeds[value] = struct{}{}
			}
		}
	}

	if hash == "" {
		return Magnet{}, fmt.Errorf("no hash (xt) found in magnet link")
	}

	return Magnet{
		Hash:        hash,
		DisplayName: dn,
		Trackers:    maps.Keys(trackers),
		WebSeeds:    maps.Keys(webSeeds),
	}, nil
}
