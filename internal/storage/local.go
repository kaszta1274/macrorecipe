package storage

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const MaxImageBytes = 5 << 20 // 5 MB

var (
	ErrFileTooLarge     = errors.New("file exceeds 5MB limit")
	ErrInvalidImageType = errors.New("unsupported image type: allowed jpeg, png, webp")
	ErrMissingFile      = errors.New("image file is required")
)

var allowedImageExts = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type LocalStorage struct {
	BaseDir string
}

func NewLocalStorage(baseDir string) *LocalStorage {
	if baseDir == "" {
		baseDir = "./uploads"
	}
	return &LocalStorage{BaseDir: baseDir}
}

func (s *LocalStorage) SaveImage(w http.ResponseWriter, r *http.Request, fieldName, subdir string) (string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxImageBytes)

	if err := r.ParseMultipartForm(MaxImageBytes); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) || strings.Contains(err.Error(), "request body too large") {
			return "", ErrFileTooLarge
		}
		return "", fmt.Errorf("%w: %v", ErrMissingFile, err)
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, _, err := r.FormFile(fieldName)
	if err != nil {
		return "", ErrMissingFile
	}
	defer file.Close()

	head := make([]byte, 512)
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", fmt.Errorf("read image header: %w", err)
	}
	if n == 0 {
		return "", ErrInvalidImageType
	}

	mime := http.DetectContentType(head[:n])
	ext, ok := allowedImageExts[mime]
	if !ok {
		return "", ErrInvalidImageType
	}

	targetDir := filepath.Join(s.BaseDir, subdir)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("create upload dir: %w", err)
	}

	filename := uuid.NewString() + ext
	dstPath := filepath.Join(targetDir, filename)

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return "", fmt.Errorf("create image file: %w", err)
	}

	if _, err := dst.Write(head[:n]); err != nil {
		_ = dst.Close()
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("write image header: %w", err)
	}
	if _, err := io.Copy(dst, file); err != nil {
		_ = dst.Close()
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("write image body: %w", err)
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("close image file: %w", err)
	}

	return "/uploads/" + subdir + "/" + filename, nil
}

func (s *LocalStorage) Delete(urlPath string) error {
	if urlPath == "" {
		return nil
	}
	if !strings.HasPrefix(urlPath, "/uploads/") {
		return errors.New("invalid upload path")
	}

	rel := filepath.Clean(strings.TrimPrefix(urlPath, "/uploads/"))
	if rel == "." || rel == "" || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return errors.New("invalid upload path")
	}

	absBase, err := filepath.Abs(s.BaseDir)
	if err != nil {
		return err
	}
	absTarget, err := filepath.Abs(filepath.Join(absBase, rel))
	if err != nil {
		return err
	}
	checkRel, err := filepath.Rel(absBase, absTarget)
	if err != nil || strings.HasPrefix(checkRel, "..") {
		return errors.New("invalid upload path")
	}

	if err := os.Remove(absTarget); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *LocalStorage) FileServer() http.Handler {
	fs := http.FileServer(noDirFS{http.Dir(s.BaseDir)})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fs.ServeHTTP(w, r)
	})
}

// noDirFS disables directory listings so /uploads/ cannot be browsed.
type noDirFS struct {
	fs http.FileSystem
}

func (n noDirFS) Open(name string) (http.File, error) {
	f, err := n.fs.Open(name)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if stat.IsDir() {
		_ = f.Close()
		return nil, os.ErrNotExist
	}
	return f, nil
}
