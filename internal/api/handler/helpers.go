package handler

import "math"

type Macros struct {
	Calories      float32 `json:"calories"`
	Protein       float32 `json:"protein"`
	Fat           float32 `json:"fat"`
	Carbohydrates float32 `json:"carbohydrates"`
}

func calculateMacros(protein, fat, carbs float64) Macros {
	return calculateMacrosPerPortion(protein, fat, carbs, 1)
}

func calculateMacrosPerPortion(protein, fat, carbs float64, portions int) Macros {
	if portions < 1 {
		portions = 1
	}
	n := float64(portions)
	// Strip float32 storage noise (REAL columns) before dividing so list and detail views agree.
	p := (math.Round(protein*100) / 100) / n
	f := (math.Round(fat*100) / 100) / n
	c := (math.Round(carbs*100) / 100) / n
	return Macros{
		Calories:      round1(4*p + 9*f + 4*c),
		Protein:       round1(p),
		Fat:           round1(f),
		Carbohydrates: round1(c),
	}
}

func round1(v float64) float32 {
	return float32(math.Round(v*10) / 10)
}

func mapSlice[In, Out any](in []In, fn func(In) Out) []Out {
	out := make([]Out, len(in)) // non-nil when empty so JSON marshals as [] instead of null
	for i, v := range in {
		out[i] = fn(v)
	}
	return out
}
