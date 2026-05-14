package plugin

import (
	"strings"
	"time"

	sdk "github.com/dusthoff/hashpoint/plugin/sdk"

	"github.com/DustHoff/hashpoint-plugin-personio-dayoff/internal/personio"
)

// buildIntervals maps personio.TimeOffEvent values to sdk.OffHoursInterval,
// applying the user-configured absence-type filter and clipping each event
// hard to the requested half-open window [req.From, req.To). The second
// return value reports how many events were dropped because they fell
// entirely outside the window — useful for debug logging.
func buildIntervals(events []personio.TimeOffEvent, filter []string, req sdk.OffHoursRequest) ([]sdk.OffHoursInterval, int) {
	intervals := make([]sdk.OffHoursInterval, 0, len(events))
	dropped := 0
	for _, e := range events {
		if !matchesFilter(e.AbsenceType, filter) {
			continue
		}
		start, end, ok := clip(e.Start, e.End, req.From, req.To)
		if !ok {
			dropped++
			continue
		}
		intervals = append(intervals, sdk.OffHoursInterval{
			Start:  start,
			End:    end,
			Kind:   sdk.OffHoursAdd,
			Reason: e.AbsenceType,
			Source: personio.SourceUpcoming,
		})
	}
	return intervals, dropped
}

// clip returns the intersection of [start, end) and [from, to). ok is false
// when the result would be empty.
func clip(start, end, from, to time.Time) (time.Time, time.Time, bool) {
	if start.Before(from) {
		start = from
	}
	if end.After(to) {
		end = to
	}
	return start, end, start.Before(end)
}

// matchesFilter returns true when absenceType is allowed by filter. An empty
// filter is a no-op (every type allowed). Comparison is case-insensitive
// so users can write "Urlaub" or "urlaub" interchangeably.
func matchesFilter(absenceType string, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, allowed := range filter {
		if strings.EqualFold(allowed, absenceType) {
			return true
		}
	}
	return false
}
