package schedule

import (
	"testing"
	"time"
)

type weekdays struct{}

func (weekdays) IsWorkday(d time.Time) bool {
	return d.Weekday() != time.Saturday && d.Weekday() != time.Sunday
}

func day(s string) time.Time { v, _ := time.Parse(time.DateOnly, s); return v }

func TestResolve(t *testing.T) {
	office := Shift{Name: "Jam kantor", Start: "09:00", End: "18:00"}
	night := &Shift{ID: 2, Name: "Malam", Start: "22:00", End: "06:00", Grace: 10}
	// Monday 2026-03-02.
	if d := Resolve(day("2026-03-02"), weekdays{}, nil, nil, office); d.Off || d.Start.Hour() != 9 || d.ShiftName != "Jam kantor" {
		t.Fatalf("office hours: %+v", d)
	}
	d := Resolve(day("2026-03-02"), weekdays{}, nil, night, office)
	if d.End.Sub(d.Start) != 8*time.Hour || d.End.Day() != 3 || *d.ShiftID != 2 {
		t.Fatalf("overnight default shift: %+v", d)
	}
	if !Resolve(day("2026-03-07"), weekdays{}, nil, night, office).Off {
		t.Fatal("default shift must not apply on a weekend")
	}
	if d := Resolve(day("2026-03-07"), weekdays{}, &Assignment{Shift: night}, nil, office); d.Off || !d.Assigned {
		t.Fatal("assignment must apply on a weekend")
	}
	if !Resolve(day("2026-03-02"), weekdays{}, &Assignment{}, night, office).Off {
		t.Fatal("assigned day off must win")
	}
}

func TestLatenessAndDistance(t *testing.T) {
	d := Resolve(day("2026-03-02"), weekdays{}, nil, &Shift{ID: 1, Name: "Pagi", Start: "08:00", End: "17:00", Grace: 10}, Shift{})
	at := func(s string) time.Time { v, _ := time.ParseInLocation("2006-01-02 15:04", s, WIB); return v }
	if LateMinutes(d, at("2026-03-02 08:10")) != 0 || LateMinutes(d, at("2026-03-02 08:25")) != 15 {
		t.Fatal("late minutes must start after grace")
	}
	if EarlyMinutes(d, at("2026-03-02 16:30")) != 30 || EarlyMinutes(d, at("2026-03-02 17:05")) != 0 {
		t.Fatal("early leave")
	}
	// Monas to Bundaran HI is roughly 2.2 km.
	if m := DistanceMeters(-6.1754, 106.8272, -6.1950, 106.8230); m < 2000 || m > 2400 {
		t.Fatalf("distance %f", m)
	}
}
