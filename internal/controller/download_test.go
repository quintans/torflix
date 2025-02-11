package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtract(t *testing.T) {
	result := extractTitle("lioness s02 2160p")
	assert.Equal(t, "lioness", result)

	s, _ := extractSeasonEpisode("lioness 01 2160p")
	assert.Equal(t, 1, s)
	result = extractTitle("lioness 01 2160p")
	assert.Equal(t, "lioness", result)

	s, _ = extractSeasonEpisode("01.lioness")
	assert.Equal(t, 1, s)
	result = extractTitle("01.lioness")
	assert.Equal(t, "lioness", result)

	s, e := extractSeasonEpisode("The.Witcher.Sirens.of.the.Deep.2025.1080p.NF.WEB-DL.DDP5.1.Atmos.H.264-TURG")
	assert.Equal(t, 0, s)
	assert.Equal(t, 0, e)
}
