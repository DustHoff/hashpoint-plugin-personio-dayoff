// Package personio kapselt den Zugriff auf die interne Personio-Frontend-API.
// Nicht die offizielle Public-API - es werden die Endpunkte verwendet, die
// das User-Frontend (app.personio.de) selbst aufruft. Session-Cookies und
// CSRF-Token werden vom Host bereitgestellt (sdk.HostAPI.RequestPersonioSession).
package personio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	sdk "github.com/dusthoff/hashpoint/plugin/sdk"
)

const (
	pathUpcomingTimeOff = "/platform/dashboard/api/v2/upcoming-time-off"
	defaultTimezone     = "Europe/Berlin"
	userAgent           = "hashpoint-personio-dayoff/0.2"
	defaultTimeout      = 30 * time.Second
	maxBodyBytes        = 1 << 20 // 1 MiB
)

// SourceUpcoming is the value placed in OffHoursInterval.Source for events
// originating from the upcoming-time-off dashboard endpoint.
const SourceUpcoming = "personio-upcoming"

// Client talks to the Personio internal UI API. It does not negotiate
// sessions — callers obtain one via HostAPI.RequestPersonioSession and
// pass it on every call.
type Client struct {
	http *http.Client
}

// NewClient returns a Client with a 30-second timeout against the
// process-default transport.
func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: defaultTimeout}}
}

// NewClientWithHTTPClient lets callers (tests, callers needing custom
// retry/proxy/cert behaviour) supply their own *http.Client. The caller
// owns the timeout and transport.
func NewClientWithHTTPClient(httpClient *http.Client) *Client {
	return &Client{http: httpClient}
}

// UpcomingTimeOff fetches up to ~5 next absence events for the signed-in
// employee from the dashboard widget endpoint. The endpoint mixes
// employer-defined holidays (status == "") with regular absences
// (status == "approved", "pending", …); both are returned and the caller
// decides how to treat each kind.
//
// On non-2xx, returns *HTTPError so callers can branch on Status (e.g.
// retry once on 401/403 with a freshly captured session).
func (c *Client) UpcomingTimeOff(ctx context.Context, sess sdk.PersonioSession) ([]TimeOffEvent, error) {
	if sess.AppHost == "" {
		return nil, errors.New("personio: session has no AppHost")
	}

	u := url.URL{Scheme: "https", Host: sess.AppHost, Path: pathUpcomingTimeOff}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("personio: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", userAgent)
	if sess.CSRFToken != "" {
		req.Header.Set("x-athena-xsrf-token", sess.CSRFToken)
	}
	req.Header.Set("timezone", defaultTimezone)
	for _, ck := range sess.Cookies {
		// Outbound request cookies serialise as plain Name=Value pairs in
		// the Cookie header — Secure/HttpOnly/SameSite are browser-enforced
		// response-side attributes and have no effect here.
		//nolint:gosec // G124: outbound request cookie, security flags not applicable
		req.AddCookie(&http.Cookie{Name: ck.Name, Value: ck.Value})
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("personio: do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("personio: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPError{
			Method:  req.Method,
			URL:     req.URL.String(),
			Status:  resp.StatusCode,
			Snippet: snippet(body),
		}
	}

	return parseUpcomingResponse(body)
}

func snippet(body []byte) string {
	const limit = 256
	if len(body) <= limit {
		return string(body)
	}
	return string(body[:limit]) + "…"
}
