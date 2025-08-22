package model

import "github.com/quintans/faults"

type Search struct {
	query             string
	selectedProviders map[string]bool
	subtitles         bool
}

func NewSearch() *Search {
	return &Search{
		query:             "",
		selectedProviders: map[string]bool{},
		subtitles:         false,
	}
}

func (m *Search) Hydrate(query string, selectedProviders map[string]bool, subtitles bool) {
	m.query = query
	m.selectedProviders = selectedProviders
	m.subtitles = subtitles
}

func (m *Search) SetQuery(query string) error {
	if query == "" {
		return faults.Errorf("query cannot be empty")
	}
	m.query = query

	return nil
}

func (m *Search) Query() string {
	return m.query
}

func (m *Search) SelectedProviders() map[string]bool {
	return m.selectedProviders
}

func (m *Search) SetSelectedProviders(selectedProviders map[string]bool) {
	m.selectedProviders = selectedProviders
}

func (m *Search) Subtitles() bool {
	return m.subtitles
}

func (m *Search) SetSubtitles(subtitles bool) {
	m.subtitles = subtitles
}
