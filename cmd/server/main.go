package main

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"go-reader/internal/api"
	"go-reader/internal/convert"
	"go-reader/internal/fetch"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	apiKey := os.Getenv("API_KEY")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	fetcher := fetch.NewClient()
	converter := convert.NewConverter()
	handlers := api.NewHandlers(fetcher, converter)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handlers.Health)
	mux.HandleFunc("GET /readyz", handlers.Ready)
	mux.Handle("POST /extract", apiKeyAuth(apiKey, http.HandlerFunc(handlers.Extract)))
	mux.Handle("GET /read", apiKeyAuth(apiKey, http.HandlerFunc(handlers.Read)))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           requestLogger(logger, mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logger.Info("server started", "addr", server.Addr, "api_key_enabled", apiKey != "")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func apiKeyAuth(apiKey string, next http.Handler) http.Handler {
	if apiKey == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided := strings.TrimSpace(r.Header.Get("X-API-KEY"))
		if subtle.ConstantTimeCompare([]byte(provided), []byte(apiKey)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	})
}
