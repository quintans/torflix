package model

import "fmt"

type Search struct {
	query             string
	selectedProviders map[string]bool
}

func NewSearch() *Search {
	return &Search{
		query: "",
		selectedProviders: map[string]bool{
			"1337x": true,
			"bt4g":  false,
			"tgx":   false,
			"nyaa":  false,
		},
	}
}

func (m *Search) SetQuery(query string) error {
	if query == "" {
		return fmt.Errorf("query cannot be empty")
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

func (m *Search) Hydrate(query string, selectedProviders map[string]bool) {
	m.query = query
	m.selectedProviders = selectedProviders
}
