package trip

import (
	"context"
	"fmt"
	"strings"
)

// MockGenerator provides deterministic, structured city itineraries for offline testing and local development.
type MockGenerator struct{}

// NewMockGenerator constructs a MockGenerator instance.
func NewMockGenerator() *MockGenerator {
	return &MockGenerator{}
}

// Generate builds two themed Itineraries sized by DurationDays (R9, R10) with N-1 Segments (R12).
func (m *MockGenerator) Generate(ctx context.Context, params CreateTripParams) ([]*Itinerary, error) {
	city := strings.TrimSpace(params.City)
	if city == "" {
		parts := strings.Split(params.Destination, ",")
		city = strings.TrimSpace(parts[0])
	}
	if city == "" {
		city = "City"
	}

	month := strings.TrimSpace(params.Month)
	if month == "" {
		month = "Summer"
	}

	pace := strings.TrimSpace(params.Pace)
	if pace == "" {
		pace = "Moderate"
	}

	daysCount := params.DurationDays
	if daysCount < 1 {
		daysCount = 1
	}

	if strings.EqualFold(city, "Rome") || strings.Contains(strings.ToLower(params.Destination), "rome") {
		return m.generateRomeItineraries(month, pace, daysCount), nil
	}

	return m.generateGenericItineraries(city, month, pace, daysCount, params.Interests, params.Mobility), nil
}

// stopsForDuration returns how many stops an itinerary should have for the given trip DurationDays.
// More days => longer sequences (R10), flat lists, clamped to keep mock reasonable.
func stopsForDuration(days int) int {
	// 1 day -> 4 stops, 2 days -> 5, 3 -> 6-7, 7 -> 10, 14+ -> 14 max
	n := 3 + days
	if days >= 7 {
		n = 5 + days
	}
	if n < 4 {
		n = 4
	}
	if n > 14 {
		n = 14
	}
	return n
}

