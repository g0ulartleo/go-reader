package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"go-reader/internal/convert"
	"go-reader/internal/fetch"
)

func TestReadStatusForUpstreamFailures(t *testing.T) {
	cases := []struct {
		upstream int
		want     int
	}{
		{http.StatusForbidden, http.StatusForbidden},
		{http.StatusNotFound, http.StatusNotFound},
		{http.StatusGone, http.StatusGone},
		{http.StatusUnauthorized, http.StatusBadGateway},
		{http.StatusTooManyRequests, http.StatusBadGateway},
		{http.StatusInternalServerError, http.StatusBadGateway},
	}
	handlers := NewHandlers(fetch.NewClient(), convert.NewConverter())

	for _, tc := range cases {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.upstream)
		}))
		rec := httptest.NewRecorder()
		handlers.Read(rec, httptest.NewRequest(http.MethodGet, "/read?url="+url.QueryEscape(upstream.URL), nil))
		upstream.Close()

		if rec.Code != tc.want {
			t.Errorf("upstream %d: got %d, want %d", tc.upstream, rec.Code, tc.want)
		}
	}
}

func TestExtractReportsUpstreamStatus(t *testing.T) {
	handlers := NewHandlers(fetch.NewClient(), convert.NewConverter())
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()

	body := strings.NewReader(`{"url":"` + upstream.URL + `"}`)
	rec := httptest.NewRecorder()
	handlers.Extract(rec, httptest.NewRequest(http.MethodPost, "/extract", body))

	var res ExtractResponse
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusNotFound || res.StatusCode != http.StatusNotFound || res.Error.Code != "ERR_UPSTREAM_STATUS" {
		t.Errorf("got status %d, body status %d, error %+v", rec.Code, res.StatusCode, res.Error)
	}
}
