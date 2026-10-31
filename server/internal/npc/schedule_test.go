package npc

import (
	"testing"
	"time"
)

func TestEveryTripHasARoute(t *testing.T) {
	for _, trip := range Trips() {
		for _, key := range trip {
			if _, ok := places[key]; !ok {
				t.Errorf("unknown place %q", key)
			}
		}
		r, ok := routes[TripKey(trip[0], trip[1])]
		if !ok || len(r.Points) < 2 {
			t.Errorf("no route for %s; run go generate", TripKey(trip[0], trip[1]))
		}
	}
	// Wandering can always get anywhere else in the same area.
	for _, area := range append([][]string{localStops}, destinations...) {
		for _, a := range area {
			for _, b := range area {
				if _, ok := fastest[a][b]; !ok {
					t.Errorf("no way from %s to %s", a, b)
				}
			}
		}
	}
}

// distance is the rough distance in meters between two [lng, lat] points.
func distance(a, b [2]float64) float64 {
	return metersBetween(Place{Lng: a[0], Lat: a[1]}, Place{Lng: b[0], Lat: b[1]})
}

// checkMoving walks n through days from day a second at a time and fails on
// any jump, then returns how many of those seconds they spent driving and
// their longest stop.
func checkMoving(t *testing.T, n NPC, day time.Time, days int) (drove, secs int, longestStop time.Duration) {
	t.Helper()
	end := addDays(day, days)
	prev, _ := position(n, day)
	var stopped time.Duration
	for at := day.Add(time.Second); at.Before(end); at = at.Add(time.Second) {
		p, driving := position(n, at)
		secs++
		if driving {
			drove++
			stopped = 0
		} else {
			stopped += time.Second
			longestStop = max(longestStop, stopped)
		}
		// Under 60 m a second: no jumps, and no faster than a highway.
		if d := distance(prev, p); d > 60 {
			t.Fatalf("%s jumped %.0f m at %s", n.Email, d, at.Format(time.DateTime))
		}
		prev = p
	}
	return drove, secs, longestStop
}

func TestAlwaysOnTheMove(t *testing.T) {
	// Across a DST change too (Nov 1, 2026).
	day := time.Date(2026, 10, 30, 0, 0, 0, 0, albany)
	for _, n := range cast {
		drove, secs, longest := checkMoving(t, n, day, 4)
		t.Logf("%s: driving %.0f%% of the time, longest stop %s", n.Email, 100*float64(drove)/float64(secs), longest)
		if share := float64(drove) / float64(secs); share < 0.7 {
			t.Errorf("%s drove only %.0f%% of the time", n.Email, share*100)
		}
		if longest > 20*time.Minute {
			t.Errorf("%s stopped for %s", n.Email, longest)
		}
	}
}

func TestRoadTrips(t *testing.T) {
	const years = 5
	first := time.Date(2026, 1, 1, 0, 0, 0, 0, albany)
	trips, weekend := 0, 0
	var some *trip
	var someNPC NPC
	for _, n := range cast {
		for day := first; day.Before(addDays(first, 365*years)); day = addDays(day, 1) {
			tr := tripFrom(n, day)
			if tr == nil {
				continue
			}
			trips++
			if wd := day.Weekday(); wd == time.Friday || wd == time.Saturday || wd == time.Sunday {
				weekend++
			}
			if some == nil {
				some, someNPC = tr, n
			}
		}
	}
	perYear := float64(trips) / float64(len(cast)*years)
	if perYear < 2.5 || perYear > 6 {
		t.Errorf("%.1f trips a year each, want about 4", perYear)
	}
	if weekend*2 < trips {
		t.Errorf("only %d of %d trips leave Friday to Sunday", weekend, trips)
	}
	if some == nil {
		t.Fatal("no trips at all")
	}

	// A trip: away from Albany in the middle, home after, and smooth throughout.
	checkMoving(t, someNPC, addDays(some.start, -1), some.nights+3)
	h := places[hub]
	home := [2]float64{h.Lng, h.Lat}
	if p, _ := position(someNPC, addDays(some.start, 1).Add(12*time.Hour)); some.nights > 1 && distance(p, home) < 100_000 {
		t.Errorf("%s within 100 km of Albany on day 2 of a trip to %s", someNPC.Email, destinations[some.dest][0])
	}
	if p, _ := position(someNPC, addDays(some.start, some.nights+1).Add(12*time.Hour)); distance(p, home) > 40_000 {
		t.Errorf("%s not back from %s the day after", someNPC.Email, destinations[some.dest][0])
	}
}

func TestHighwaysAreFaster(t *testing.T) {
	// Highway stretches run well over 55 mph; city streets well under 35.
	var fast, slow bool
	for _, r := range routes {
		for i := 1; i < len(r.Points); i++ {
			dt := r.Secs[i] - r.Secs[i-1]
			d := distance(r.Points[i-1], r.Points[i])
			if dt <= 0 || d < 200 {
				continue
			}
			mph := d / dt * 2.237
			fast = fast || mph > 55
			slow = slow || mph < 35
		}
	}
	if !fast || !slow {
		t.Errorf("fast %v, slow %v: want both", fast, slow)
	}
}
