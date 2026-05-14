package plugin

import (
	"testing"
	"time"

	sdk "github.com/dusthoff/hashpoint/plugin/sdk"

	"github.com/DustHoff/hashpoint-plugin-personio-dayoff/internal/personio"
)

func mkUTC(year int, month time.Month, day, hour int) time.Time {
	return time.Date(year, month, day, hour, 0, 0, 0, time.UTC)
}

func TestBuildIntervals_ClipsToWindow(t *testing.T) {
	events := []personio.TimeOffEvent{
		// fully inside window
		{
			AbsenceType: "Urlaub",
			Start:       mkUTC(2026, 5, 5, 0),
			End:         mkUTC(2026, 5, 10, 0),
		},
		// starts before window
		{
			AbsenceType: "Maifeiertag",
			Start:       mkUTC(2026, 4, 30, 22),
			End:         mkUTC(2026, 5, 1, 22),
		},
		// ends after window
		{
			AbsenceType: "Pfingstmontag",
			Start:       mkUTC(2026, 5, 24, 22),
			End:         mkUTC(2026, 5, 25, 22),
		},
		// fully outside (in the past)
		{
			AbsenceType: "Karfreitag",
			Start:       mkUTC(2026, 4, 3, 0),
			End:         mkUTC(2026, 4, 4, 0),
		},
	}
	req := sdk.OffHoursRequest{
		From: mkUTC(2026, 5, 1, 0),
		To:   mkUTC(2026, 5, 25, 0),
	}

	intervals, dropped := buildIntervals(events, nil, req)
	if dropped != 1 {
		t.Errorf("dropped = %d, want 1 (Karfreitag)", dropped)
	}
	if len(intervals) != 3 {
		t.Fatalf("len(intervals) = %d, want 3", len(intervals))
	}

	// Urlaub: untouched
	if !intervals[0].Start.Equal(mkUTC(2026, 5, 5, 0)) || !intervals[0].End.Equal(mkUTC(2026, 5, 10, 0)) {
		t.Errorf("Urlaub interval = [%s,%s)", intervals[0].Start, intervals[0].End)
	}
	// Maifeiertag: start clipped to window.From
	if !intervals[1].Start.Equal(req.From) {
		t.Errorf("Maifeiertag Start = %s, want %s (clipped)", intervals[1].Start, req.From)
	}
	if !intervals[1].End.Equal(mkUTC(2026, 5, 1, 22)) {
		t.Errorf("Maifeiertag End = %s", intervals[1].End)
	}
	// Pfingstmontag: end clipped to window.To
	if !intervals[2].End.Equal(req.To) {
		t.Errorf("Pfingstmontag End = %s, want %s (clipped)", intervals[2].End, req.To)
	}

	// Source + Kind are stamped uniformly
	for i, iv := range intervals {
		if iv.Kind != sdk.OffHoursAdd {
			t.Errorf("intervals[%d].Kind = %q, want %q", i, iv.Kind, sdk.OffHoursAdd)
		}
		if iv.Source != personio.SourceUpcoming {
			t.Errorf("intervals[%d].Source = %q, want %q", i, iv.Source, personio.SourceUpcoming)
		}
	}
}

func TestBuildIntervals_FilterAllowlist(t *testing.T) {
	events := []personio.TimeOffEvent{
		{AbsenceType: "Urlaub", Start: mkUTC(2026, 6, 1, 0), End: mkUTC(2026, 6, 2, 0)},
		{AbsenceType: "Maifeiertag", Start: mkUTC(2026, 5, 1, 0), End: mkUTC(2026, 5, 2, 0)},
		{AbsenceType: "Krankheit", Start: mkUTC(2026, 6, 5, 0), End: mkUTC(2026, 6, 6, 0)},
	}
	req := sdk.OffHoursRequest{
		From: mkUTC(2026, 1, 1, 0),
		To:   mkUTC(2026, 12, 31, 0),
	}

	intervals, _ := buildIntervals(events, []string{"urlaub", "Krankheit"}, req)
	if len(intervals) != 2 {
		t.Fatalf("len(intervals) = %d, want 2", len(intervals))
	}
	gotTypes := map[string]bool{}
	for _, iv := range intervals {
		gotTypes[iv.Reason] = true
	}
	if !gotTypes["Urlaub"] || !gotTypes["Krankheit"] {
		t.Errorf("expected Urlaub+Krankheit, got %+v", gotTypes)
	}
	if gotTypes["Maifeiertag"] {
		t.Errorf("Maifeiertag should be filtered out")
	}
}

func TestBuildIntervals_EmptyFilter_KeepsEverything(t *testing.T) {
	events := []personio.TimeOffEvent{
		{AbsenceType: "Urlaub", Start: mkUTC(2026, 6, 1, 0), End: mkUTC(2026, 6, 2, 0)},
		{AbsenceType: "Maifeiertag", Start: mkUTC(2026, 5, 1, 0), End: mkUTC(2026, 5, 2, 0)},
	}
	req := sdk.OffHoursRequest{From: mkUTC(2026, 1, 1, 0), To: mkUTC(2026, 12, 31, 0)}

	intervals, _ := buildIntervals(events, nil, req)
	if len(intervals) != 2 {
		t.Errorf("len(intervals) = %d, want 2", len(intervals))
	}
}

func TestClip(t *testing.T) {
	from := mkUTC(2026, 5, 1, 0)
	to := mkUTC(2026, 5, 31, 0)

	cases := []struct {
		name               string
		start, end         time.Time
		wantStart, wantEnd time.Time
		wantOK             bool
	}{
		{"fully inside", mkUTC(2026, 5, 10, 0), mkUTC(2026, 5, 12, 0), mkUTC(2026, 5, 10, 0), mkUTC(2026, 5, 12, 0), true},
		{"clip start", mkUTC(2026, 4, 30, 0), mkUTC(2026, 5, 5, 0), from, mkUTC(2026, 5, 5, 0), true},
		{"clip end", mkUTC(2026, 5, 25, 0), mkUTC(2026, 6, 10, 0), mkUTC(2026, 5, 25, 0), to, true},
		{"fully past", mkUTC(2026, 4, 1, 0), mkUTC(2026, 4, 10, 0), from, mkUTC(2026, 4, 10, 0), false},
		{"fully future", mkUTC(2026, 6, 1, 0), mkUTC(2026, 6, 10, 0), mkUTC(2026, 6, 1, 0), to, false},
		{"touching from (end == from)", mkUTC(2026, 4, 25, 0), from, from, from, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotStart, gotEnd, ok := clip(tc.start, tc.end, from, to)
			if ok != tc.wantOK {
				t.Errorf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && (!gotStart.Equal(tc.wantStart) || !gotEnd.Equal(tc.wantEnd)) {
				t.Errorf("got [%s, %s), want [%s, %s)", gotStart, gotEnd, tc.wantStart, tc.wantEnd)
			}
		})
	}
}

func TestParseFilter(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{"Urlaub", []string{"Urlaub"}},
		{"Urlaub, Krankheit", []string{"Urlaub", "Krankheit"}},
		{"  Urlaub  ,  ,Krankheit ", []string{"Urlaub", "Krankheit"}},
		{",,,", nil},
	}
	for _, tc := range cases {
		got := parseFilter(tc.in)
		if !equalSlice(got, tc.want) {
			t.Errorf("parseFilter(%q) = %#v, want %#v", tc.in, got, tc.want)
		}
	}
}

func equalSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
