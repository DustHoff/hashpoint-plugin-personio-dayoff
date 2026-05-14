package plugin

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/dusthoff/hashpoint/plugin/sdk"

	"github.com/DustHoff/hashpoint-plugin-personio-dayoff/internal/personio"
)

const (
	pluginName    = "personio-dayoff"
	pluginVersion = "0.1.0"
)

type Plugin struct {
	host          sdk.HostAPI
	baseURL       string
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

func (p *Plugin) Configure(_ context.Context, cfg sdk.PluginConfig) error {
	p.baseURL = strings.TrimRight(cfg.Fields["base_url"], "/")
	if p.baseURL == "" {
		return sdk.ErrNotConfigured
	}

	if raw := cfg.Fields["absence_type_filter"]; raw != "" {
		for _, t := range strings.Split(raw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				p.absenceFilter = append(p.absenceFilter, t)
			}
		}
	}

	p.client = personio.NewClient(p.baseURL, p.host)
	return nil
}

func (p *Plugin) OffHours(ctx context.Context, req sdk.OffHoursRequest) ([]sdk.OffHoursInterval, error) {
	if p.client == nil {
		return nil, sdk.ErrNotConfigured
	}

	absences, err := p.client.Absences(ctx, req.From, req.To)
	if err != nil {
		return nil, fmt.Errorf("personio absences: %w", err)
	}

	intervals := make([]sdk.OffHoursInterval, 0, len(absences))
	for _, a := range absences {
		if !p.matchesFilter(a.Type) {
			continue
		}
		intervals = append(intervals, sdk.OffHoursInterval{
			Start:  a.Start,
			End:    a.End,
			Reason: a.Type,
		})
	}
	return intervals, nil
}

func (p *Plugin) matchesFilter(absenceType string) bool {
	if len(p.absenceFilter) == 0 {
		return true
	}
	for _, allowed := range p.absenceFilter {
		if strings.EqualFold(allowed, absenceType) {
			return true
		}
	}
	return false
}
