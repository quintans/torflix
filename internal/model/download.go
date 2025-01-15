package model

import "fmt"

type Download struct {
	link          string
	originalQuery string
}

func NewDownload() *Download {
	return &Download{}
}

func (m *Download) SetQueryAndLink(originalQuery, link string) error {
	if link == "" {
		return fmt.Errorf("link cannot be empty")
	}
	// TODO validate link. Logic is in client.
	m.link = link

	if originalQuery == "" {
		return fmt.Errorf("originalQuery cannot be empty")
	}
	m.originalQuery = originalQuery

	return nil
}

func (m *Download) Link() string {
	return m.link
}

func (m *Download) OriginalQuery() string {
	return m.originalQuery
}
