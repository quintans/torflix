package fails

import (
	"errors"
	"maps"
	"strings"

	"github.com/quintans/torflix/internal/lib/values"
)

type Valuer interface {
	error
	Values() map[string]any
	WithValues(args ...any) Valuer
}

func New(msg string, args ...any) Valuer {
	return &ValuesError{
		msg:    msg,
		values: values.ToMap(args),
	}
}

func NewWithErr(err error, msg string, args ...any) Valuer {
	return &ValuesError{
		err:    err,
		msg:    msg,
		values: values.ToMap(args),
	}
}

type ValuesError struct {
	err    error
	msg    string
	values map[string]any
}

func (e *ValuesError) Error() string {
	var str strings.Builder
	str.WriteString(e.msg)
	if len(e.values) > 0 {
		str.WriteString(" ")
		str.WriteString(values.ToStr(e.values))
	}

	if e.err != nil {
		str.WriteString(": ")
		str.WriteString(e.err.Error())
	}

	return str.String()
}

func (e *ValuesError) Unwrap() error {
	return e.err
}

// Values returns the values associated with the error and its cause.
func (e *ValuesError) Values() map[string]any {
	m := maps.Clone(e.values)
	var valuer Valuer
	if errors.As(e.err, &valuer) {
		for k, v := range valuer.Values() {
			m[k] = v
		}
	}
	return m
}

func (e *ValuesError) WithValues(args ...any) Valuer {
	m := values.ToMap(args)
	for k, v := range m {
		e.values[k] = v
	}

	return e
}
