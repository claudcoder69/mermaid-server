package internal

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// stubGenerator writes a fixed payload to disk so the HTTP layer has something to serve.
type stubGenerator struct {
	payload []byte
	dir     string
	imgType string
}

func (s *stubGenerator) Generate(d *Diagram) error {
	id, err := d.ID()
	if err != nil {
		return err
	}
	d.Output = filepath.Join(s.dir, id+"."+s.imgType)
	return os.WriteFile(d.Output, s.payload, 0o644)
}

func (s *stubGenerator) CleanUp(time.Duration) error { return nil }

func newTestRouter(t *testing.T, allowAll bool) (http.Handler, *stubGenerator) {
	t.Helper()
	stub := &stubGenerator{payload: []byte("<svg/>"), dir: t.TempDir(), imgType: "svg"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(logger, stub, allowAll), stub
}

func TestRouterPOSTReturnsSVG(t *testing.T) {
	h, _ := newTestRouter(t, false)
	req := httptest.NewRequest(http.MethodPost, "/generate", strings.NewReader("graph LR\nA-->B"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Fatalf("Content-Type = %q", ct)
	}
	if rec.Body.String() != "<svg/>" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestRouterGETReturnsSVG(t *testing.T) {
	h, _ := newTestRouter(t, false)
	u := "/generate?data=" + url.QueryEscape("graph LR\nA-->B")
	req := httptest.NewRequest(http.MethodGet, u, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestRouterRejectsUnsupportedImageType(t *testing.T) {
	h, _ := newTestRouter(t, false)
	req := httptest.NewRequest(http.MethodPost, "/generate?type=gif", strings.NewReader("x"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestRouterRejectsUnsupportedMethod(t *testing.T) {
	h, _ := newTestRouter(t, false)
	req := httptest.NewRequest(http.MethodDelete, "/generate", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestRouterGETMissingData(t *testing.T) {
	h, _ := newTestRouter(t, false)
	req := httptest.NewRequest(http.MethodGet, "/generate", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestRouterAllowAllOriginsPreflight(t *testing.T) {
	h, _ := newTestRouter(t, true)
	req := httptest.NewRequest(http.MethodOptions, "/generate", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}

func TestRouterHealth(t *testing.T) {
	h, _ := newTestRouter(t, false)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestRouterOpenAPISpec(t *testing.T) {
	h, _ := newTestRouter(t, false)
	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/yaml") {
		t.Fatalf("Content-Type = %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "openapi: 3.0.3") {
		t.Fatalf("expected OpenAPI version header in body, got:\n%s", body)
	}
	if !strings.Contains(body, "/generate:") || !strings.Contains(body, "/health:") {
		t.Fatal("expected /generate and /health paths in the served spec")
	}
}

func TestRouterDocsPage(t *testing.T) {
	h, _ := newTestRouter(t, false)
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "swagger-ui") {
		t.Fatal("docs page should reference swagger-ui")
	}
}
