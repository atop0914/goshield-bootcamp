package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPHandler(t *testing.T) {
	r := NewRegistryWithNamespace("")
	r.Register(CollectorFunc(func() []MetricSample {
		return []MetricSample{
			{
				Name:  "test_metric",
				Value: 42,
				Type:  Counter,
				Help:  "A test metric.",
			},
		}
	}))

	handler := HTTPHandler(r)
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("expected text/plain content type, got %s", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "test_metric") {
		t.Errorf("expected test_metric in response body, got: %s", body)
	}
	if !strings.Contains(body, "# HELP test_metric A test metric.") {
		t.Errorf("expected HELP line in response body, got: %s", body)
	}
}

func TestHTTPHandlerEmpty(t *testing.T) {
	r := NewRegistry()
	handler := HTTPHandler(r)
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestHTTPHandlerCacheControl(t *testing.T) {
	r := NewRegistry()
	handler := HTTPHandler(r)
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	if !strings.Contains(cc, "no-cache") {
		t.Errorf("expected no-cache in Cache-Control, got %s", cc)
	}
}
