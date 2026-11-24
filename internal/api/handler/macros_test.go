package handler

import "testing"

func TestCalculateMacros(t *testing.T) {
	// Seeded Chicken and Rice Bowl totals (2 portions): P=73.3, F=18.5, C=89.9
	total := calculateMacros(73.3, 18.5, 89.9)
	if total.Calories != 819.3 || total.Protein != 73.3 || total.Fat != 18.5 || total.Carbohydrates != 89.9 {
		t.Fatalf("total = %+v, want {819.3 73.3 18.5 89.9}", total)
	}

	perPortion := calculateMacrosPerPortion(73.3, 18.5, 89.9, 2)
	if perPortion.Calories != 409.7 || perPortion.Protein != 36.7 || perPortion.Fat != 9.3 || perPortion.Carbohydrates != 45.0 {
		t.Fatalf("perPortion = %+v, want {409.7 36.7 9.3 45.0}", perPortion)
	}

	zero := calculateMacrosPerPortion(0, 0, 0, 0)
	if zero != (Macros{}) {
		t.Fatalf("zero = %+v, want all zeros", zero)
	}
}
