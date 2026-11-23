package seeder

import "github.com/kaszta1274/macrorecipe/internal/database"

const (
	TestUser1Email    = "alice@example.com"
	TestUser1Username = "alice"
	TestUser2Email    = "bob@example.com"
	TestUser2Username = "bob"
	TestUserPassword  = "password123"
)

type seedUser struct {
	email    string
	username string
	password string
}

type seedRecipeIngredient struct {
	ingredientName string
	amount         float32
}

type seedRecipe struct {
	ownerIndex   int
	name         string
	description  string
	instructions string
	portions     int16
	ingredients  []seedRecipeIngredient
}

type seedLike struct {
	userIndex   int
	recipeIndex int
}

type seedComment struct {
	userIndex   int
	recipeIndex int
	body        string
}

// Per-100g macros for starter ingredients seeded in prod and dev modes.
var starterIngredients = []database.CreateIngredientParams{
	{Name: "Chicken Breast", Protein: 31.0, Fat: 3.6, Carbohydrates: 0.0},
	{Name: "White Rice", Protein: 7.1, Fat: 0.7, Carbohydrates: 80.0},
	{Name: "Rolled Oats", Protein: 13.2, Fat: 6.5, Carbohydrates: 67.7},
	{Name: "Whole Eggs", Protein: 12.6, Fat: 9.5, Carbohydrates: 0.7},
	{Name: "Olive Oil", Protein: 0.0, Fat: 100.0, Carbohydrates: 0.0},
	{Name: "Greek Yogurt", Protein: 10.0, Fat: 0.4, Carbohydrates: 3.6},
	{Name: "Broccoli", Protein: 2.8, Fat: 0.4, Carbohydrates: 6.6},
	{Name: "Salmon Fillet", Protein: 20.4, Fat: 13.4, Carbohydrates: 0.0},
	{Name: "Sweet Potato", Protein: 1.6, Fat: 0.1, Carbohydrates: 20.1},
	{Name: "Whey Protein", Protein: 80.0, Fat: 5.0, Carbohydrates: 6.0},
	{Name: "Banana", Protein: 1.1, Fat: 0.3, Carbohydrates: 22.8},
	{Name: "Peanut Butter", Protein: 25.0, Fat: 50.0, Carbohydrates: 20.0},
}

var testUsers = []seedUser{
	{email: TestUser1Email, username: TestUser1Username, password: TestUserPassword},
	{email: TestUser2Email, username: TestUser2Username, password: TestUserPassword},
}

var testRecipes = []seedRecipe{
	{
		ownerIndex:   0,
		name:         "Chicken and Rice Bowl",
		description:  "Simple high-protein chicken, rice, and steamed broccoli bowl.",
		instructions: "Cook rice, pan-sear chicken in olive oil, steam broccoli, and combine.",
		portions:     2,
		ingredients: []seedRecipeIngredient{
			{ingredientName: "Chicken Breast", amount: 200},
			{ingredientName: "White Rice", amount: 100},
			{ingredientName: "Broccoli", amount: 150},
			{ingredientName: "Olive Oil", amount: 10},
		},
	},
	{
		ownerIndex:   0,
		name:         "Protein Oatmeal",
		description:  "Quick breakfast oats mixed with whey protein, banana, and peanut butter.",
		instructions: "Simmer oats in water, stir in whey protein off heat, top with sliced banana and peanut butter.",
		portions:     1,
		ingredients: []seedRecipeIngredient{
			{ingredientName: "Rolled Oats", amount: 80},
			{ingredientName: "Whey Protein", amount: 30},
			{ingredientName: "Banana", amount: 100},
			{ingredientName: "Peanut Butter", amount: 15},
		},
	},
	{
		ownerIndex:   1,
		name:         "Baked Salmon with Sweet Potato",
		description:  "Oven-roasted salmon fillet served with crispy sweet potato wedges.",
		instructions: "Toss sweet potato wedges in olive oil, roast at 200C for 25 minutes, bake salmon for last 12 minutes.",
		portions:     1,
		ingredients: []seedRecipeIngredient{
			{ingredientName: "Salmon Fillet", amount: 180},
			{ingredientName: "Sweet Potato", amount: 250},
			{ingredientName: "Olive Oil", amount: 10},
		},
	},
}

var testLikes = []seedLike{
	{userIndex: 0, recipeIndex: 2}, // alice likes bob's Baked Salmon with Sweet Potato
	{userIndex: 1, recipeIndex: 0}, // bob likes alice's Chicken and Rice Bowl
}

var testComments = []seedComment{
	{userIndex: 0, recipeIndex: 0, body: "Great for weekly meal prep."},
	{userIndex: 1, recipeIndex: 0, body: "Tried this with brown rice and it turned out awesome!"},
	{userIndex: 0, recipeIndex: 2, body: "Crispy sweet potatoes make this recipe."},
}
