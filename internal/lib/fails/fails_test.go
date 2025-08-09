package fails_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/quintans/faults"
	"github.com/quintans/torflix/internal/lib/fails"
	"github.com/stretchr/testify/assert"
)

func TestValuerError(t *testing.T) {
	err := errors.New("error")
	err2 := fails.NewWithErr(err, "wrap1", "key", "value")
	err = faults.Errorf("something: %w", err2)
	err2 = fails.NewWithErr(err, "wrap2", "key2", "value2")

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%v", err2)
	assert.Equal(t, "wrap2 (key2=value2): something: wrap1 (key=value): error", buf.String())

	assert.Equal(t, map[string]interface{}{"key": "value", "key2": "value2"}, err2.Values())
}
