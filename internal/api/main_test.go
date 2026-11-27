package api_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/kaszta1274/macrorecipe/internal/api"
	"github.com/kaszta1274/macrorecipe/internal/auth"
	"github.com/kaszta1274/macrorecipe/internal/database"
	"github.com/kaszta1274/macrorecipe/internal/seeder"
	_ "github.com/lib/pq"
)

const (
	testTokenSecret   = "test-token-secret"
	testAdminEmail    = "admin@example.com"
	testAdminPassword = "adminpassword123"
)

var (
	testDB   *sql.DB
	testSeed seeder.SeedData
)

func TestMain(m *testing.M) {
	// go test runs with cwd=internal/api; real env vars take precedence over .env.
	_ = godotenv.Load("../../.env")

	testDBURL := os.Getenv("TEST_DB_URL")
	if testDBURL == "" {
		fmt.Println("skipping api integration tests: TEST_DB_URL not set")
		os.Exit(m.Run())
	}
	if testDBURL == os.Getenv("DB_URL") {
		fmt.Fprintln(os.Stderr, "refusing to run tests: TEST_DB_URL must not match DB_URL")
		os.Exit(1)
	}

	db, err := sql.Open("postgres", testDBURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open test db: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	if err := db.PingContext(ctx); err != nil {
		cancel()
		_ = db.Close()
		fmt.Fprintf(os.Stderr, "ping test db: %v\n", err)
		os.Exit(1)
	}

	if err := seeder.Reset(ctx, db); err != nil {
		cancel()
		_ = db.Close()
		fmt.Fprintf(os.Stderr, "reset test db: %v\n", err)
		os.Exit(1)
	}

	data, err := seeder.SeedTestData(ctx, database.New(db), testAdminEmail, testAdminPassword)
	cancel()
	if err != nil {
		_ = db.Close()
		fmt.Fprintf(os.Stderr, "seed test db: %v\n", err)
		os.Exit(1)
	}

	testDB = db
	testSeed = data

	code := m.Run()
	_ = db.Close()
	os.Exit(code)
}

func requireTestDB(t *testing.T) {
	t.Helper()
	if testDB == nil {
		t.Skip("TEST_DB_URL not set")
	}
}

func newTestTxQueries(t *testing.T) *database.Queries {
	t.Helper()
	requireTestDB(t)

	tx, err := testDB.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	t.Cleanup(func() {
		_ = tx.Rollback()
	})
	return database.New(testDB).WithTx(tx)
}

// newTestRouter runs requests inside a rolled-back tx. Postgres aborts a tx on any SQL error
// (e.g. 409 UniqueViolation), so give each subtest its own newTestRouter(t).
func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	cfg := api.ApiConfig{
		DB:          newTestTxQueries(t),
		TokenSecret: testTokenSecret,
		UploadDir:   t.TempDir(),
	}
	return cfg.Router()
}

func mustToken(t *testing.T, user database.User) string {
	t.Helper()
	token, err := auth.MakeJWTWithRole(user.ID, user.Role, testTokenSecret, time.Hour)
	if err != nil {
		t.Fatalf("make jwt: %v", err)
	}
	return token
}

func doJSON(t *testing.T, h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(rec.Body).Decode(&v); err != nil {
		t.Fatalf("decode %T: %v (body = %s)", v, err, rec.Body.String())
	}
	return v
}

func assertErrorJSON(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	resp := decodeJSON[struct {
		Error string `json:"error"`
	}](t, rec)
	if resp.Error == "" {
		t.Fatal("expected non-empty error field in JSON response")
	}
}

func TestSeederIdempotency(t *testing.T) {
	requireTestDB(t)

	t.Run("SeedBase", func(t *testing.T) {
		again, err := seeder.SeedBase(context.Background(), newTestTxQueries(t), testAdminEmail, testAdminPassword)
		if err != nil {
			t.Fatalf("SeedBase failed: %v", err)
		}
		if again.Admin.ID != testSeed.Admin.ID {
			t.Fatalf("admin ID changed: got %s, want %s", again.Admin.ID, testSeed.Admin.ID)
		}
		if len(again.Ingredients) != len(testSeed.Ingredients) {
			t.Fatalf("ingredients count = %d, want %d", len(again.Ingredients), len(testSeed.Ingredients))
		}
	})

	t.Run("SeedTestData", func(t *testing.T) {
		again, err := seeder.SeedTestData(context.Background(), newTestTxQueries(t), testAdminEmail, testAdminPassword)
		if err != nil {
			t.Fatalf("SeedTestData failed: %v", err)
		}
		if again.Admin.ID != testSeed.Admin.ID {
			t.Fatalf("admin ID changed: got %s, want %s", again.Admin.ID, testSeed.Admin.ID)
		}
		if len(again.Users) != len(testSeed.Users) || again.Users[0].ID != testSeed.Users[0].ID {
			t.Fatalf("users not idempotent: got %+v", again.Users)
		}
		if len(again.Recipes) != len(testSeed.Recipes) || again.Recipes[0].ID != testSeed.Recipes[0].ID {
			t.Fatalf("recipes not idempotent: got %+v", again.Recipes)
		}
		if len(again.Likes) != len(testSeed.Likes) || again.Likes[0].RecipeID != testSeed.Likes[0].RecipeID {
			t.Fatalf("likes not idempotent: got %+v", again.Likes)
		}
		if len(again.Comments) != len(testSeed.Comments) || again.Comments[0].ID != testSeed.Comments[0].ID {
			t.Fatalf("comments not idempotent: got %+v", again.Comments)
		}
		if len(again.Follows) != len(testSeed.Follows) || again.Follows[0].FollowingID != testSeed.Follows[0].FollowingID {
			t.Fatalf("follows not idempotent: got %+v", again.Follows)
		}
	})
}
