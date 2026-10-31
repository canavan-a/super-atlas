// Package npc runs simulated friends for testing: made-up people around
// Albany, NY who are always on the move, driving between businesses on real
// roads at the speed limit (highways faster than streets) and stopping at
// each for a few minutes. Now and then, more often on weekends, one goes on a
// road trip for a few days and comes back. Each has a fixed
// @lunargen.com email; sending a friend request to one is accepted at once,
// and from then on its position streams like a real friend's.
package npc

import (
	"math"
	"sort"
	"strings"
)

// Place is somewhere an NPC spends time. Coordinates are from OpenStreetMap
// (Nominatim) for the address given.
type Place struct {
	Name    string
	Address string
	Lat     float64
	Lng     float64
}

var places = map[string]Place{
	// Homes.
	"home-nadia":  {"Home", "1175 Lenox Rd, Schenectady, NY 12308", 42.8156725, -73.9252418},
	"home-owen":   {"Home", "1460 Union St, Schenectady, NY 12308", 42.8082751, -73.9137637},
	"home-lucia":  {"Home", "143 S Allen St, Albany, NY 12208", 42.6622499, -73.7964001},
	"home-caleb":  {"Home", "212 Hudson Ave, Albany, NY 12210", 42.6518134, -73.7630076},
	"home-clara":  {"Home", "325 Kenwood Ave, Delmar, NY 12054", 42.6214391, -73.8251613},
	"home-felix":  {"Home", "1480 New Scotland Rd, Slingerlands, NY 12159", 42.6314006, -73.8585370},
	"home-imani":  {"Home", "18 Westmere Terrace, Guilderland, NY 12203", 42.6902269, -73.8599200},
	"home-declan": {"Home", "12 Norwood St, Albany, NY 12203", 42.6811401, -73.8330250},

	// Workplaces.
	"work-ellis":      {"Ellis Hospital", "1101 Nott St, Schenectady, NY", 42.8202132, -73.9164609},
	"work-ge":         {"GE Vernova", "1 River Rd, Schenectady, NY", 42.8098463, -73.9518464},
	"work-amc":        {"Albany Med", "43 New Scotland Ave, Albany, NY", 42.6532132, -73.7766789},
	"work-esp":        {"Empire State Plaza", "Empire State Plaza, Albany, NY", 42.6509231, -73.7597839},
	"work-bchs":       {"Bethlehem Central High School", "700 Delaware Ave, Delmar, NY", 42.6117182, -73.8571073},
	"work-stpeters":   {"St. Peter's Hospital", "315 S Manning Blvd, Albany, NY", 42.6565930, -73.8040510},
	"work-ualbany":    {"University at Albany", "1400 Washington Ave, Albany, NY", 42.6872347, -73.8269388},
	"work-crossgates": {"Crossgates Mall", "1 Crossgates Mall Rd, Albany, NY", 42.6896022, -73.8499955},

	// Restaurants.
	"food-jacks":       {"Jack's Oyster House", "42 State St, Albany, NY", 42.6488078, -73.7512345},
	"food-yonos":       {"Yono's", "25 Chapel St, Albany, NY", 42.6532897, -73.7516606},
	"food-lombardos":   {"Lombardo's", "121 Madison Ave, Albany, NY", 42.6460242, -73.7551492},
	"food-capriccio":   {"Café Capriccio", "49 Grand St, Albany, NY", 42.6469090, -73.7561820},
	"food-larktavern":  {"Lark Tavern", "453 Madison Ave, Albany, NY", 42.6529567, -73.7680816},
	"food-bombers-alb": {"Bombers Burrito Bar", "258 Lark St, Albany, NY", 42.6540102, -73.7658905},
	"food-irongate":    {"Iron Gate Café", "182 Washington Ave, Albany, NY", 42.6559310, -73.7625820},
	"food-cafemadison": {"Café Madison", "1108 Madison Ave, Albany, NY", 42.6660126, -73.7917654},
	"food-tanpopo":     {"Tanpopo Ramen", "893 Broadway, Albany, NY", 42.6608662, -73.7448314},
	"food-wolffs":      {"Wolff's Biergarten", "895 Broadway, Albany, NY", 42.6609730, -73.7447700},
	"food-pumpstation": {"Albany Pump Station", "19 Quackenbush Sq, Albany, NY", 42.6541338, -73.7477671},
	"food-cornells":    {"Cornell's in Little Italy", "39 N Jay St, Schenectady, NY", 42.8176599, -73.9383179},
	"food-perrecas":    {"Perreca's Bakery", "33 N Jay St, Schenectady, NY", 42.8172945, -73.9383904},
	"food-ambition":    {"Ambition Café", "154 Jay St, Schenectady, NY", 42.8132930, -73.9408301},
	"food-mexradio":    {"Mexican Radio", "325 State St, Schenectady, NY", 42.8136981, -73.9426072},
	"food-bombers-sch": {"Bombers Burrito Bar", "447 State St, Schenectady, NY", 42.8124583, -73.9409542},
	"food-vandyck":     {"Van Dyck Lounge", "237 Union St, Schenectady, NY", 42.8165456, -73.9431394},
	"food-swiftys":     {"Swifty's", "392 Kenwood Ave, Delmar, NY", 42.6220471, -73.8327691},
	"food-bountiful":   {"Bountiful Bread", "Stuyvesant Plaza, Albany, NY", 42.6845192, -73.8397010},
	"food-cheesecake":  {"The Cheesecake Factory", "1417 Central Ave, Colonie, NY", 42.7103709, -73.8173242},

	// Road trip stops.
	"trip-buffalo-canalside": {"Canalside", "Canalside, Buffalo, NY", 42.8766286, -78.8786732},
	"trip-buffalo-anchorbar": {"Anchor Bar", "1047 Main St, Buffalo, NY", 42.9023050, -78.8686878},
	"trip-buffalo-niagara":   {"Niagara Falls State Park", "Niagara Falls, NY", 43.0809524, -79.0683019},
	"trip-nyc-katzs":         {"Katz's Delicatessen", "205 E Houston St, New York, NY", 40.7223431, -73.9873530},
	"trip-nyc-joes":          {"Joe's Pizza", "7 Carmine St, New York, NY", 40.7305674, -74.0020920},
	"trip-nyc-bbpark":        {"Brooklyn Bridge Park", "Brooklyn, NY", 40.6982888, -73.9999297},
	"trip-li-jonesbeach":     {"Jones Beach State Park", "Wantagh, NY", 40.5903821, -73.5463473},
	"trip-li-greenport":      {"Greenport", "Greenport, NY", 41.1032958, -72.3593093},
	"trip-li-montauk":        {"Montauk Point Lighthouse", "Montauk, NY", 41.0708988, -71.8573081},
	"trip-mtl-schwartzs":     {"Schwartz's", "3895 Boul Saint-Laurent, Montréal, QC", 45.5163277, -73.5776499},
	"trip-mtl-jeantalon":     {"Marché Jean-Talon", "Montréal, QC", 45.5359145, -73.6150625},
	"trip-mtl-vieuxport":     {"Vieux-Port", "Montréal, QC", 45.5042344, -73.5529866},
	"trip-tor-stlawrence":    {"St. Lawrence Market", "Toronto, ON", 43.6447293, -79.3804008},
	"trip-tor-cntower":       {"CN Tower", "Toronto, ON", 43.6425637, -79.3870872},
	"trip-tor-kensington":    {"Kensington Market", "Toronto, ON", 43.6552136, -79.4022604},
	"trip-cape-hyannis":      {"Main Street", "Hyannis, MA", 41.6528557, -70.2837675},
	"trip-cape-chatham":      {"Chatham Lighthouse", "Chatham, MA", 41.6714041, -69.9498826},
	"trip-cape-ptown":        {"MacMillan Pier", "Provincetown, MA", 42.0492944, -70.1816492},
}

