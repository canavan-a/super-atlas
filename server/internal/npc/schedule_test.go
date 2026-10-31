package npc

import (
	"math"
	"testing"
	"time"
)

func TestEveryTripHasARoute(t *testing.T) {
	for _, trip := range Trips() {
		r, ok := routes[TripKey(trip[0], trip[1])]
		if !ok || len(r.Points) < 2 {
			t.Errorf("no route for %s; run go generate", TripKey(trip[0], trip[1]))
		}
	}
	for _, n := range cast {
		for _, key := range append([]string{n.Home, n.Work}, append(n.Lunch, n.Dinner...)...) {
			if _, ok := places[key]; !ok {
				t.Errorf("%s: unknown place %q", n.Email, key)
			}
		}
	}
}

// metersBetween is the rough distance between two [lng, lat] points.
func metersBetween(a, b [2]float64) float64 {
	const mPerDeg = 111_320
	dx := (a[0] - b[0]) * mPerDeg * math.Cos(a[1]*math.Pi/180)
	dy := (a[1] - b[1]) * mPerDeg
	return math.Hypot(dx, dy)
}

func TestDayIsContinuous(t *testing.T) {
	// Across a DST change too (Nov 1, 2026).
	for _, day := range []time.Time{
		time.Date(2026, 10, 30, 0, 0, 0, 0, albany),
		time.Date(2026, 11, 1, 0, 0, 0, 0, albany),
	} {
		for _, n := range cast {
			home := places[n.Home]
			at3am, driving := position(n, day.Add(3*time.Hour))
			if driving || metersBetween(at3am, [2]float64{home.Lng, home.Lat}) > 1 {
				t.Errorf("%s at 3am: %v (driving %v), want home", n.Email, at3am, driving)
			}
			if p, _ := position(n, day.Add(14*time.Hour)); metersBetween(p, at3am) < 100 {
				t.Errorf("%s still home at 2pm", n.Email)
			}
			prev, _ := position(n, day)
			drove := 0
			for s := 1; s < 26*3600; s++ {
				p, driving := position(n, day.Add(time.Duration(s)*time.Second))
				if driving {
					drove++
				}
				// Under 60 m a second: no jumps, and no faster than a highway.
				if d := metersBetween(prev, p); d > 60 {
					t.Fatalf("%s jumped %.0f m at %s", n.Email, d, day.Add(time.Duration(s)*time.Second).Format(time.Kitchen))
				}
				prev = p
			}
			if drove < 15*60 {
				t.Errorf("%s drove only %d s all day", n.Email, drove)
			}
		}
	}
}
