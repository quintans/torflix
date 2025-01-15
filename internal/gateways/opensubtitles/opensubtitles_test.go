package opensubtitles

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSearch(t *testing.T) {
	osd := New("", "")

	subs, err := osd.Search("lioness", 2, 1, []string{"en", "pt"})
	require.NoError(t, err)
	require.NotEmpty(t, subs)
}