// localStops are where NPCs stop around Albany: every workplace and
// restaurant. (Homes are kept for reference but no longer visited.)
var localStops = func() []string {
	var out []string
	for key := range places {
		if strings.HasPrefix(key, "work-") || strings.HasPrefix(key, "food-") {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}()

// hub is where road trips leave from and come back to: Crossgates, by the
// Thruway and the Northway.
const hub = "work-crossgates"

// destinations are road trips: drive from hub to the first stop, wander
// between the stops for a few days, then drive home from the first stop.
var destinations = [][]string{
	{"trip-buffalo-canalside", "trip-buffalo-anchorbar", "trip-buffalo-niagara"},
	{"trip-nyc-katzs", "trip-nyc-joes", "trip-nyc-bbpark"},
	{"trip-li-jonesbeach", "trip-li-greenport", "trip-li-montauk"},
	{"trip-mtl-schwartzs", "trip-mtl-jeantalon", "trip-mtl-vieuxport"},
	{"trip-tor-stlawrence", "trip-tor-cntower", "trip-tor-kensington"},
	{"trip-cape-hyannis", "trip-cape-chatham", "trip-cape-ptown"},
}

// NPC is one simulated friend. Home, work and favorite restaurants are kept
// from when NPCs ran a daily routine; they now wander (schedule.go).
type NPC struct {
	Name   string
	Email  string
	Home   string
	Work   string
	Lunch  []string
	Dinner []string
}

var cast = []NPC{
	{"Nadia Brennan", "nadia@lunargen.com", "home-nadia", "work-ellis",
		[]string{"food-perrecas", "food-ambition", "food-bombers-sch"},
		[]string{"food-cornells", "food-mexradio", "food-vandyck"}},
	{"Owen Takahashi", "owen@lunargen.com", "home-owen", "work-ge",
		[]string{"food-ambition", "food-bombers-sch", "food-perrecas"},
		[]string{"food-vandyck", "food-cornells", "food-mexradio"}},
	{"Lucia Ferreira", "lucia@lunargen.com", "home-lucia", "work-amc",
		[]string{"food-irongate", "food-bombers-alb", "food-cafemadison"},
		[]string{"food-larktavern", "food-lombardos", "food-capriccio"}},
	{"Caleb Whitmore", "caleb@lunargen.com", "home-caleb", "work-esp",
		[]string{"food-irongate", "food-pumpstation", "food-bombers-alb"},
		[]string{"food-jacks", "food-yonos", "food-wolffs"}},
	{"Clara Jensen", "clara@lunargen.com", "home-clara", "work-bchs",
		[]string{"food-swiftys", "food-cafemadison"},
		[]string{"food-swiftys", "food-larktavern", "food-capriccio"}},
	{"Felix Novak", "felix@lunargen.com", "home-felix", "work-stpeters",
		[]string{"food-cafemadison", "food-bountiful", "food-irongate"},
		[]string{"food-swiftys", "food-lombardos", "food-tanpopo"}},
	{"Imani Carter", "imani@lunargen.com", "home-imani", "work-ualbany",
		[]string{"food-bountiful", "food-cafemadison"},
		[]string{"food-cheesecake", "food-tanpopo", "food-wolffs"}},
	{"Declan O'Shea", "declan@lunargen.com", "home-declan", "work-crossgates",
		[]string{"food-bountiful", "food-cheesecake"},
		[]string{"food-cheesecake", "food-tanpopo", "food-yonos"}},
}

// Trips lists every drive an NPC can make, as [from, to] place keys: what
// the route generator fetches and routes.json holds. That is each edge of
// the wandering graph (neighbors) plus hub to each destination and back.
func Trips() [][2]string {
	var out [][2]string
	for _, from := range sortedKeys(neighbors) {
		for _, to := range neighbors[from] {
			out = append(out, [2]string{from, to})
		}
	}
	for _, d := range destinations {
		out = append(out, [2]string{hub, d[0]}, [2]string{d[0], hub})
	}
	return out
}

// neighbors is where an NPC can drive next from each stop. Around Albany:
// the 4 nearest stops at least a kilometer off, plus 2 farther ones so they
// roam the whole area (both ways, so every drive can be driven back). On a
// road trip: any other stop at the destination.
var neighbors = func() map[string][]string {
	set := map[string]map[string]bool{}
	link := func(a, b string) {
		for _, p := range [][2]string{{a, b}, {b, a}} {
			if set[p[0]] == nil {
				set[p[0]] = map[string]bool{}
			}
			set[p[0]][p[1]] = true
		}
	}
	for _, from := range localStops {
		var others []string
		for _, to := range localStops {
			if to != from && metersBetween(places[from], places[to]) >= 1000 {
				others = append(others, to)
			}
		}
		sort.SliceStable(others, func(i, j int) bool {
			return metersBetween(places[from], places[others[i]]) < metersBetween(places[from], places[others[j]])
		})
		for _, to := range others[:4] {
			link(from, to)
		}
		link(from, others[len(others)/2])
		link(from, others[len(others)-1])
	}
	for _, d := range destinations {
		for _, a := range d {
			for _, b := range d {
				if a != b {
					link(a, b)
				}
			}
		}
	}
	out := map[string][]string{}
	for from, tos := range set {
		out[from] = sortedKeys(tos)
	}
	return out
}()

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// metersBetween is the rough straight-line distance between two places.
func metersBetween(a, b Place) float64 {
	const mPerDeg = 111_320
	dx := (a.Lng - b.Lng) * mPerDeg * math.Cos(a.Lat*math.Pi/180)
	dy := (a.Lat - b.Lat) * mPerDeg
	return math.Hypot(dx, dy)
}

// PlaceAt returns the place with this key, for the route generator.
func PlaceAt(key string) Place { return places[key] }
