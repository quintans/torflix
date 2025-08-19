package viewmodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanTorrentName(t *testing.T) {
	// test cases
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "tv show s00e00",
			input:    "The.Office.S01E01.720p.BluRay.x264-GROUP",
			expected: "The Office S01E01",
		},
		{
			name:     "movie with year",
			input:    "Inception.2010.1080p.BluRay.x264-YTS",
			expected: "Inception 2010",
		},
		{
			name:     "with s00e00 and year",
			input:    "The.Big.Bang.Theory.S01E01.2007.HDTV.XviD-XOR.avi",
			expected: "The Big Bang Theory S01E01",
		},
		{
			name:     "with dots and year and season",
			input:    "Breaking.Bad.S05E14.Ozymandias.1080p.WEB-DL.DD5.1.H264-RARBG",
			expected: "Breaking Bad S05E14",
		},
		{
			name:     "with season and episode",
			input:    "Stranger.Things.Season.02.Episode.05.1080p.WEB-DL.x264-XYZ",
			expected: "Stranger Things Season 02 Episode 05",
		},
		{
			name:     "with year and season and something else",
			input:    "The.Big.Bang.Theory.S01.COMPLETE",
			expected: "The Big Bang Theory S01",
		},
	}

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanTorrentName(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
