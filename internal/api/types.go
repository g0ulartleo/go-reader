package api

import "time"

type ExtractRequest struct {
	URL       string `json:"url"`
	TimeoutMS int    `json:"timeout_ms"`
	UserAgent string `json:"user_agent"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ExtractResponse struct {
	URL         string    `json:"url"`
	FinalURL    string    `json:"final_url,omitempty"`
	StatusCode  int       `json:"status_code,omitempty"`
	ContentType string    `json:"content_type,omitempty"`
	Title       string    `json:"title,omitempty"`
	Markdown    string    `json:"markdown,omitempty"`
	ElapsedMS   int64     `json:"elapsed_ms"`
	FetchedAt   time.Time `json:"fetched_at"`
	Error       *APIError `json:"error"`
}
