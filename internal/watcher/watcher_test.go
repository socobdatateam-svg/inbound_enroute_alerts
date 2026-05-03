package watcher

import (
	"testing"
	"time"
)

func TestNextScheduledSendSameDay(t *testing.T) {
	location := time.FixedZone("PHT", 8*60*60)
	now := time.Date(2026, time.April, 26, 17, 47, 0, 0, location)

	got := nextScheduledSend(now, scheduledSendHours)
	want := time.Date(2026, time.April, 26, 18, 0, 0, 0, location)
	if !got.Equal(want) {
		t.Fatalf("next scheduled send = %s, want %s", got, want)
	}
}

func TestNextScheduledSendNextDay(t *testing.T) {
	location := time.FixedZone("PHT", 8*60*60)
	now := time.Date(2026, time.April, 26, 21, 0, 0, 0, location)

	got := nextScheduledSend(now, scheduledSendHours)
	want := time.Date(2026, time.April, 27, 0, 0, 0, 0, location)
	if !got.Equal(want) {
		t.Fatalf("next scheduled send = %s, want %s", got, want)
	}
}

func TestNextScheduledSendEarlyMorning(t *testing.T) {
	location := time.FixedZone("PHT", 8*60*60)
	now := time.Date(2026, time.April, 26, 0, 1, 0, 0, location)

	got := nextScheduledSend(now, scheduledSendHours)
	want := time.Date(2026, time.April, 26, 4, 0, 0, 0, location)
	if !got.Equal(want) {
		t.Fatalf("next scheduled send = %s, want %s", got, want)
	}
}

func TestScheduledSendHours(t *testing.T) {
	want := []int{0, 4, 6, 10, 13, 15, 18, 21}
	if len(scheduledSendHours) != len(want) {
		t.Fatalf("scheduled hours = %v, want %v", scheduledSendHours, want)
	}
	for i := range want {
		if scheduledSendHours[i] != want[i] {
			t.Fatalf("scheduled hours = %v, want %v", scheduledSendHours, want)
		}
	}
}

func TestFormatLinehaulAlert(t *testing.T) {
	now := time.Date(2026, time.May, 1, 9, 53, 0, 0, time.FixedZone("PHT", 8*60*60))

	got := formatLinehaulAlert("2 hours", now, "v3 data", "v4 data", "v5 data")
	want := "<mention-tag target=\"seatalk://user?id=0\"/> IB Expected Linehauls to Arrive within 2 hours including Late Units as of 9:53AM Update.\n\nv3 data\nv4 data\nv5 data"
	if got != want {
		t.Fatalf("alert text = %q, want %q", got, want)
	}
}

func TestFormatDailyUpdateAlert(t *testing.T) {
	now := time.Date(2026, time.May, 1, 9, 53, 0, 0, time.FixedZone("PHT", 8*60*60))

	got := formatDailyUpdateAlert(now)
	want := "<mention-tag target=\"seatalk://user?id=0\"/> En Route, Docked & On-Queue Update as of 9:53AM"
	if got != want {
		t.Fatalf("alert text = %q, want %q", got, want)
	}
}
