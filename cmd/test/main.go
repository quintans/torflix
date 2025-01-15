package main

import (
	"fmt"
	"regexp"
	"strings"
)

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

func main() {
	torrentNames := []string{
		"The.Office.S01E01.720p.BluRay.x264-GROUP",
		"Inception.2010.1080p.BluRay.x264-YTS",
		"Breaking.Bad.S05E14.Ozymandias.1080p.WEB-DL.DD5.1.H264-RARBG",
		"Game.of.Thrones.S08E03.1080p.BluRay.x265-MEGROUP",
		"Stranger.Things.Season.02.Episode.05.1080p.WEB-DL.x264-XYZ",
		"SAS Rogue Heroes S02E01 1080p HEVC x265-MeGusta",
	}

	for _, name := range torrentNames {
		fmt.Printf("Original: %s\nCleaned:  %s\n\n", name, cleanTorrentName(name))
	}
}