func placeholderFor(title string) string {
	seed := strings.ToLower(strings.TrimSpace(title))
	seed = strings.ReplaceAll(seed, " ", "-")
	var b strings.Builder
	for _, r := range seed {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if s == "" {
		s = "walk-the-city"
	}
	if len(s) > 40 {
		s = s[:40]
	}
	return "https://picsum.photos/seed/" + s + "/600/400"
}

func buildSegments(stops []Stop, mobility string) []Segment {
	if len(stops) < 2 {
		return nil
	}
	segments := make([]Segment, 0, len(stops)-1)
	for i := 0; i < len(stops)-1; i++ {
		from := stops[i]
		to := stops[i+1]
		mode := pickMode(i, mobility)
		dist := 400 + (i*230)%1100 // 400..1500 deterministic
		dur := dist / 85
		if dur < 4 {
			dur = 4
		}
		if mode == SegmentModeTransit {
			dur += 4
		} else if mode == SegmentModeBicycle {
			dur = dur/2 + 2
		}
		instr := fmt.Sprintf("Go from %s to %s", from.Title, to.Title)
		if mode == SegmentModeTransit {
			instr = fmt.Sprintf("Take transit from %s to %s", from.Title, to.Title)
		} else if mode == SegmentModeBicycle {
			instr = fmt.Sprintf("Cycle from %s to %s", from.Title, to.Title)
		}
		segments = append(segments, Segment{
			Position:        i,
			Mode:            mode,
			DistanceMeters:  dist,
			DurationMinutes: dur,
			Instruction:     instr,
		})
	}
	return segments
}

func pickMode(idx int, mobility string) string {
	m := strings.ToLower(mobility)
	if strings.Contains(m, "bicycle") {
		if idx%3 == 0 {
			return SegmentModeBicycle
		}
		return SegmentModeWalk
	}
	if strings.Contains(m, "walking only") {
		return SegmentModeWalk
	}
	if strings.Contains(m, "drive") {
		if idx%3 == 1 {
			return SegmentModeDrive
		}
		return SegmentModeWalk
	}
	// Default: alternate walk/transit
	if idx%2 == 1 {
		return SegmentModeTransit
	}
	return SegmentModeWalk
}

func (m *MockGenerator) generateRomeItineraries(month, pace string, daysCount int) []*Itinerary {
	n := stopsForDuration(daysCount)

	// Pool A: Classic Highlights (architecture/history/trivia focus)
	poolA := []Stop{
		{Kind: StopKindLandmark, Title: "Colosseum & Ludus Magnus", Body: "Explore the monumental 1st-century Flavian amphitheater and adjacent gladiatorial training school.", Neighborhood: "Celio", Lat: 41.8902, Lon: 12.4922, WikiQuery: "Colosseum", RecommendedMinutes: 60, Tip: "Book the underground tour early and enter via the Gladiator's Gate.", ImageURL: placeholderFor("Colosseum"), ImageAlt: "Colosseum", ImageSource: "placeholder"},
		{Kind: StopKindHistory, Title: "Roman Forum & Via Sacra", Body: "Walk the ancient triumphal way between the Curia Julia, Temple of Saturn, and Arch of Titus.", Neighborhood: "Campitelli", Lat: 41.8925, Lon: 12.4853, WikiQuery: "Roman Forum", RecommendedMinutes: 45, ImageURL: placeholderFor("Roman Forum"), ImageAlt: "Roman Forum", ImageSource: "placeholder"},
		{Kind: StopKindLandmark, Title: "Pantheon & Oculus Architecture", Body: "Stand beneath the unreinforced concrete dome that has stood intact for nearly 2,000 years.", Neighborhood: "Pigna", Lat: 41.8986, Lon: 12.4769, WikiQuery: "Pantheon, Rome", RecommendedMinutes: 30, Tip: "Visit at midday to see the sunbeam through the oculus.", ImageURL: placeholderFor("Pantheon"), ImageAlt: "Pantheon", ImageSource: "placeholder"},
		{Kind: StopKindHistory, Title: "Piazza Navona & Fountain of Four Rivers", Body: "Bernini's theatrical fountain set in the footprint of Emperor Domitian's athletics stadium.", Neighborhood: "Parione", Lat: 41.8989, Lon: 12.4731, WikiQuery: "Piazza Navona", RecommendedMinutes: 25, ImageURL: placeholderFor("Piazza Navona"), ImageAlt: "Piazza Navona", ImageSource: "placeholder"},
		{Kind: StopKindLandmark, Title: "Basilica of Santa Maria in Trastevere", Body: "Admire 12th-century glittering golden mosaics by Pietro Cavallini inside one of Rome's oldest churches.", Neighborhood: "Trastevere", Lat: 41.8894, Lon: 12.4697, WikiQuery: "Santa Maria in Trastevere", RecommendedMinutes: 30, ImageURL: placeholderFor("Santa Maria Trastevere"), ImageAlt: "Santa Maria in Trastevere", ImageSource: "placeholder"},
		{Kind: StopKindHistory, Title: "Belvedere del Gianicolo", Body: "Climb for a sweeping panoramic vista over the red rooftops, domes, and bell towers of Rome.", Neighborhood: "Gianicolo", Lat: 41.8916, Lon: 12.4619, WikiQuery: "Janiculum", RecommendedMinutes: 30, ImageURL: placeholderFor("Gianicolo"), ImageAlt: "Janiculum", ImageSource: "placeholder"},
		{Kind: StopKindLandmark, Title: "Villa Borghese Gardens & Galleria", Body: "Lush landscaped parkland featuring classical sculptures and peaceful shaded promenades.", Neighborhood: "Pinciano", Lat: 41.9142, Lon: 12.4921, WikiQuery: "Villa Borghese gardens", RecommendedMinutes: 60, Tip: "Rent a rowboat on the lake for a quiet break.", ImageURL: placeholderFor("Villa Borghese"), ImageAlt: "Villa Borghese", ImageSource: "placeholder"},
		{Kind: StopKindHistory, Title: "Appian Way Ancient Walk", Body: "Ancient basalt paving stones lined with towering pines, Roman tombs, and tranquil meadows.", Neighborhood: "Appio Latino", Lat: 41.8540, Lon: 12.5200, WikiQuery: "Appian Way", RecommendedMinutes: 45, ImageURL: placeholderFor("Appian Way"), ImageAlt: "Appian Way", ImageSource: "placeholder"},
		{Kind: StopKindNeighborhood, Title: "Quartiere Coppedè Fairy-Tale Palazzi", Body: "Whimsical 1920s Liberty-Baroque-Gothic ensemble with animal carvings and outdoor frescoes.", Neighborhood: "Trieste", Lat: 41.9192, Lon: 12.5028, WikiQuery: "Quartiere Coppedè", RecommendedMinutes: 25, Tip: "Look for the wrought-iron chandelier suspended across Via Tagliamento.", ImageURL: placeholderFor("Coppedè"), ImageAlt: "Coppede", ImageSource: "placeholder"},
		{Kind: StopKindTrivia, Title: "2,500+ Nasone Drinking Fountains", Body: "Rome's 2,500 cast-iron nasoni pour pure mountain spring water — cover the bottom nozzle to drink from the top hole.", Neighborhood: "Historic Center", Lat: 41.9000, Lon: 12.4833, WikiQuery: "Nasone", RecommendedMinutes: 10, ImageURL: placeholderFor("Nasone"), ImageAlt: "Nasone fountain", ImageSource: "placeholder"},
		{Kind: StopKindTrivia, Title: "San Pietro in Vincoli's Horned Moses", Body: "Michelangelo's Moses has horns due to Saint Jerome mistranslating Hebrew 'karan' (radiant) as 'keren' (horned).", Neighborhood: "Monti", Lat: 41.8938, Lon: 12.4931, WikiQuery: "Moses (Michelangelo)", RecommendedMinutes: 15, ImageURL: placeholderFor("Moses"), ImageAlt: "Moses statue", ImageSource: "placeholder"},
		{Kind: StopKindParkNature, Title: "Orange Garden (Giardino degli Aranci)", Body: "Terraced hilltop garden with bitter-orange trees framing a perfect dome view through the keyhole.", Neighborhood: "Aventino", Lat: 41.8833, Lon: 12.4802, WikiQuery: "Giardino degli Aranci", RecommendedMinutes: 20, ImageURL: placeholderFor("Orange Garden"), ImageAlt: "Orange Garden", ImageSource: "placeholder"},
	}

	// Pool B: Neighborhood & Food Immersion (food/drink, hidden gems, neighborhoods)
	poolB := []Stop{
		{Kind: StopKindFoodDrink, Title: "Monti Artisan Alleys & Gelateria", Body: "Stroll Via Panisperna for artisanal pistachio gelato from a family-run gelateria.", Neighborhood: "Monti", Lat: 41.8950, Lon: 12.4920, WikiQuery: "Monti (rione of Rome)", RecommendedMinutes: 30, Tip: "Order pistachio with a dash of sea salt.", ImageURL: placeholderFor("Monti Gelateria"), ImageAlt: "Gelateria", ImageSource: "placeholder"},
		{Kind: StopKindFoodDrink, Title: "Campo de' Fiori Bakery & Deli", Body: "Taste warm pizza bianca fresh from the historic forno paired with aged pecorino romano.", Neighborhood: "Regola", Lat: 41.8956, Lon: 12.4722, WikiQuery: "Campo de' Fiori", RecommendedMinutes: 25, ImageURL: placeholderFor("Campo Fiori"), ImageAlt: "Campo de' Fiori", ImageSource: "placeholder"},
		{Kind: StopKindNeighborhood, Title: "Piazza Madonna dei Monti Aperitivo", Body: "Unwind by the renaissance fountain with supplì al telefono and Lazio wine.", Neighborhood: "Monti", Lat: 41.8942, Lon: 12.4905, WikiQuery: "Piazza della Madonna dei Monti", RecommendedMinutes: 40, Tip: "Arrive at 6:30 PM for aperitivo hour.", ImageURL: placeholderFor("Madonna dei Monti"), ImageAlt: "Piazza Madonna", ImageSource: "placeholder"},
		{Kind: StopKindFoodDrink, Title: "Trastevere Artisan Bakeries & Supplì", Body: "Local forni bake crispy pizza bianca and golden supplì following century-old family recipes.", Neighborhood: "Trastevere", Lat: 41.8885, Lon: 12.4700, WikiQuery: "Trastevere", RecommendedMinutes: 25, Tip: "Order supplì at 5 PM when fresh batches come out.", ImageURL: placeholderFor("Supplì"), ImageAlt: "Suppli", ImageSource: "placeholder"},
		{Kind: StopKindFoodDrink, Title: "Trattoria Da Enzo al 29", Body: "Savor cacio e pepe, amatriciana, and carbonara in an authentic trattoria setting.", Neighborhood: "Trastevere", Lat: 41.8881, Lon: 12.4770, WikiQuery: "Roman cuisine", RecommendedMinutes: 60, Tip: "No reservations — arrive before 7 PM or after 9:30 PM.", ImageURL: placeholderFor("Da Enzo"), ImageAlt: "Da Enzo", ImageSource: "placeholder"},
		{Kind: StopKindHiddenGem, Title: "Testaccio Covered Market", Body: "Traditional stalls serving hot panini con trippa, panelle, and freshly brewed espresso.", Neighborhood: "Testaccio", Lat: 41.8767, Lon: 12.4764, WikiQuery: "Testaccio (rione of Rome)", RecommendedMinutes: 35, ImageURL: placeholderFor("Testaccio Market"), ImageAlt: "Testaccio Market", ImageSource: "placeholder"},
		{Kind: StopKindParkNature, Title: "Ponte Sisto Sunset Walk", Body: "Cross the pedestrian renaissance bridge into Trastevere with Tiber river views.", Neighborhood: "Regola", Lat: 41.8920, Lon: 12.4708, WikiQuery: "Ponte Sisto", RecommendedMinutes: 20, ImageURL: placeholderFor("Ponte Sisto"), ImageAlt: "Ponte Sisto", ImageSource: "placeholder"},
		{Kind: StopKindNeighborhood, Title: "Ghetto & Portico d'Ottavia", Body: "Jewish-Roman heritage with fried artichokes (carciofi alla giudia) and quiet courtyards.", Neighborhood: "Sant'Angelo", Lat: 41.8922, Lon: 12.4783, WikiQuery: "Roman Ghetto", RecommendedMinutes: 30, ImageURL: placeholderFor("Roman Ghetto"), ImageAlt: "Roman Ghetto", ImageSource: "placeholder"},
		{Kind: StopKindHiddenGem, Title: "Aventine Keyhole Surprise", Body: "Peek through the Priory's keyhole for a perfectly framed view of St. Peter's dome.", Neighborhood: "Aventino", Lat: 41.8829, Lon: 12.4775, WikiQuery: "Santa Maria del Priorato", RecommendedMinutes: 15, Tip: "Early morning has no queue.", ImageURL: placeholderFor("Aventine Keyhole"), ImageAlt: "Aventine Keyhole", ImageSource: "placeholder"},
		{Kind: StopKindFoodDrink, Title: "Sant'Eustachio Gran Caffè", Body: "Historic coffee bar famous for its frothy gran caffè — shaken, sweet, and ice-cold.", Neighborhood: "Sant'Eustachio", Lat: 41.8980, Lon: 12.4758, WikiQuery: "Sant'Eustachio", RecommendedMinutes: 15, ImageURL: placeholderFor("Sant Eustachio"), ImageAlt: "Sant Eustachio", ImageSource: "placeholder"},
		{Kind: StopKindTrivia, Title: "Roman Aperitivo Timing", Body: "Locals gather 6:30–8 PM for Spritz or Frascati with olives and chips before dinner at 8:30+.", Neighborhood: "Centro", Lat: 41.9005, Lon: 12.4770, WikiQuery: "Aperitivo", RecommendedMinutes: 10, ImageURL: placeholderFor("Aperitivo"), ImageAlt: "Aperitivo", ImageSource: "placeholder"},
		{Kind: StopKindParkNature, Title: "Pincian Hill Terrace Sunset", Body: "Wide terrace above Piazza del Popolo with sunset light over the domes and pines.", Neighborhood: "Pinciano", Lat: 41.9105, Lon: 12.4785, WikiQuery: "Pincian Hill", RecommendedMinutes: 20, ImageURL: placeholderFor("Pincian Hill"), ImageAlt: "Pincian Hill", ImageSource: "placeholder"},
	}

	if n > len(poolA) {
		n = len(poolA)
	}
	if n > len(poolB) {
		n = len(poolB)
	}

	stopsA := make([]Stop, n)
	copy(stopsA, poolA[:n])
	stopsB := make([]Stop, n)
	copy(stopsB, poolB[:n])

	// Assign positions
	for i := range stopsA {
		stopsA[i].Position = i
	}
	for i := range stopsB {
		stopsB[i].Position = i
	}

	return []*Itinerary{
		{
			Title:      "Classic Highlights",
			Theme:      "Ancient & Architectural",
			Summary:    fmt.Sprintf("A %s-paced %d-day classic route through Rome in %s, balancing ancient monuments, piazzas, and panoramic terraces.", strings.ToLower(pace), daysCount, month),
			BestSeason: fmt.Sprintf("%s is ideal with mild Mediterranean breezes and comfortable walking weather.", month),
			Position:   0,
			Stops:      stopsA,
			Segments:   buildSegments(stopsA, ""),
		},
		{
			Title:      "Neighborhood & Food Immersion",
			Theme:      "Local Life & Hidden Gems",
			Summary:    fmt.Sprintf("A %s-paced %d-day neighborhood route in %s — artisan alleys, market stalls, and Trastevere evenings.", strings.ToLower(pace), daysCount, month),
			BestSeason: fmt.Sprintf("%s evenings are lively and comfortable for aperitivo and slow dining.", month),
			Position:   1,
			Stops:      stopsB,
			Segments:   buildSegments(stopsB, ""),
		},
	}
}

func (m *MockGenerator) generateGenericItineraries(city, month, pace string, daysCount int, interests []string, mobility string) []*Itinerary {
	interestsSummary := "architectural sights, historic quarters, and authentic local food"
	if len(interests) > 0 {
		interestsSummary = strings.Join(interests, ", ")
	}

	n := stopsForDuration(daysCount)

	poolA := make([]Stop, 0, 12)
	for i := 0; i < 12; i++ {
		kind := StopKindLandmark
		switch i % 4 {
		case 0:
			kind = StopKindLandmark
		case 1:
			kind = StopKindHistory
		case 2:
			kind = StopKindParkNature
		case 3:
			kind = StopKindTrivia
		}
		title := fmt.Sprintf("%s Old Town Landmark %d", city, i+1)
		body := fmt.Sprintf("Explore the civic architecture and landmark square %d in %s.", i+1, city)
		if kind == StopKindTrivia {
			title = fmt.Sprintf("Local quirk %d in %s", i+1, city)
			body = fmt.Sprintf("A surprising local custom or hidden fact %d cherished by %s residents.", i+1, city)
		}
		poolA = append(poolA, Stop{
			Kind: kind, Title: title, Body: body,
			Neighborhood: fmt.Sprintf("District %d", (i%3)+1),
			WikiQuery: fmt.Sprintf("%s %d", city, i+1), RecommendedMinutes: 20 + (i%3)*10,
			ImageURL: placeholderFor(title), ImageAlt: title, ImageSource: "placeholder",
		})
	}

	poolB := make([]Stop, 0, 12)
	for i := 0; i < 12; i++ {
		kind := StopKindFoodDrink
		switch i % 4 {
		case 0:
			kind = StopKindFoodDrink
		case 1:
			kind = StopKindHiddenGem
		case 2:
			kind = StopKindNeighborhood
		case 3:
			kind = StopKindTrivia
		}
		title := fmt.Sprintf("%s Food & Alley %d", city, i+1)
		body := fmt.Sprintf("Wander backstreets and sample regional specialties at hidden spot %d in %s.", i+1, city)
		if kind == StopKindHiddenGem {
			title = fmt.Sprintf("%s Secret Courtyard %d", city, i+1)
			body = fmt.Sprintf("A quiet courtyard or pocket park %d loved by %s locals.", i+1, city)
		}
		if kind == StopKindTrivia {
			title = fmt.Sprintf("%s Market Etiquette %d", city, i+1)
			body = fmt.Sprintf("Local shopping and dining custom %d in %s.", i+1, city)
		}
		poolB = append(poolB, Stop{
			Kind: kind, Title: title, Body: body,
			Neighborhood: fmt.Sprintf("Quarter %d", (i%3)+1),
			WikiQuery: fmt.Sprintf("%s food %d", city, i+1), RecommendedMinutes: 20 + (i%3)*10,
			Tip: func() string {
				if kind == StopKindFoodDrink {
					return "Ask for the daily seasonal special."
				}
				return ""
			}(),
			ImageURL: placeholderFor(title), ImageAlt: title, ImageSource: "placeholder",
		})
	}

	if n > len(poolA) {
		n = len(poolA)
	}
	stopsA := make([]Stop, n)
	copy(stopsA, poolA[:n])
	stopsB := make([]Stop, n)
	copy(stopsB, poolB[:n])
	for i := range stopsA {
		stopsA[i].Position = i
	}
	for i := range stopsB {
		stopsB[i].Position = i
	}

	return []*Itinerary{
		{
			Title:      fmt.Sprintf("Classic %s", city),
			Theme:      "Landmarks & History",
			Summary:    fmt.Sprintf("A tailored %s-paced %d-day classic itinerary for %s in %s, focusing on %s.", strings.ToLower(pace), daysCount, city, month, interestsSummary),
			BestSeason: fmt.Sprintf("%s is a wonderful time to explore %s on foot with pleasant temperatures and lively cultural events.", month, city),
			Position:   0,
			Stops:      stopsA,
			Segments:   buildSegments(stopsA, mobility),
		},
		{
			Title:      fmt.Sprintf("Local %s", city),
			Theme:      "Food & Neighborhoods",
			Summary:    fmt.Sprintf("A %s-paced local-life route for %s in %s — neighborhood eateries and hidden green spaces.", strings.ToLower(pace), city, month),
			BestSeason: fmt.Sprintf("%s evenings are perfect for sidewalk dining in %s.", month, city),
			Position:   1,
			Stops:      stopsB,
			Segments:   buildSegments(stopsB, mobility),
		},
	}
}
