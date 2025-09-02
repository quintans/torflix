package text

import (
	"fmt"
)

func Fmt(format string, args ...any) string {
	if len(args) > 0 {
		// any(format).(string) is telling the vet: “this is a dynamic format string — don’t check it.”
		return fmt.Sprintf(any(format).(string), args...)
	}
	return format
}
