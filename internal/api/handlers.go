package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"go-reader/internal/convert"
	"go-reader/internal/extract"
	"go-reader/internal/fetch"
	"go-reader/internal/validate"
)

type Handlers struct {
	fetcher   *fetch.Client
	converter *convert.Converter
}

func NewHandlers(fetcher *fetch.Client, converter *convert.Converter) *Handlers {
	return &Handlers{fetcher: fetcher, converter: converter}
}

func (h *Handlers) Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handlers) Ready(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handlers) Extract(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	res := ExtractResponse{FetchedAt: started.UTC()}

	var req ExtractRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 64*1024)).Decode(&req); err != nil {
		res.Error = &APIError{Code: "ERR_BAD_REQUEST", Message: "invalid json body"}
		res.ElapsedMS = time.Since(started).Milliseconds()
		writeJSON(w, http.StatusBadRequest, res)
		return
	}

	urlValue, timeoutMS, err := validate.Input(req.URL, req.TimeoutMS)
	if err != nil {
		res.Error = &APIError{Code: "ERR_BAD_REQUEST", Message: err.Error()}
		res.ElapsedMS = time.Since(started).Milliseconds()
		writeJSON(w, http.StatusBadRequest, res)
		return
	}

	userAgent := resolveUserAgent(r, req.UserAgent)
	out, status := h.process(r.Context(), urlValue, timeoutMS, userAgent, started)
	writeJSON(w, status, out)
}

func (h *Handlers) Read(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	urlValue := strings.TrimSpace(r.URL.Query().Get("url"))
	if urlValue == "" {
		http.Error(w, "missing url query param", http.StatusBadRequest)
		return
	}

	_, timeoutMS, err := validate.Input(urlValue, validate.TimeoutParam(r.URL.Query().Get("timeout_ms")))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userAgent := resolveUserAgent(r, "")
	out, status := h.process(r.Context(), urlValue, timeoutMS, userAgent, started)
	if status != http.StatusOK {
		message := "request failed"
		if out.Error != nil {
			message = out.Error.Message
		}
		http.Error(w, message, status)
		return
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(out.Markdown))
}

func (h *Handlers) process(parent context.Context, urlValue string, timeoutMS int, userAgent string, started time.Time) (ExtractResponse, int) {
	res := ExtractResponse{URL: urlValue, FetchedAt: started.UTC()}

	ctx, cancel := context.WithTimeout(parent, time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()

	fetchResult, err := h.fetcher.Get(ctx, urlValue, userAgent)
	if err != nil {
		res.Error = mapFetchError(err)
		res.ElapsedMS = time.Since(started).Milliseconds()
		if res.Error.Code == "ERR_TIMEOUT" {
			return res, http.StatusGatewayTimeout
		}
		var upstream *fetch.UpstreamStatusError
		if errors.As(err, &upstream) && upstream.StatusCode >= 400 && upstream.StatusCode < 500 {
			res.StatusCode = upstream.StatusCode
			return res, upstream.StatusCode
		}
		return res, http.StatusBadGateway
	}

	res.FinalURL = fetchResult.FinalURL
	res.StatusCode = fetchResult.StatusCode
	res.ContentType = fetchResult.ContentType
	res.Title = strings.TrimSpace(extract.Title(fetchResult.Body))

	mainContent := extract.MainContent(fetchResult.Body)
	markdown, err := h.converter.HTMLToMarkdown(mainContent, fetchResult.FinalURL)
	if err != nil {
		res.Error = &APIError{Code: "ERR_CONVERT_FAILED", Message: "failed to convert html to markdown"}
		res.ElapsedMS = time.Since(started).Milliseconds()
		return res, http.StatusUnprocessableEntity
	}

	res.Markdown = markdown
	res.ElapsedMS = time.Since(started).Milliseconds()
	return res, http.StatusOK
}

func resolveUserAgent(r *http.Request, bodyUserAgent string) string {
	headerUserAgent := strings.TrimSpace(r.Header.Get("X-User-Agent"))
	if headerUserAgent != "" {
		return headerUserAgent
	}

	bodyUserAgent = strings.TrimSpace(bodyUserAgent)
	if bodyUserAgent != "" {
		return bodyUserAgent
	}

	return ""
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func mapFetchError(err error) *APIError {
	if fetch.IsTimeout(err) {
		return &APIError{Code: "ERR_TIMEOUT", Message: "upstream request timed out"}
	}
	msg := err.Error()
	if strings.Contains(msg, "upstream status") {
		return &APIError{Code: "ERR_UPSTREAM_STATUS", Message: msg}
	}
	if strings.Contains(msg, "bad url") {
		return &APIError{Code: "ERR_BAD_URL", Message: "invalid target url"}
	}
	if strings.Contains(msg, "response too large") {
		return &APIError{Code: "ERR_RESPONSE_TOO_LARGE", Message: msg}
	}
	return &APIError{Code: "ERR_FETCH_FAILED", Message: "failed to fetch url"}
}
