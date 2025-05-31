package scraper

// Copied from github.com/jpillora/scraper

import (
	"fmt"
	"io"

	"github.com/PuerkitoBio/goquery"
)

type Result map[string]string

type Endpoint struct {
	List   string                `json:"list,omitempty"`
	Result map[string]Extractors `json:"result"`
	Debug  bool
}

// Execute will execute an Endpoint with the given params
func (e *Endpoint) Execute(body io.Reader) ([]Result, error) {
	//parse HTML
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, err
	}
	sel := doc.Selection
	//results will be either a list of results, or a single result
	var results []Result
	if e.List != "" {
		sels := sel.Find(e.List)
		if e.Debug {
			logf("list: %s => #%d elements", e.List, sels.Length())
		}
		if e.Debug && sels.Length() == 0 {
			logf("no results, printing HTML")
			h, _ := sel.Html()
			fmt.Println(h)
		}
		sels.Each(func(i int, sel *goquery.Selection) {
			r := e.extract(sel)
			if len(r) == len(e.Result) {
				results = append(results, r)
			} else if e.Debug {
				logf("excluded #%d: has %d fields, expected %d", i, len(r), len(e.Result))
			}
		})
	} else {
		results = append(results, e.extract(sel))
	}
	return results, nil
}

// extract 1 result using this endpoints extractor map
func (e *Endpoint) extract(sel *goquery.Selection) Result {
	r := Result{}
	for field, ext := range e.Result {
		if v := ext.execute(sel); v != "" {
			r[field] = v
		} else if e.Debug {
			logf("missing %s", field)
		}
	}
	return r
}
