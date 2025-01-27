package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtract(t *testing.T) {
	// result, _, _ := extractSeasonEpisode("lioness s02 2160p", true)
	// assert.Equal(t, "lioness", result)

	// result, s, _ := extractSeasonEpisode("lioness 01 2160p", true)
	// assert.Equal(t, "lioness", result)
	// assert.Equal(t, 1, s)

	result, s, _ := extractSeasonEpisode("01.lioness", true)
	assert.Equal(t, "lioness", result)
	assert.Equal(t, 1, s)
}
