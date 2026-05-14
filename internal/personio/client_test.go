package personio_test

import (
	"context"
	_ "embed"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	sdk "github.com/dusthoff/hashpoint/plugin/sdk"

	"github.com/DustHoff/hashpoint-plugin-personio-dayoff/internal/personio"
)

//go:embed testdata/upcoming_time_off_har_april.json
var harAprilTestdata []byte

func TestClient_UpcomingTimeOff_HappyPath(t *testing.T) {
	var capturedReq *http.Request
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r.Clone(r.Context())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(harAprilTestdata)
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse srv url: %v", err)
	}

	c := personio.NewClientWithHTTPClient(srv.Client())
	//nolint:gosec // G101: test fixtures, not real credentials
	sess := sdk.PersonioSession{
		AppHost:   u.Host,
		CSRFToken: "test-xsrf-token",
		Cookies: []sdk.PersonioCookie{
			{Name: "PERSONIO_SESSION", Value: "sess-value"},
			{Name: "XSRF-TOKEN", Value: "xsrf-cookie-value"},
		},
	}

	events, err := c.UpcomingTimeOff(context.Background(), sess)
	if err != nil {
		t.Fatalf("UpcomingTimeOff: %v", err)
	}
	if len(events) != 5 {
		t.Fatalf("got %d events, want 5", len(events))
	}

	if capturedReq == nil {
		t.Fatal("no request captured")
	}
	if got, want := capturedReq.Method, http.MethodGet; got != want {
		t.Errorf("method = %s, want %s", got, want)
	}
	if got, want := capturedReq.URL.Path, "/platform/dashboard/api/v2/upcoming-time-off"; got != want {
		t.Errorf("path = %s, want %s", got, want)
	}
	if got, want := capturedReq.Header.Get("x-athena-xsrf-token"), "test-xsrf-token"; got != want {
		t.Errorf("CSRF header = %q, want %q", got, want)
	}
	if got, want := capturedReq.Header.Get("timezone"), "Europe/Berlin"; got != want {
		t.Errorf("timezone header = %q, want %q", got, want)
	}
	cookieNames := map[string]string{}
	for _, ck := range capturedReq.Cookies() {
		cookieNames[ck.Name] = ck.Value
	}
	if cookieNames["PERSONIO_SESSION"] != "sess-value" {
		t.Errorf("session cookie missing or wrong: %+v", cookieNames)
	}
	if cookieNames["XSRF-TOKEN"] != "xsrf-cookie-value" {
		t.Errorf("xsrf cookie missing or wrong: %+v", cookieNames)
	}
}

func TestClient_UpcomingTimeOff_401_ReturnsHTTPError(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	t.Cleanup(srv.Close)

	u, _ := url.Parse(srv.URL)
	c := personio.NewClientWithHTTPClient(srv.Client())
	sess := sdk.PersonioSession{AppHost: u.Host, CSRFToken: "x"}

	_, err := c.UpcomingTimeOff(context.Background(), sess)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var httpErr *personio.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("err = %v (%T), want *HTTPError", err, err)
	}
	if httpErr.Status != http.StatusUnauthorized {
		t.Errorf("Status = %d, want 401", httpErr.Status)
	}
	if httpErr.Snippet == "" {
		t.Errorf("Snippet is empty, want body excerpt")
	}
}

func TestClient_UpcomingTimeOff_NoAppHost(t *testing.T) {
	c := personio.NewClient()
	_, err := c.UpcomingTimeOff(context.Background(), sdk.PersonioSession{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
