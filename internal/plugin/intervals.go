package plugin

import (
	"strings"
	"time"

	_ "time/tzdata"

	sdk "github.com/dusthoff/hashpoint/plugin/sdk"

	"github.com/DustHoff/hashpoint-plugin-personio-dayoff/internal/personio"
)

// displayTimezone is the timezone used to format interval boundaries for the
// success log. Personio's UI API delivers events in Europe/Berlin and our
// parser stores them as UTC — rendering back in Berlin keeps the log entries
// readable for European users.
const displayTimezone = "Europe/Berlin"

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

// formatIntervalsForLog renders intervals as a compact, single-line summary
// fit for a host.Log() field. Single-day intervals collapse to one date;
// multi-day intervals render as `start..end (Reason)`. Boundaries are
// presented in Europe/Berlin (Personio's source timezone), and the
// half-open End is converted to an inclusive last calendar day by
// subtracting one nanosecond before formatting — so an Urlaub running
// 17.–26. July does not display as 17.–27. July.
//
// Empty input produces an empty string; callers should gate the log call
// on len(intervals) > 0 to avoid noise.
func formatIntervalsForLog(intervals []sdk.OffHoursInterval) string {
	if len(intervals) == 0 {
		return ""
	}
	loc, err := time.LoadLocation(displayTimezone)
	if err != nil {
		loc = time.UTC
	}
	parts := make([]string, 0, len(intervals))
	for _, iv := range intervals {
		startLocal := iv.Start.In(loc).Format("2006-01-02")
		endLocal := iv.End.Add(-time.Nanosecond).In(loc).Format("2006-01-02")
		var dateRange string
		if startLocal == endLocal {
			dateRange = startLocal
		} else {
			dateRange = startLocal + ".." + endLocal
		}
		parts = append(parts, dateRange+" ("+iv.Reason+")")
	}
	return strings.Join(parts, "; ")
}
