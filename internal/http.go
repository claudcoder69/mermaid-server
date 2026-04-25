package internal

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

//go:embed openapi.yaml
var openapiSpec []byte

//go:embed docs.html
var docsPage []byte

// RunHTTPServer starts the HTTP server and blocks until ctx is cancelled,
// at which point it gracefully shuts the server down.
func RunHTTPServer(ctx context.Context, logger *slog.Logger, generator Generator, addr string, allowAllOrigins bool) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           NewRouter(logger, generator, allowAllOrigins),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down http server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	case err := <-serverErr:
		return err
	}
}

// NewRouter wires the HTTP routes. Exposed so tests can hit the full router.
func NewRouter(logger *slog.Logger, generator Generator, allowAllOrigins bool) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	if allowAllOrigins {
		r.Use(allowAllOriginsMiddleware)
	}

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	})

	r.Get("/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		_, _ = w.Write(openapiSpec)
	})

	r.Get("/docs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(docsPage)
	})

	gen := generateHTTPHandler(generator, logger)
	r.Get("/generate", gen)
	r.Post("/generate", gen)

	return r
}

// allowAllOriginsMiddleware sets permissive CORS headers and handles preflight.
func allowAllOriginsMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func writeJSON(rw http.ResponseWriter, value any, status int) {
	bytes, err := json.Marshal(value)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	_, _ = rw.Write(bytes)
}

func writeImage(rw http.ResponseWriter, data []byte, status int, imgType string) error {
	switch imgType {
	case "png":
		rw.Header().Set("Content-Type", "image/png")
	case "svg":
		rw.Header().Set("Content-Type", "image/svg+xml")
	default:
		return fmt.Errorf("unhandled image type: %s", imgType)
	}
	rw.WriteHeader(status)
	if _, err := rw.Write(data); err != nil {
		return fmt.Errorf("could not write image bytes: %w", err)
	}
	return nil
}

func writeErr(rw http.ResponseWriter, logger *slog.Logger, err error, status int) {
	logger.Error("request failed", "status", status, "err", err)
	writeJSON(rw, map[string]string{"error": err.Error()}, status)
}

// URLParam is the URL parameter getDiagramFromGET uses to look for data.
const URLParam = "data"

func getDiagramFromGET(r *http.Request, imgType string) (*Diagram, error) {
	queryVal := strings.TrimSpace(r.URL.Query().Get(URLParam))
	if queryVal == "" {
		return nil, fmt.Errorf("missing data")
	}
	data, err := url.QueryUnescape(queryVal)
	if err != nil {
		return nil, fmt.Errorf("could not read query param: %s", err)
	}
	return NewDiagram([]byte(data), imgType), nil
}

func getDiagramFromPOST(r *http.Request, imgType string) (*Diagram, error) {
	body, err := io.ReadAll(http.MaxBytesReader(nil, r.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("could not read body: %s", err)
	}
	return NewDiagram(body, imgType), nil
}

const URLParamImageType = "type"

// generateHTTPHandler returns a HTTP handler used to generate a diagram.
// Method dispatch is handled by the router; this handler assumes GET or POST.
func generateHTTPHandler(generator Generator, logger *slog.Logger) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		imgType := r.URL.Query().Get(URLParamImageType)
		switch imgType {
		case "png", "svg":
		case "":
			imgType = "svg"
		default:
			writeErr(rw, logger, fmt.Errorf("unsupported image type (%s) use svg or png", imgType), http.StatusBadRequest)
			return
		}

		var (
			diagram *Diagram
			err     error
		)
		switch r.Method {
		case http.MethodGet:
			diagram, err = getDiagramFromGET(r, imgType)
		case http.MethodPost:
			diagram, err = getDiagramFromPOST(r, imgType)
		}
		if err != nil {
			writeErr(rw, logger, err, http.StatusBadRequest)
			return
		}

		if err := generator.Generate(diagram); err != nil {
			writeErr(rw, logger, fmt.Errorf("could not generate diagram: %s", err), http.StatusInternalServerError)
			return
		}

		diagramBytes, err := os.ReadFile(diagram.Output)
		if err != nil {
			writeErr(rw, logger, fmt.Errorf("could not read diagram bytes: %s", err), http.StatusInternalServerError)
			return
		}
		if err := writeImage(rw, diagramBytes, http.StatusOK, imgType); err != nil {
			writeErr(rw, logger, fmt.Errorf("could not write diagram: %w", err), http.StatusInternalServerError)
		}
	}
}
