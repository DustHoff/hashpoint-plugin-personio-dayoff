package personio

import (
	_ "embed"
	"testing"
	"time"
)

//go:embed testdata/upcoming_time_off_har_april.json
var harApril []byte

func TestParseUpcomingResponse_HARSample(t *testing.T) {
	events, err := parseUpcomingResponse(harApril)
	if err != nil {
		t.Fatalf("parseUpcomingResponse: %v", err)
	}
	if len(events) != 5 {
		t.Fatalf("got %d events, want 5", len(events))
	}

	wantTypes := []string{"Maifeiertag", "Christi Himmelfahrt", "Day4Me", "Pfingstmontag", "Urlaub"}
	for i, want := range wantTypes {
		if events[i].AbsenceType != want {
			t.Errorf("event[%d].AbsenceType = %q, want %q", i, events[i].AbsenceType, want)
		}
	}

	// Holidays: status == "" → IsHoliday true
	for i := 0; i < 4; i++ {
		if !events[i].IsHoliday {
			t.Errorf("event[%d] (%s) IsHoliday = false, want true", i, events[i].AbsenceType)
		}
		if events[i].Status != "" {
			t.Errorf("event[%d] Status = %q, want empty", i, events[i].Status)
		}
	}

	// Vacation entry
	urlaub := events[4]
	if urlaub.IsHoliday {
		t.Errorf("Urlaub IsHoliday = true, want false")
	}
	if urlaub.Status != "approved" {
		t.Errorf("Urlaub.Status = %q, want approved", urlaub.Status)
	}
	if urlaub.EmployeeID != 10076878 {
		t.Errorf("Urlaub.EmployeeID = %d, want 10076878", urlaub.EmployeeID)
	}
}

func TestMapEvent_Holiday_TimezoneZ(t *testing.T) {
	// Maifeiertag in HAR: timezone="Z", is_full_day_absence=true.
	// Expected: treat as Europe/Berlin local day. 2026-05-01 is CEST (UTC+2).
	raw := apiTimeOff{
		Status:      nil,
		StartTime:   "2026-05-01",
		EndTime:     "2026-05-01",
		Timezone:    "Z",
		IsFullDay:   true,
		AbsenceType: "Maifeiertag",
		EmployeeID:  10076878,
	}
	got, err := mapEvent(raw)
	if err != nil {
		t.Fatalf("mapEvent: %v", err)
	}
	wantStart := time.Date(2026, 4, 30, 22, 0, 0, 0, time.UTC) // 2026-05-01 00:00 +02:00
	wantEnd := time.Date(2026, 5, 1, 22, 0, 0, 0, time.UTC)    // 2026-05-02 00:00 +02:00
	if !got.Start.Equal(wantStart) {
		t.Errorf("Start = %s, want %s", got.Start, wantStart)
	}
	if !got.End.Equal(wantEnd) {
		t.Errorf("End = %s, want %s", got.End, wantEnd)
	}
	if !got.IsHoliday {
		t.Errorf("IsHoliday = false, want true")
	}
	if got.Status != "" {
		t.Errorf("Status = %q, want empty", got.Status)
	}
}

func TestMapEvent_Vacation_MultiDay(t *testing.T) {
	// Urlaub 2026-07-17 → 2026-07-26 (CEST, UTC+2). End is exclusive: next
	// midnight after the last vacation day.
	status := "approved"
	sdt := "2026-07-17T00:00:00+02:00"
	edt := "2026-07-26T23:59:59.999999999+02:00"
	raw := apiTimeOff{
		Status:              &status,
		StartTime:           "2026-07-17",
		EndTime:             "2026-07-26",
		Timezone:            "Europe/Berlin",
		IsFullDay:           true,
		AbsenceType:         "Urlaub",
		EmployeeID:          10076878,
		AbsenceTypeLegacyID: 611738,
		StartDateTime:       &sdt,
		EndDateTime:         &edt,
	}
	got, err := mapEvent(raw)
	if err != nil {
		t.Fatalf("mapEvent: %v", err)
	}
	wantStart := time.Date(2026, 7, 16, 22, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 7, 26, 22, 0, 0, 0, time.UTC)
	if !got.Start.Equal(wantStart) {
		t.Errorf("Start = %s, want %s", got.Start, wantStart)
	}
	if !got.End.Equal(wantEnd) {
		t.Errorf("End = %s, want %s", got.End, wantEnd)
	}
	if got.IsHoliday {
		t.Errorf("IsHoliday = true, want false")
	}
}

func TestMapEvent_UnknownTimezone_FallsBackToDefault(t *testing.T) {
	raw := apiTimeOff{
		Status:      nil,
		StartTime:   "2026-05-01",
		EndTime:     "2026-05-01",
		Timezone:    "Mars/Olympus_Mons",
		IsFullDay:   true,
		AbsenceType: "Marstag",
	}
	got, err := mapEvent(raw)
	if err != nil {
		t.Fatalf("mapEvent: %v", err)
	}
	// Fallback is Europe/Berlin → CEST in May → start at 2026-04-30T22:00Z
	wantStart := time.Date(2026, 4, 30, 22, 0, 0, 0, time.UTC)
	if !got.Start.Equal(wantStart) {
		t.Errorf("Start = %s, want %s (fallback Europe/Berlin)", got.Start, wantStart)
	}
}

func TestResolveLocation_Defaults(t *testing.T) {
	cases := []struct{ in string }{{""}, {"Z"}}
	for _, c := range cases {
		loc, err := resolveLocation(c.in)
		if err != nil {
			t.Fatalf("resolveLocation(%q): %v", c.in, err)
		}
		if loc.String() != defaultTimezone {
			t.Errorf("resolveLocation(%q) = %s, want %s", c.in, loc, defaultTimezone)
		}
	}
}
