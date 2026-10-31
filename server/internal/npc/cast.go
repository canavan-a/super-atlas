// Package npc runs simulated friends for testing: made-up people around
// Albany, NY who drive from home to work, out to lunch and dinner, and home
// again each day, on real roads at the speed limit. Each has a fixed
// @lunargen.com email; sending a friend request to one is accepted at once,
// and from then on its position streams like a real friend's.
package npc

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
}

// NPC is one simulated friend: where they live and work, and where they go
// for lunch (from work, back to work) and dinner (from work, then home).
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

// Trips lists every drive any NPC can make, as [from, to] place keys: what
// the route generator fetches and routes.json holds.
func Trips() [][2]string {
	seen := map[[2]string]bool{}
	var out [][2]string
	add := func(from, to string) {
		t := [2]string{from, to}
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	for _, n := range cast {
		add(n.Home, n.Work)
		for _, f := range n.Lunch {
			add(n.Work, f)
			add(f, n.Work)
		}
		for _, f := range n.Dinner {
			add(n.Work, f)
			add(f, n.Home)
		}
	}
	return out
}

// PlaceAt returns the place with this key, for the route generator.
func PlaceAt(key string) Place { return places[key] }
