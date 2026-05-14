package personio

import "fmt"

// HTTPError represents a non-2xx response from the Personio frontend API.
// Callers can branch on Status (e.g. retry once on 401/403 with a freshly
// captured session).
type HTTPError struct {
	Method  string
	URL     string
	Status  int
	Snippet string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("personio: %s %s: status %d: %s", e.Method, e.URL, e.Status, e.Snippet)
}
