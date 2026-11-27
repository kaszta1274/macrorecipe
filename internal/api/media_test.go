package api_test

import (
	"bytes"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/api/handler"
)

func TestMediaUploads(t *testing.T) {
	requireTestDB(t)

	alice := testSeed.Users[0]
	bob := testSeed.Users[1]
	aliceRecipe := testSeed.Recipes[0]

	aliceToken := mustToken(t, alice)
	bobToken := mustToken(t, bob)
	adminToken := mustToken(t, testSeed.Admin)

	pngBytes := makeTestPNG(t)

	t.Run("user avatar upload serve replace delete", func(t *testing.T) {
		router := newTestRouter(t)

		// Upload initial avatar
		rec := doMultipartImage(t, router, http.MethodPut, "/api/users/avatar", aliceToken, "avatar.png", pngBytes)
		if rec.Code != http.StatusOK {
			t.Fatalf("upload avatar status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}
		first := decodeJSON[map[string]*string](t, rec)["avatar_url"]
		if first == nil || !strings.HasPrefix(*first, "/uploads/avatars/") {
			t.Fatalf("avatar_url = %v, want /uploads/avatars/...", first)
		}

		// Served publicly via /uploads/* without auth header
		fileRec := doJSON(t, router, http.MethodGet, *first, "", nil)
		if fileRec.Code != http.StatusOK || !bytes.Equal(fileRec.Body.Bytes(), pngBytes) {
			t.Fatalf("GET %s status = %d, want 200 with matching png bytes", *first, fileRec.Code)
		}

		// Exposed on GET /api/users/me and GET /api/users/{id}
		meRec := doJSON(t, router, http.MethodGet, "/api/users/me", aliceToken, nil)
		me := decodeJSON[handler.User](t, meRec)
		if me.AvatarURL == nil || *me.AvatarURL != *first {
			t.Fatalf("me.avatar_url = %v, want %q", me.AvatarURL, *first)
		}

		pubRec := doJSON(t, router, http.MethodGet, "/api/users/"+alice.ID.String(), bobToken, nil)
		pub := decodeJSON[handler.UserPublicProfile](t, pubRec)
		if pub.AvatarURL == nil || *pub.AvatarURL != *first {
			t.Fatalf("public.avatar_url = %v, want %q", pub.AvatarURL, *first)
		}

		// Replace avatar deletes old file
		rec2 := doMultipartImage(t, router, http.MethodPut, "/api/users/avatar", aliceToken, "avatar2.png", pngBytes)
		if rec2.Code != http.StatusOK {
			t.Fatalf("replace avatar status = %d, want 200", rec2.Code)
		}
		second := decodeJSON[map[string]*string](t, rec2)["avatar_url"]
		if second == nil || *second == *first {
			t.Fatalf("expected new avatar_url, got %v", second)
		}
		if oldRec := doJSON(t, router, http.MethodGet, *first, "", nil); oldRec.Code != http.StatusNotFound {
			t.Fatalf("old avatar still served: status = %d, want 404", oldRec.Code)
		}

		// Delete avatar clears DB and removes file
		delRec := doJSON(t, router, http.MethodDelete, "/api/users/avatar", aliceToken, nil)
		if delRec.Code != http.StatusNoContent {
			t.Fatalf("delete avatar status = %d, want 204", delRec.Code)
		}
		if goneRec := doJSON(t, router, http.MethodGet, *second, "", nil); goneRec.Code != http.StatusNotFound {
			t.Fatalf("deleted avatar still served: status = %d, want 404", goneRec.Code)
		}
		meAfter := decodeJSON[handler.User](t, doJSON(t, router, http.MethodGet, "/api/users/me", aliceToken, nil))
		if meAfter.AvatarURL != nil {
			t.Fatalf("expected nil avatar_url after delete, got %q", *meAfter.AvatarURL)
		}
	})

	t.Run("recipe image upload serve replace delete", func(t *testing.T) {
		router := newTestRouter(t)
		imagePath := "/api/recipes/" + aliceRecipe.ID.String() + "/image"

		// Upload recipe image
		rec := doMultipartImage(t, router, http.MethodPut, imagePath, aliceToken, "bowl.png", pngBytes)
		if rec.Code != http.StatusOK {
			t.Fatalf("upload recipe image status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}
		first := decodeJSON[map[string]*string](t, rec)["image_url"]
		if first == nil || !strings.HasPrefix(*first, "/uploads/recipes/") {
			t.Fatalf("image_url = %v, want /uploads/recipes/...", first)
		}

		// Served publicly via /uploads/*
		fileRec := doJSON(t, router, http.MethodGet, *first, "", nil)
		if fileRec.Code != http.StatusOK || !bytes.Equal(fileRec.Body.Bytes(), pngBytes) {
			t.Fatalf("GET %s status = %d, want 200 with matching png bytes", *first, fileRec.Code)
		}

		// Exposed on GET /api/recipes/{id} and GET /api/recipes
		detail := decodeJSON[handler.RecipeDetail](t, doJSON(t, router, http.MethodGet, "/api/recipes/"+aliceRecipe.ID.String(), aliceToken, nil))
		if detail.ImageURL == nil || *detail.ImageURL != *first {
			t.Fatalf("detail.image_url = %v, want %q", detail.ImageURL, *first)
		}

		// Admin can replace and delete
		rec2 := doMultipartImage(t, router, http.MethodPut, imagePath, adminToken, "bowl2.png", pngBytes)
		if rec2.Code != http.StatusOK {
			t.Fatalf("admin replace status = %d, want 200", rec2.Code)
		}
		second := decodeJSON[map[string]*string](t, rec2)["image_url"]
		if oldRec := doJSON(t, router, http.MethodGet, *first, "", nil); oldRec.Code != http.StatusNotFound {
			t.Fatalf("old recipe image still served: status = %d, want 404", oldRec.Code)
		}

		delRec := doJSON(t, router, http.MethodDelete, imagePath, adminToken, nil)
		if delRec.Code != http.StatusNoContent {
			t.Fatalf("delete recipe image status = %d, want 204", delRec.Code)
		}
		if goneRec := doJSON(t, router, http.MethodGet, *second, "", nil); goneRec.Code != http.StatusNotFound {
			t.Fatalf("deleted recipe image still served: status = %d, want 404", goneRec.Code)
		}
	})

	errorCases := []struct {
		name       string
		method     string
		path       string
		token      string
		data       []byte
		wantStatus int
	}{
		{"avatar invalid file type", http.MethodPut, "/api/users/avatar", aliceToken, []byte("not an image"), http.StatusBadRequest},
		{"avatar unauthenticated", http.MethodPut, "/api/users/avatar", "", pngBytes, http.StatusUnauthorized},
		{"recipe image invalid file type", http.MethodPut, "/api/recipes/" + aliceRecipe.ID.String() + "/image", aliceToken, []byte("<svg></svg>"), http.StatusBadRequest},
		{"recipe image bob forbidden", http.MethodPut, "/api/recipes/" + aliceRecipe.ID.String() + "/image", bobToken, pngBytes, http.StatusForbidden},
		{"recipe image delete bob forbidden", http.MethodDelete, "/api/recipes/" + aliceRecipe.ID.String() + "/image", bobToken, nil, http.StatusForbidden},
		{"recipe image missing recipe", http.MethodPut, "/api/recipes/" + uuid.NewString() + "/image", aliceToken, pngBytes, http.StatusNotFound},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouter(t)

			var rec *httptest.ResponseRecorder
			if tc.data != nil {
				rec = doMultipartImage(t, router, tc.method, tc.path, tc.token, "file.png", tc.data)
			} else {
				rec = doJSON(t, router, tc.method, tc.path, tc.token, nil)
			}
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			assertErrorJSON(t, rec)
		})
	}

	t.Run("uploads directory listing disabled", func(t *testing.T) {
		router := newTestRouter(t)
		if rec := doJSON(t, router, http.MethodGet, "/uploads/", "", nil); rec.Code != http.StatusNotFound {
			t.Fatalf("GET /uploads/ status = %d, want 404", rec.Code)
		}
	})
}

func makeTestPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func doMultipartImage(t *testing.T, h http.Handler, method, path, token, filename string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("image", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
