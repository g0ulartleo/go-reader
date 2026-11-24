package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
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
