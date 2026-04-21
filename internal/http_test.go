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

func newTestHandler(t *testing.T, allowAll bool) (http.Handler, *stubGenerator) {
	t.Helper()
	stub := &stubGenerator{payload: []byte("<svg/>"), dir: t.TempDir(), imgType: "svg"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	var h http.Handler = generateHTTPHandler(stub, logger)
	if allowAll {
		h = allowAllOriginsMiddleware(h)
	}
	return h, stub
}

func TestHandlerPOSTReturnsSVG(t *testing.T) {
	h, _ := newTestHandler(t, false)
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

func TestHandlerGETReturnsSVG(t *testing.T) {
	h, _ := newTestHandler(t, false)
	u := "/generate?data=" + url.QueryEscape("graph LR\nA-->B")
	req := httptest.NewRequest(http.MethodGet, u, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerRejectsUnsupportedImageType(t *testing.T) {
	h, _ := newTestHandler(t, false)
	req := httptest.NewRequest(http.MethodPost, "/generate?type=gif", strings.NewReader("x"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandlerRejectsUnsupportedMethod(t *testing.T) {
	h, _ := newTestHandler(t, false)
	req := httptest.NewRequest(http.MethodDelete, "/generate", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandlerGETMissingData(t *testing.T) {
	h, _ := newTestHandler(t, false)
	req := httptest.NewRequest(http.MethodGet, "/generate", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestAllowAllOriginsPreflight(t *testing.T) {
	h, _ := newTestHandler(t, true)
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
