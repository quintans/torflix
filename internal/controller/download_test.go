package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtract(t *testing.T) {
	result, _, _ := extractSeasonEpisode("lioness s02 2160p", true)

	assert.Equal(t, "lioness", result)
}
