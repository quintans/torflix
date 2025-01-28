package slices

// Map applies a function to each element of a slice and returns a new slice with the results.
func Map[I, O any](s []I, f func(I) O) []O {
	m := make([]O, len(s))
	for i, v := range s {
		m[i] = f(v)
	}
	return m
}
