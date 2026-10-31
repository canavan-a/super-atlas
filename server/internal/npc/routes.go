package npc

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
)

//go:generate go run ./gen

// Route is one drive: its path as [lng, lat] points and, for each point, the
// seconds since leaving at which the car passes it.
type Route struct {
	Points [][2]float64 `json:"p"`
	Secs   []float64    `json:"s"`
}

// Duration is how long the drive takes, in seconds.
func (r Route) Duration() float64 { return r.Secs[len(r.Secs)-1] }

// At returns the car's [lng, lat] secs into the drive.
func (r Route) At(secs float64) [2]float64 {
	if secs <= 0 {
		return r.Points[0]
	}
	i := sort.SearchFloat64s(r.Secs, secs)
	if i >= len(r.Secs) {
		return r.Points[len(r.Points)-1]
	}
	if i == 0 || r.Secs[i] == r.Secs[i-1] {
		return r.Points[i]
	}
	t := (secs - r.Secs[i-1]) / (r.Secs[i] - r.Secs[i-1])
	a, b := r.Points[i-1], r.Points[i]
	return [2]float64{a[0] + (b[0]-a[0])*t, a[1] + (b[1]-a[1])*t}
}

// TripKey names the drive from one place to another in routes.json.
func TripKey(from, to string) string { return from + ">" + to }

//go:embed routes.json
var routesJSON []byte

// routes holds every trip in Trips, from routes.json.
var routes = mustLoadRoutes()

func mustLoadRoutes() map[string]Route {
	var out map[string]Route
	if err := json.Unmarshal(routesJSON, &out); err != nil {
		panic(fmt.Sprintf("npc: routes.json: %v", err))
	}
	return out
}
