// Command gen fetches the driving route for every NPC trip (npc.Trips) from
// the public OSRM server and writes them to routes.json, which the npc
// package embeds. OSRM's car profile times each stretch of road by its speed
// limit, which is how NPCs drive. Run it from internal/npc with
// `go generate` after changing places or the cast.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/canavan-a/super-atlas/server/internal/npc"
)

const parkSecs = 20

const osrm = "https://router.project-osrm.org/route/v1/driving/%f,%f;%f,%f?overview=full&geometries=geojson&annotations=duration"

type osrmResponse struct {
	Code   string `json:"code"`
	Routes []struct {
		Geometry struct {
			Coordinates [][2]float64 `json:"coordinates"`
		} `json:"geometry"`
		Legs []struct {
			Annotation struct {
				Duration []float64 `json:"duration"`
			} `json:"annotation"`
		} `json:"legs"`
	} `json:"routes"`
}

func main() {
	routes := map[string]npc.Route{}
	client := &http.Client{Timeout: 30 * time.Second}
	for i, trip := range npc.Trips() {
		from, to := npc.PlaceAt(trip[0]), npc.PlaceAt(trip[1])
		resp, err := client.Get(fmt.Sprintf(osrm, from.Lng, from.Lat, to.Lng, to.Lat))
		if err != nil {
			log.Fatalf("%s: %v", npc.TripKey(trip[0], trip[1]), err)
		}
		var body osrmResponse
		err = json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if err != nil || body.Code != "Ok" || len(body.Routes) == 0 || len(body.Routes[0].Legs) == 0 {
			log.Fatalf("%s: bad response (%v, code %q)", npc.TripKey(trip[0], trip[1]), err, body.Code)
		}
		coords := body.Routes[0].Geometry.Coordinates
		durations := body.Routes[0].Legs[0].Annotation.Duration
		if len(durations) != len(coords)-1 {
			log.Fatalf("%s: %d points but %d durations", npc.TripKey(trip[0], trip[1]), len(coords), len(durations))
		}
		// OSRM starts and ends on the nearest road; pulling out of and into
		// the place itself (a driveway, a parking lot) takes parkSecs each,
		// so the parked position and the route meet.
		coords = append(append([][2]float64{{from.Lng, from.Lat}}, coords...), [2]float64{to.Lng, to.Lat})
		durations = append(append([]float64{parkSecs}, durations...), parkSecs)
		secs := make([]float64, len(coords))
		for j := 1; j < len(coords); j++ {
			secs[j] = secs[j-1] + durations[j-1]
		}
		// Keeping only the points that shape the road keeps routes.json
		// small (road trips run to thousands of points); the car still
		// passes each kept point at its own time.
		var r npc.Route
		for _, j := range simplify(coords, 0, len(coords)-1, []int{0}) {
			r.Points = append(r.Points, [2]float64{round(coords[j][0], 5), round(coords[j][1], 5)})
			r.Secs = append(r.Secs, round(secs[j], 1))
		}
		routes[npc.TripKey(trip[0], trip[1])] = r
		log.Printf("%d %s: %d of %d points, %.0f min", i+1, npc.TripKey(trip[0], trip[1]), len(r.Points), len(coords), r.Secs[len(r.Secs)-1]/60)
		time.Sleep(time.Second) // the demo server allows about one request a second
	}
	out, err := json.Marshal(routes)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("routes.json", out, 0o644); err != nil {
		log.Fatal(err)
	}
}

func round(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}

// tolerance is how far (in meters) a dropped point may be from the line
// between the points kept either side of it.
const tolerance = 4

// simplify appends to kept the indexes of the points between first
// (already kept) and last that a car can't skip without leaving the road by
// more than tolerance, then last (Douglas–Peucker).
func simplify(coords [][2]float64, first, last int, kept []int) []int {
	far, farthest := 0.0, -1
	for j := first + 1; j < last; j++ {
		if d := offLine(coords[j], coords[first], coords[last]); d > far {
			far, farthest = d, j
		}
	}
	if far > tolerance {
		kept = simplify(coords, first, farthest, kept)
		return simplify(coords, farthest, last, kept)
	}
	return append(kept, last)
}

// offLine is how far p is, in meters, from the segment from a to b.
func offLine(p, a, b [2]float64) float64 {
	const mPerDeg = 111_320
	k := math.Cos(a[1] * math.Pi / 180)
	px, py := (p[0]-a[0])*mPerDeg*k, (p[1]-a[1])*mPerDeg
	bx, by := (b[0]-a[0])*mPerDeg*k, (b[1]-a[1])*mPerDeg
	t := 0.0
	if l := bx*bx + by*by; l > 0 {
		t = math.Max(0, math.Min(1, (px*bx+py*by)/l))
	}
	return math.Hypot(px-t*bx, py-t*by)
}
