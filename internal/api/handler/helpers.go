package handler

func mapSlice[In, Out any](in []In, fn func(In) Out) []Out {
	out := make([]Out, len(in)) // non-nil when empty so JSON marshals as [] instead of null
	for i, v := range in {
		out[i] = fn(v)
	}
	return out
}
