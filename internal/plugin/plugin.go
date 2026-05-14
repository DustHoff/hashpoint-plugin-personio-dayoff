package plugin

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdk "github.com/dusthoff/hashpoint/plugin/sdk"

	"github.com/DustHoff/hashpoint-plugin-personio-dayoff/internal/personio"
)

const (
	pluginName    = "personio-dayoff"
	pluginVersion = "1.0.0"
)

type Plugin struct {
	host          sdk.HostAPI
	absenceFilter []string
	client        *personio.Client
}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Init(_ context.Context, host sdk.HostAPI) error {
	p.host = host
	return nil
}

func (p *Plugin) Metadata(_ context.Context) (sdk.Metadata, error) {
	return sdk.Metadata{
		Name:         pluginName,
		Version:      pluginVersion,
		APIVersion:   sdk.HostAPIVersion,
		Capabilities: []sdk.Capability{sdk.CapOffHoursProvider},
		Description:  "Liest Dayoff-Tage aus der Personio-Frontend-API.",
	}, nil
}

// Configure stores the per-plugin settings. The Personio AppHost is no
// longer a config field — it is delivered by the host via
// HostAPI.RequestPersonioSession at call time.
func (p *Plugin) Configure(_ context.Context, cfg sdk.PluginConfig) error {
	p.absenceFilter = parseFilter(cfg.Fields["absence_type_filter"])
	p.client = personio.NewClient()
	return nil
}

// OffHours fetches the next ~5 upcoming time-off events from Personio and
// returns them clipped to the requested window. Holidays (status == "")
// and regular absences are both reported; users can filter by absence_type
// name via the absence_type_filter setting.
func (p *Plugin) OffHours(ctx context.Context, req sdk.OffHoursRequest) ([]sdk.OffHoursInterval, error) {
	if p.client == nil {
		return nil, sdk.ErrNotConfigured
	}

	sess, err := p.host.RequestPersonioSession(ctx)
	if err != nil {
		if errors.Is(err, sdk.ErrPersonioNotAvailable) {
			_ = p.host.Log(ctx, "warn", "personio session not available, skipping", nil)
			return nil, nil
		}
		return nil, fmt.Errorf("request personio session: %w", err)
	}

	events, err := p.client.UpcomingTimeOff(ctx, sess)
	if err != nil {
		var httpErr *personio.HTTPError
		if !errors.As(err, &httpErr) || (httpErr.Status != 401 && httpErr.Status != 403) {
			return nil, fmt.Errorf("personio upcoming-time-off: %w", err)
		}
		// Session may have expired between host validation and our call.
		// Refresh once and retry; further failures bubble up.
		sess, err = p.host.RequestPersonioSession(ctx)
		if err != nil {
			if errors.Is(err, sdk.ErrPersonioNotAvailable) {
				_ = p.host.Log(ctx, "warn", "personio session unavailable after 401, skipping", nil)
				return nil, nil
			}
			return nil, fmt.Errorf("refresh personio session: %w", err)
		}
		events, err = p.client.UpcomingTimeOff(ctx, sess)
		if err != nil {
			return nil, fmt.Errorf("personio upcoming-time-off (after refresh): %w", err)
		}
	}

	intervals, dropped := buildIntervals(events, p.absenceFilter, req)
	if dropped > 0 {
		_ = p.host.Log(ctx, "debug", "events outside requested window dropped", map[string]string{
			"dropped":     strconv.Itoa(dropped),
			"window_from": req.From.Format(time.RFC3339),
			"window_to":   req.To.Format(time.RFC3339),
		})
	}
	return intervals, nil
}

func parseFilter(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, t := range parts {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}
