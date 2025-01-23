package https

import (
	"strconv"
	"time"

	"github.com/quintans/torflix/internal/lib/fails"
)

func DelayFunc(retry int, err error) time.Duration {
	if e, ok := err.(fails.Valuer); ok {
		vals := e.Values()
		if v, ok := vals["retry-after"]; ok {
			if i, ok := v.(string); ok {
				if i, err := strconv.Atoi(i); err == nil {
					return time.Duration(i) * time.Second
				}
			}
		}
	}

	return time.Second
}
