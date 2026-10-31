package npc

import (
	"hash/fnv"
	"math/rand/v2"
	"sync"
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

const (
	// A road trip starts on a given day with this chance on Friday, Saturday
	// or Sunday and a third of it on other days: 4×p/3 + 3×p a week, about
	// one trip every 3 months on average, at no fixed interval.
	weekendTripChance = 0.0185
	// A trip lasts 1 to maxTripNights nights away.
	maxTripNights = 3
)

// leg is one stretch of an NPC's day: parked at a place, or driving a route.
type leg struct {
	start, end time.Time
	place      string // while parked
	route      *Route // while driving
}

// dayRNG is the random source for one thing about n's day: the same every
// time, so plans need no storing.
func dayRNG(n NPC, day time.Time, what string) *rand.Rand {
	h := fnv.New64a()
	h.Write([]byte(n.Email + day.Format("2006-01-02") + what))
	return rand.New(rand.NewPCG(h.Sum64(), 0))
}

// addDays is midnight (in Albany) k days after the midnight day.
func addDays(day time.Time, k int) time.Time {
	y, m, d := day.Date()
	return time.Date(y, m, d+k, 0, 0, 0, 0, albany)
}

// trip is a road trip: to destinations[dest], leaving on start and coming
// back nights days later.
type trip struct {
	start  time.Time
	dest   int
	nights int
}

// tripFrom is the trip n leaves for on day, if they do. A day only starts a
// trip if none of the maxTripNights days before drew one, so trips never
// overlap.
func tripFrom(n NPC, day time.Time) *trip {
	draw := func(day time.Time) (bool, *rand.Rand) {
		rng := dayRNG(n, day, "trip")
		chance := weekendTripChance
		if wd := day.Weekday(); wd != time.Friday && wd != time.Saturday && wd != time.Sunday {
			chance /= 3
		}
		return rng.Float64() < chance, rng
	}
	ok, rng := draw(day)
	if !ok {
		return nil
	}
	for k := 1; k <= maxTripNights; k++ {
		if earlier, _ := draw(addDays(day, -k)); earlier {
			return nil
		}
	}
	return &trip{start: day, dest: rng.IntN(len(destinations)), nights: 1 + rng.IntN(maxTripNights)}
}

// tripOn is the trip n is on at some point during day (leaving, away or
// coming back), or nil.
func tripOn(n NPC, day time.Time) *trip {
	for k := 0; k <= maxTripNights; k++ {
		if t := tripFrom(n, addDays(day, -k)); t != nil && k <= t.nights {
			return t
		}
	}
	return nil
}

// anchor is where n is at the stroke of midnight starting day: a stop
// around Albany, or at their destination if away overnight.
func anchor(n NPC, day time.Time) string {
	stops := localStops
	if t := tripOn(n, day); t != nil && !day.Equal(t.start) {
		stops = destinations[t.dest]
	}
	return stops[dayRNG(n, day, "anchor").IntN(len(stops))]
}

// planner lays out legs one after another from now.
type planner struct {
	legs []leg
	now  time.Time
	at   string
	rng  *rand.Rand
}

func (p *planner) park(until time.Time) {
	if until.After(p.now) {
		p.legs = append(p.legs, leg{start: p.now, end: until, place: p.at})
		p.now = until
	}
}

func (p *planner) drive(to string) {
	r := routes[TripKey(p.at, to)]
	end := p.now.Add(seconds(r.Duration()))
	p.legs = append(p.legs, leg{start: p.now, end: end, route: &r})
	p.now, p.at = end, to
}

// blip is how long a stop lasts: 1 to 5 minutes.
func (p *planner) blip() time.Duration {
	return time.Minute + time.Duration(p.rng.Int64N(int64(4*time.Minute)))
}

// wander drives from stop to stop, a short stop at each, going straight back
// to the one just left only if nowhere else will do, and so it ends at stop
// to by until: once no hop leaves time to get there after, it drives there
// the quickest way and waits out what's left.
func (p *planner) wander(until time.Time, to string) {
	prev := ""
	for {
		stop := p.blip()
		fits := func(next string) bool {
			arrive := p.now.Add(stop + seconds(routes[TripKey(p.at, next)].Duration()) + seconds(fastest[next][to].secs))
			return !arrive.After(until)
		}
		options := neighbors[p.at]
		next := ""
		for _, i := range p.rng.Perm(len(options)) {
			if o := options[i]; fits(o) && (next == "" || next == prev) {
				next = o
			}
		}
		if next == "" {
			break
		}
		p.park(p.now.Add(stop))
		prev = p.at
		p.drive(next)
	}
	for p.at != to {
		p.drive(fastest[p.at][to].next)
	}
	p.park(until)
}

// dayPlan lays out n's day (in Albany) containing t: driving almost all the
// time, from midnight's anchor to the next midnight's. A road trip day
// wanders around Albany until leaving from hub between 8 and noon, then
// around the destination; the day back wanders there until leaving between
// 10 and 2, then around Albany.
func dayPlan(n NPC, t time.Time) []leg {
	local := t.In(albany)
	y, m, d := local.Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, albany)
	if legs, ok := plans.get(n.Email, day); ok {
		return legs
	}
	next := addDays(day, 1)
	p := &planner{now: day, at: anchor(n, day), rng: dayRNG(n, day, "plan")}
	minutes := func(lo, hi int) time.Duration { return time.Duration(lo+p.rng.IntN(hi-lo+1)) * time.Minute }
	switch tr := tripOn(n, day); {
	case tr != nil && day.Equal(tr.start):
		p.wander(day.Add(8*time.Hour+minutes(0, 240)), hub)
		p.drive(destinations[tr.dest][0])
	case tr != nil && day.Equal(addDays(tr.start, tr.nights)):
		p.wander(day.Add(10*time.Hour+minutes(0, 240)), destinations[tr.dest][0])
		p.drive(hub)
	}
	p.wander(next, anchor(n, next))
	plans.put(n.Email, day, p.legs)
	return p.legs
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
	p := places[localStops[0]]
	return [2]float64{p.Lng, p.Lat}, false
}

