package personio

import (
	"encoding/json"
	"fmt"
	"time"

	_ "time/tzdata"
)

// TimeOffEvent is the domain type produced from one entry in the
// upcoming-time-off response. Times are UTC; the range is half-open
// [Start, End).
type TimeOffEvent struct {
	AbsenceType string
	Start       time.Time
	End         time.Time
	Status      string // empty for employer-defined holidays
	EmployeeID  int64
	FullDay     bool
	IsHoliday   bool // derived: status == ""
}

type apiResponse struct {
	TimeOff []apiTimeOff `json:"timeOff"`
}

type apiTimeOff struct {
	Status              *string `json:"status"`
	StartTime           string  `json:"start_time"`
	EndTime             string  `json:"end_time"`
	Timezone            string  `json:"timezone"`
	IsFullDay           bool    `json:"is_full_day_absence"`
	AbsenceType         string  `json:"absence_type"`
	EmployeeID          int64   `json:"employeeId"`
	AbsenceTypeLegacyID int64   `json:"absence_type_legacy_id"`
	StartDateTime       *string `json:"startDateTime"`
	EndDateTime         *string `json:"endDateTime"`
}

func parseUpcomingResponse(body []byte) ([]TimeOffEvent, error) {
	var r apiResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("personio: parse response: %w", err)
	}
	out := make([]TimeOffEvent, 0, len(r.TimeOff))
	for i, raw := range r.TimeOff {
		ev, err := mapEvent(raw)
		if err != nil {
			return nil, fmt.Errorf("personio: event %d (%s): %w", i, raw.AbsenceType, err)
		}
		out = append(out, ev)
	}
	return out, nil
}

func mapEvent(t apiTimeOff) (TimeOffEvent, error) {
	loc, err := resolveLocation(t.Timezone)
	if err != nil {
		return TimeOffEvent{}, err
	}

	var (
		start, end time.Time
	)
	switch {
	case t.IsFullDay:
		start, end, err = fullDayBounds(t.StartTime, t.EndTime, loc)
	case t.StartDateTime != nil && t.EndDateTime != nil:
		start, end, err = parseRFC3339Bounds(*t.StartDateTime, *t.EndDateTime)
	default:
		start, end, err = fullDayBounds(t.StartTime, t.EndTime, loc)
	}
	if err != nil {
		return TimeOffEvent{}, err
	}

	status := ""
	if t.Status != nil {
		status = *t.Status
	}

	return TimeOffEvent{
		AbsenceType: t.AbsenceType,
		Start:       start,
		End:         end,
		Status:      status,
		EmployeeID:  t.EmployeeID,
		FullDay:     t.IsFullDay,
		IsHoliday:   status == "",
	}, nil
}

// resolveLocation maps the Personio timezone field to a *time.Location.
// "Z" and empty string default to Europe/Berlin — Personio sends "Z" for
// employer-defined holidays even though those entries are obviously local
// dates, not UTC.
func resolveLocation(tz string) (*time.Location, error) {
	if tz == "" || tz == "Z" {
		tz = defaultTimezone
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		// Fall back to default so an unknown tz string from Personio doesn't
		// poison the whole response.
		fallback, ferr := time.LoadLocation(defaultTimezone)
		if ferr != nil {
			return nil, fmt.Errorf("load timezone %q: %w", tz, err)
		}
		return fallback, nil
	}
	return loc, nil
}

// fullDayBounds parses YYYY-MM-DD start/end dates and returns half-open
// [Start, End) UTC times: Start at 00:00 local on startStr, End at
// 00:00 local on (endStr + 1 day).
func fullDayBounds(startStr, endStr string, loc *time.Location) (time.Time, time.Time, error) {
	sd, err := time.ParseInLocation("2006-01-02", startStr, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse start_time: %w", err)
	}
	ed, err := time.ParseInLocation("2006-01-02", endStr, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse end_time: %w", err)
	}
	start := time.Date(sd.Year(), sd.Month(), sd.Day(), 0, 0, 0, 0, loc).UTC()
	end := time.Date(ed.Year(), ed.Month(), ed.Day()+1, 0, 0, 0, 0, loc).UTC()
	return start, end, nil
}

// parseRFC3339Bounds parses ISO timestamps from startDateTime/endDateTime
// and converts to UTC. The half-open semantic is preserved by trusting the
// API's bounds verbatim.
func parseRFC3339Bounds(startStr, endStr string) (time.Time, time.Time, error) {
	s, err := time.Parse(time.RFC3339Nano, startStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse startDateTime: %w", err)
	}
	e, err := time.Parse(time.RFC3339Nano, endStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse endDateTime: %w", err)
	}
	return s.UTC(), e.UTC(), nil
}
