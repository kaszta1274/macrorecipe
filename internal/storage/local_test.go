package storage_test

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kaszta1274/macrorecipe/internal/storage"
)

func makePNGBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func makeMultipartReq(t *testing.T, fieldName, filename string, data []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/upload", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestLocalStorageSaveServeDelete(t *testing.T) {
	store := storage.NewLocalStorage(t.TempDir())
	pngData := makePNGBytes(t)

	rec := httptest.NewRecorder()
	req := makeMultipartReq(t, "image", "photo.png", pngData)

	urlPath, err := store.SaveImage(rec, req, "image", "recipes")
	if err != nil {
		t.Fatalf("SaveImage failed: %v", err)
	}
	if !strings.HasPrefix(urlPath, "/uploads/recipes/") || !strings.HasSuffix(urlPath, ".png") {
		t.Fatalf("unexpected urlPath = %q", urlPath)
	}

	// File server serves uploaded bytes with security headers.
	srv := http.StripPrefix("/uploads/", store.FileServer())
	getRec := httptest.NewRecorder()
	srv.ServeHTTP(getRec, httptest.NewRequest(http.MethodGet, urlPath, nil))
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200", urlPath, getRec.Code)
	}
	if getRec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff header")
	}
	if !bytes.Equal(getRec.Body.Bytes(), pngData) {
		t.Fatal("served bytes did not match uploaded png")
	}

	// Directory listing is disabled.
	for _, dirPath := range []string{"/uploads/", "/uploads/recipes/"} {
		dirRec := httptest.NewRecorder()
		srv.ServeHTTP(dirRec, httptest.NewRequest(http.MethodGet, dirPath, nil))
		if dirRec.Code != http.StatusNotFound {
			t.Fatalf("GET %s status = %d, want 404", dirPath, dirRec.Code)
		}
	}

	// Delete removes the file.
	if err := store.Delete(urlPath); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	afterRec := httptest.NewRecorder()
	srv.ServeHTTP(afterRec, httptest.NewRequest(http.MethodGet, urlPath, nil))
	if afterRec.Code != http.StatusNotFound {
		t.Fatalf("GET after Delete status = %d, want 404", afterRec.Code)
	}
}

func TestLocalStorageValidation(t *testing.T) {
	store := storage.NewLocalStorage(t.TempDir())

	t.Run("rejects svg", func(t *testing.T) {
		req := makeMultipartReq(t, "image", "evil.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`))
		if _, err := store.SaveImage(httptest.NewRecorder(), req, "image", "avatars"); !errors.Is(err, storage.ErrInvalidImageType) {
			t.Fatalf("err = %v, want ErrInvalidImageType", err)
		}
	})

	t.Run("rejects plain text disguised as jpg", func(t *testing.T) {
		req := makeMultipartReq(t, "image", "fake.jpg", []byte("not really a jpeg"))
		if _, err := store.SaveImage(httptest.NewRecorder(), req, "image", "avatars"); !errors.Is(err, storage.ErrInvalidImageType) {
			t.Fatalf("err = %v, want ErrInvalidImageType", err)
		}
	})

	t.Run("rejects oversized file", func(t *testing.T) {
		big := make([]byte, storage.MaxImageBytes+1024)
		req := makeMultipartReq(t, "image", "huge.png", big)
		if _, err := store.SaveImage(httptest.NewRecorder(), req, "image", "recipes"); !errors.Is(err, storage.ErrFileTooLarge) {
			t.Fatalf("err = %v, want ErrFileTooLarge", err)
		}
	})

	t.Run("blocks path traversal on delete", func(t *testing.T) {
		if err := store.Delete("/uploads/../../etc/passwd"); err == nil {
			t.Fatal("expected error on path traversal delete")
		}
	})
}