func seconds(s float64) time.Duration { return time.Duration(s * float64(time.Second)) }

// hop is the quickest way from one stop to another along neighbors: how long
// it takes, and the stop to drive to first.
type hop struct {
	secs float64
	next string
}

// fastest holds the quickest way between every two stops that are linked at
// all (Floyd–Warshall over neighbors, timed by routes).
var fastest = func() map[string]map[string]hop {
	keys := sortedKeys(neighbors)
	out := make(map[string]map[string]hop, len(keys))
	for _, a := range keys {
		out[a] = map[string]hop{a: {0, a}}
		for _, b := range neighbors[a] {
			if r, ok := routes[TripKey(a, b)]; ok {
				out[a][b] = hop{r.Duration(), b}
			}
		}
	}
	for _, k := range keys {
		for _, a := range keys {
			ak, ok := out[a][k]
			if !ok {
				continue
			}
			for _, b := range keys {
				kb, ok := out[k][b]
				if !ok {
					continue
				}
				if ab, ok := out[a][b]; !ok || ak.secs+kb.secs < ab.secs {
					out[a][b] = hop{ak.secs + kb.secs, ak.next}
				}
			}
		}
	}
	return out
}()

// plans keeps recently worked-out days, since position is asked for every
// second and a day takes some working out.
var plans = &planCache{m: map[string][]leg{}}

type planCache struct {
	mu sync.Mutex
	m  map[string][]leg
}

func (c *planCache) get(email string, day time.Time) ([]leg, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	legs, ok := c.m[email+day.Format("2006-01-02")]
	return legs, ok
}

func (c *planCache) put(email string, day time.Time, legs []leg) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.m) >= 4*len(cast) {
		clear(c.m)
	}
	c.m[email+day.Format("2006-01-02")] = legs
}

// speedAt is how fast n is going at t, in meters a second: how far they get
// in the next second.
func speedAt(n NPC, t time.Time) float64 {
	a, _ := position(n, t)
	b, _ := position(n, t.Add(time.Second))
	return metersBetween(Place{Lng: a[0], Lat: a[1]}, Place{Lng: b[0], Lat: b[1]})
}
