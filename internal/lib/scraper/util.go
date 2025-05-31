package scraper

import (
	"bytes"
	"fmt"
	"log"

	"github.com/PuerkitoBio/goquery"
)

func checkSelector(s string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	doc, _ := goquery.NewDocumentFromReader(bytes.NewBufferString(`<html>
		<body>
			<h3>foo bar</h3>
		</body>
	</html>`))
	doc.Find(s)
	return
}

func logf(format string, args ...interface{}) {
	log.Printf("[scraper] "+format, args...)
}
