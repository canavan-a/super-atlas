package npc

import (
	"hash/fnv"
	"math/rand/v2"
	"time"
	_ "time/tzdata" // Albany's time zone, wherever the server runs
)

// albany is the time zone NPC days run on.
var albany = mustLoadLocation("America/New_York")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// leg is one stretch of an NPC's day: parked at a place, or driving a route.
type leg struct {
	start, end time.Time
	place      string // while parked
	route      *Route // while driving
}

// dayPlan lays out n's day (in Albany) containing t. It is the same every
// time it is worked out for that day, so nothing needs storing: it is drawn
// from a random source seeded by n's email and the date. Every day goes:
//
//   - leave home between 7:00 and 8:30 and drive to work;
//   - leave for lunch between 11:45 and 12:45, eat for 30–60 min, drive back;
//   - leave work between 16:45 and 18:15 for dinner, eat for 1–2 hours;
//   - drive home and stay there until morning.
func dayPlan(n NPC, t time.Time) []leg {
	local := t.In(albany)
	y, m, d := local.Date()
	at := func(hour, min int) time.Time { return time.Date(y, m, d, hour, min, 0, 0, albany) }
	h := fnv.New64a()
	h.Write([]byte(n.Email + local.Format("2006-01-02")))
	rng := rand.New(rand.NewPCG(h.Sum64(), 0))
	minutes := func(lo, hi int) time.Duration { return time.Duration(lo+rng.IntN(hi-lo+1)) * time.Minute }
	pick := func(keys []string) string { return keys[rng.IntN(len(keys))] }

	var legs []leg
	now := at(0, 0)
	park := func(place string, until time.Time) {
		if until.After(now) {
			legs = append(legs, leg{start: now, end: until, place: place})
			now = until
		}
	}
	drive := func(from, to string) {
		r := routes[TripKey(from, to)]
		end := now.Add(time.Duration(r.Duration() * float64(time.Second)))
		legs = append(legs, leg{start: now, end: end, route: &r})
		now = end
	}

	park(n.Home, at(7, 0).Add(minutes(0, 90)))
	drive(n.Home, n.Work)
	lunch := pick(n.Lunch)
	park(n.Work, at(11, 45).Add(minutes(0, 60)))
	drive(n.Work, lunch)
	park(lunch, now.Add(minutes(30, 60)))
	drive(lunch, n.Work)
	dinner := pick(n.Dinner)
	park(n.Work, at(16, 45).Add(minutes(0, 90)))
	drive(n.Work, dinner)
	park(dinner, now.Add(minutes(60, 120)))
	drive(dinner, n.Home)
	park(n.Home, time.Date(y, m, d+1, 0, 0, 0, 0, albany))
	return legs
}

// position is where n is at t, as [lng, lat], and whether they're driving.
func position(n NPC, t time.Time) (lngLat [2]float64, driving bool) {
	for _, l := range dayPlan(n, t) {
		if t.Before(l.start) || !t.Before(l.end) {
			continue
		}
		if l.route != nil {
			return l.route.At(t.Sub(l.start).Seconds()), true
		}
		p := places[l.place]
		return [2]float64{p.Lng, p.Lat}, false
	}
	p := places[n.Home]
	return [2]float64{p.Lng, p.Lat}, false
}
