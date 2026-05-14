// Package personio kapselt den Zugriff auf die interne Personio-Frontend-API.
// Nicht die offizielle Public-API - es werden die Endpunkte verwendet, die
// das User-Frontend (app.personio.de) selbst aufruft. Session-Cookies und
// CSRF-Token werden vom Host bereitgestellt (sdk.HostAPI.RequestPersonioSession).
package personio

import (
	"context"
	"errors"
	"net/http"
	"time"

	sdk "github.com/dusthoff/hashpoint/plugin/sdk"
)

type Absence struct {
	ID    string
	Type  string
	Start time.Time
	End   time.Time
}

type Client struct {
	baseURL string
	host    sdk.HostAPI
	http    *http.Client
}

func NewClient(baseURL string, host sdk.HostAPI) *Client {
	return &Client{
		baseURL: baseURL,
		host:    host,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Absences ruft die Abwesenheiten des angemeldeten Users im angegebenen
// Zeitraum ab. Der Aufruf erfolgt gegen den internen Frontend-Endpoint;
// Auth-Daten kommen ueber HostAPI.RequestPersonioSession.
//
// TODO: Endpoint, Request-/Response-Schema und Pagination implementieren.
func (c *Client) Absences(_ context.Context, _ time.Time, _ time.Time) ([]Absence, error) {
	return nil, errors.New("personio.Client.Absences: not implemented yet")
}
