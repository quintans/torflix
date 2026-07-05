package humanize

import (
	"fmt"
	"math"
	"strconv"
)

var sizes = []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}

func Bytes(s uint64, decimalPlaces int) string {
	return humanateBytes(s, 1000, decimalPlaces)
}

func humanateBytes(s uint64, base float64, decimalPlaces int) string {
	if s < 10 {
		return fmt.Sprintf("%d B", s)
	}
	e := math.Floor(logn(float64(s), base))
	suffix := sizes[int(e)]
	val := math.Floor(float64(s)/math.Pow(base, e)*math.Pow(10, float64(decimalPlaces))+0.5) / math.Pow(10, float64(decimalPlaces))
	f := "%." + strconv.Itoa(decimalPlaces) + "f %s"
	if val < 10 {
		f = "%." + strconv.Itoa(decimalPlaces) + "f %s"
	}

	return fmt.Sprintf(f, val, suffix)
}

func logn(n, b float64) float64 {
	return math.Log(n) / math.Log(b)
}
