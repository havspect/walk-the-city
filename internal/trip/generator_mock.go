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

// Generate builds a rich, structured Itinerary based on destination, duration, pace, and interests.
func (m *MockGenerator) Generate(ctx context.Context, params CreateTripParams) (*Itinerary, error) {
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

	// Check for specialized city template (e.g. Rome)
	if strings.EqualFold(city, "Rome") || strings.Contains(strings.ToLower(params.Destination), "rome") {
		return m.generateRomeItinerary(month, pace, daysCount), nil
	}

	return m.generateGenericItinerary(city, month, pace, daysCount, params.Interests), nil
}

func (m *MockGenerator) generateRomeItinerary(month, pace string, daysCount int) *Itinerary {
	allDays := []DayPlan{
		{
			DayNumber: 1,
			Theme:     "Ancient Foundations & Monti Neighborhood Living",
			Morning: []Stop{
				{
					Name:           "Colosseum & Ludus Magnus",
					Neighborhood:   "Celio",
					Category:       "Architecture",
					Description:    "Explore the monumental 1st-century Flavian amphitheater and adjacent gladiatorial training school.",
					WalkingMinutes: 10,
					TransitTip:     "Metro Line B to Colosseo station",
					Lat:            41.8902,
					Lon:            12.4922,
					WikiQuery:      "Colosseum",
				},
				{
					Name:           "Roman Forum & Via Sacra",
					Neighborhood:   "Campitelli",
					Category:       "History",
					Description:    "Walk the ancient triumphal way between the Curia Julia, Temple of Saturn, and Arch of Titus.",
					WalkingMinutes: 5,
					TransitTip:     "Direct pedestrian walk from Colosseum",
					Lat:            41.8925,
					Lon:            12.4853,
					WikiQuery:      "Roman Forum",
				},
			},
			Afternoon: []Stop{
				{
					Name:           "Monti Artisan Alleys & Gelateria",
					Neighborhood:   "Monti",
					Category:       "Food & Living",
					Description:    "Stroll along picturesque cobblestones on Via Panisperna and sample artisanal pistachio gelato.",
					WalkingMinutes: 12,
					TransitTip:     "5-minute walk north of the Forum",
					Lat:            41.8950,
					Lon:            12.4920,
					WikiQuery:      "Monti (rione of Rome)",
				},
			},
			Evening: []Stop{
				{
					Name:           "Piazza Madonna dei Monti Aperitivo",
					Neighborhood:   "Monti",
					Category:       "Food & Living",
					Description:    "Unwind by the renaissance fountain with local Roman supplì al telefono and regional Lazio wine.",
					WalkingMinutes: 6,
					Lat:            41.8942,
					Lon:            12.4905,
					WikiQuery:      "Piazza della Madonna dei Monti",
				},
			},
		},
		{
			DayNumber: 2,
			Theme:     "Baroque Marvels & Historic Food Markets",
			Morning: []Stop{
				{
					Name:           "Pantheon & Oculus Architecture",
					Neighborhood:   "Pigna",
					Category:       "Architecture",
					Description:    "Stand beneath the unreinforced concrete dome that has stood intact for nearly 2,000 years.",
					WalkingMinutes: 15,
					TransitTip:     "Bus 64 or 70 from Termini to Largo di Torre Argentina",
					Lat:            41.8986,
					Lon:            12.4769,
					WikiQuery:      "Pantheon, Rome",
				},
				{
					Name:           "Piazza Navona & Fountain of Four Rivers",
					Neighborhood:   "Parione",
					Category:       "History",
					Description:    "Bernini's theatrical fountain set in the footprint of Emperor Domitian's ancient athletics stadium.",
					WalkingMinutes: 6,
					Lat:            41.8989,
					Lon:            12.4731,
					WikiQuery:      "Piazza Navona",
				},
			},
			Afternoon: []Stop{
				{
					Name:           "Campo de' Fiori Bakery & Deli",
					Neighborhood:   "Regola",
					Category:       "Food & Living",
					Description:    "Taste warm pizza bianca fresh from the historic forno paired with aged pecorino romano.",
					WalkingMinutes: 8,
					Lat:            41.8956,
					Lon:            12.4722,
					WikiQuery:      "Campo de' Fiori",
				},
			},
			Evening: []Stop{
				{
					Name:           "Ponte Sisto Sunset Walk",
					Neighborhood:   "Regola",
					Category:       "Food & Living",
					Description:    "Cross the pedestrian renaissance bridge into Trastevere with street musicians and Tiber river views.",
					WalkingMinutes: 7,
					Lat:            41.8920,
					Lon:            12.4708,
					WikiQuery:      "Ponte Sisto",
				},
			},
		},
		{
			DayNumber: 3,
			Theme:     "Trastevere Bohemian Lanes & Janiculum Panoramic Views",
			Morning: []Stop{
				{
					Name:           "Basilica of Santa Maria in Trastevere",
					Neighborhood:   "Trastevere",
					Category:       "Architecture",
					Description:    "Admire 12th-century glittering golden mosaics by Pietro Cavallini inside one of Rome's oldest churches.",
					WalkingMinutes: 12,
					TransitTip:     "Tram 8 to Piazza Sonnino",
					Lat:            41.8894,
					Lon:            12.4697,
					WikiQuery:      "Santa Maria in Trastevere",
				},
			},
			Afternoon: []Stop{
				{
					Name:           "Belvedere del Gianicolo",
					Neighborhood:   "Gianicolo",
					Category:       "History",
					Description:    "Climb up for a sweeping panoramic vista over the red rooftops, domes, and bell towers of Rome.",
					WalkingMinutes: 20,
					TransitTip:     "Bus 115 from Piazza della Rovere",
					Lat:            41.8916,
					Lon:            12.4619,
					WikiQuery:      "Janiculum",
				},
			},
			Evening: []Stop{
				{
					Name:           "Trattoria Da Enzo al 29",
					Neighborhood:   "Trastevere",
					Category:       "Food & Living",
					Description:    "Savor classic Roman pasta (cacio e pepe, amatriciana, and carbonara) in an authentic trattoria setting.",
					WalkingMinutes: 14,
					Lat:            41.8881,
					Lon:            12.4770,
					WikiQuery:      "Roman cuisine",
				},
			},
		},
	}

	// Slice days according to requested duration (or repeat / adapt)
	days := make([]DayPlan, 0, daysCount)
	for i := 0; i < daysCount; i++ {
		if i < len(allDays) {
			days = append(days, allDays[i])
		} else {
			dayNum := i + 1
			days = append(days, DayPlan{
				DayNumber: dayNum,
				Theme:     fmt.Sprintf("Neighborhood Immersion & Scenic Strolls (Day %d)", dayNum),
				Morning: []Stop{
					{
						Name:           "Villa Borghese Gardens & Galleria",
						Neighborhood:   "Pinciano",
						Category:       "Architecture",
						Description:    "Lush landscaped parkland featuring classical sculptures and peaceful shaded promenades.",
						WalkingMinutes: 15,
						TransitTip:     "Metro Line A to Flaminio station",
						Lat:            41.9142,
						Lon:            12.4921,
						WikiQuery:      "Villa Borghese gardens",
					},
				},
				Afternoon: []Stop{
					{
						Name:           "Testaccio Covered Market",
						Neighborhood:   "Testaccio",
						Category:       "Food & Living",
						Description:    "Traditional market stalls serving hot panini con trippa, panelle, and freshly brewed espresso.",
						WalkingMinutes: 18,
						TransitTip:     "Metro Line B to Piramide station",
						Lat:            41.8767,
						Lon:            12.4764,
						WikiQuery:      "Testaccio (rione of Rome)",
					},
				},
				Evening: []Stop{
					{
						Name:           "Appian Way Ancient Sunset Walk",
						Neighborhood:   "Appio Latino",
						Category:       "History",
						Description:    "Ancient basalt paving stones lined with towering pine trees, Roman tombs, and tranquil meadows.",
						WalkingMinutes: 20,
						TransitTip:     "Bus 118 or 218 from San Giovanni",
						Lat:            41.8540,
						Lon:            12.5200,
						WikiQuery:      "Appian Way",
					},
				},
			})
		}
	}

	return &Itinerary{
		Summary:    fmt.Sprintf("A %s-paced %d-day journey through Rome in %s, balancing ancient architecture, neighborhood alleys, and authentic trattorias.", strings.ToLower(pace), daysCount, month),
		BestSeason: fmt.Sprintf("%s is an ideal time with mild Mediterranean breezes and comfortable walking weather.", month),
		Days:       days,
		HighlightCards: []HighlightCard{
			{
				Category:       "Architecture",
				Title:          "Pantheon Concrete Dome",
				Neighborhood:   "Pigna",
				Story:          "Built under Emperor Hadrian in 125 AD, the dome remains the world's largest unreinforced concrete dome. The open 9-meter oculus at the peak connects the temple with the sky, allowing sunlight and rain to pour onto the marble floor.",
				Tip:            "Visit at midday to witness the sunbeam create a dramatic spotlight across the ancient interior.",
				WalkingMinutes: 10,
				TransitTip:     "Bus 64 or 70 to Largo di Torre Argentina",
				Lat:            41.8986,
				Lon:            12.4769,
				WikiQuery:      "Pantheon, Rome",
			},
			{
				Category:       "History",
				Title:          "Colosseum & Gladiatorial Legacy",
				Neighborhood:   "Celio",
				Story:          "Constructed between 72 and 80 AD by Emperors Vespasian and Titus, this 50,000-seat amphitheater showcased complex mechanical elevators that raised wild animals and gladiators onto the arena floor.",
				Tip:            "Walk the elevated Celian hill promenade for stunning crowd-free views of the outer facade.",
				WalkingMinutes: 8,
				TransitTip:     "Metro Line B (Colosseo station)",
				Lat:            41.8902,
				Lon:            12.4922,
				WikiQuery:      "Colosseum",
			},
			{
				Category:       "Food & Living",
				Title:          "Trastevere Artisan Bakeries & Supplì",
				Neighborhood:   "Trastevere",
				Story:          "Tucked away from main tourist thoroughfares, local forni bake crispy pizza bianca and golden supplì (crispy rice croquettes filled with mozzarella and meat ragù) following century-old family recipes.",
				Tip:            "Order supplì piping hot at 5 PM when fresh batches come out of the fryer.",
				WalkingMinutes: 12,
				TransitTip:     "Tram 8 to Piazza Sonnino",
				Lat:            41.8885,
				Lon:            12.4700,
				WikiQuery:      "Trastevere",
			},
			{
				Category:       "Hidden Gem",
				Title:          "Quartiere Coppedè Fairy-Tale Palazzi",
				Neighborhood:   "Trieste",
				Story:          "An unexpected architectural marvel designed by Gino Coppedè in the 1920s, merging Art Nouveau (Liberty), Baroque, and Gothic elements with whimsical animal carvings, chandeliers, and outdoor frescoes.",
				Tip:            "Look for the enormous outdoor wrought-iron chandelier suspended across the archway of Via Tagliamento.",
				WalkingMinutes: 20,
				TransitTip:     "Tram 3 or 19 to Piazza Buenos Aires",
				Lat:            41.9192,
				Lon:            12.5028,
				WikiQuery:      "Quartiere Coppedè",
			},
		},
		CoolFacts: []CoolFactCard{
			{
				Title:           "2,500+ Nasone Drinking Fountains",
				Fact:            "Rome features over 2,500 continuous cast-iron water fountains nicknamed 'nasoni' (big noses). They pour pure, ice-cold mountain spring water free for everyone. Covering the bottom nozzle forces water out of a small top hole, forming an easy drinking fountain.",
				SeasonalityNote: fmt.Sprintf("Carrying a reusable flask in %s keeps you effortlessly hydrated all day.", month),
			},
			{
				Title:           "Roman Aperitivo & Dinner Timing",
				Fact:            "Romans rarely sit down for dinner before 8:30 or 9:00 PM. Locals gather between 6:30 and 8:00 PM for an aperitivo (Spritz or local Frascati wine served with olives, chips, and small bites) in neighborhood squares like Piazza Madonna dei Monti.",
				SeasonalityNote: "Evening patio seating is lively and comfortable.",
			},
			{
				Title:           "San Pietro in Vincoli's Horned Moses",
				Fact:            "Michelangelo's famous statue of Moses in the church of San Pietro in Vincoli has two small horns on his head due to Saint Jerome's Latin translation mistaking the Hebrew word 'karan' (shining with light) for 'keren' (horned).",
				SeasonalityNote: "A quiet, peaceful sanctuary to escape afternoon warmth.",
			},
		},
	}
}

func (m *MockGenerator) generateGenericItinerary(city, month, pace string, daysCount int, interests []string) *Itinerary {
	interestsSummary := "architectural sights, historic quarters, and authentic local food"
	if len(interests) > 0 {
		interestsSummary = strings.Join(interests, ", ")
	}

	days := make([]DayPlan, 0, daysCount)
	for i := 1; i <= daysCount; i++ {
		days = append(days, DayPlan{
			DayNumber: i,
			Theme:     fmt.Sprintf("%s Historic Center & Quarter Exploration (Day %d)", city, i),
			Morning: []Stop{
				{
					Name:           fmt.Sprintf("%s Old Town Landmark & Square", city),
					Neighborhood:   "Historic Center",
					Category:       "Architecture",
					Description:    fmt.Sprintf("Begin your morning exploring the iconic civic architecture and landmark cathedral of %s.", city),
					WalkingMinutes: 10,
					TransitTip:     "Central Station or city center transit stop",
					WikiQuery:      fmt.Sprintf("%s architecture", city),
				},
			},
			Afternoon: []Stop{
				{
					Name:           fmt.Sprintf("%s Artisan Alleyways & Local Market", city),
					Neighborhood:   "Cultural Quarter",
					Category:       "Food & Living",
					Description:    fmt.Sprintf("Wander vibrant backstreets with local cafes, bakeries, and regional culinary specialties in %s.", city),
					WalkingMinutes: 12,
					TransitTip:     "Short walking distance through pedestrian zone",
					WikiQuery:      fmt.Sprintf("%s culture", city),
				},
			},
			Evening: []Stop{
				{
					Name:           fmt.Sprintf("%s Waterfront Promenade & Evening Dining", city),
					Neighborhood:   "Riverside / Old Quarter",
					Category:       "Food & Living",
					Description:    fmt.Sprintf("Relax at a cozy neighborhood bistro featuring seasonal %s specialties and evening ambiance.", month),
					WalkingMinutes: 8,
					WikiQuery:      fmt.Sprintf("%s cuisine", city),
				},
			},
		})
	}

	return &Itinerary{
		Summary:    fmt.Sprintf("A tailored %s-paced %d-day itinerary for %s in %s, focusing on %s.", strings.ToLower(pace), daysCount, city, month, interestsSummary),
		BestSeason: fmt.Sprintf("%s is a wonderful time to explore %s on foot with pleasant temperatures and lively cultural events.", month, city),
		Days:       days,
		HighlightCards: []HighlightCard{
			{
				Category:       "Architecture",
				Title:          fmt.Sprintf("Historic Heart & Civic Monuments of %s", city),
				Neighborhood:   "Old Quarter",
				Story:          fmt.Sprintf("The architectural heritage of %s reflects centuries of cultural evolution, featuring distinct facades, cobblestone plazas, and preserved civic landmarks.", city),
				Tip:            "Explore early in the morning for quiet photo opportunities and natural light.",
				WalkingMinutes: 10,
				WikiQuery:      fmt.Sprintf("%s history", city),
			},
			{
				Category:       "Food & Living",
				Title:          fmt.Sprintf("Authentic Neighborhood Food & Cafes in %s", city),
				Neighborhood:   "Market District",
				Story:          fmt.Sprintf("Away from major tourist corridors, neighborhood eateries in %s serve comforting regional dishes crafted from fresh seasonal ingredients.", city),
				Tip:            "Ask for the house specialty and daily seasonal specials.",
				WalkingMinutes: 12,
				WikiQuery:      fmt.Sprintf("%s food", city),
			},
			{
				Category:       "Hidden Gem",
				Title:          fmt.Sprintf("Secret Courtyards & Green Spaces of %s", city),
				Neighborhood:   "Arts District",
				Story:          fmt.Sprintf("Tucked behind historic residential gates are serene courtyards and pocket parks cherished by %s locals.", city),
				Tip:            "Look for open ironwork gates leading into quiet historic cloisters.",
				WalkingMinutes: 15,
				WikiQuery:      city,
			},
		},
		CoolFacts: []CoolFactCard{
			{
				Title:           fmt.Sprintf("Local Living & Etiquette in %s", city),
				Fact:            fmt.Sprintf("Neighborhood shops in %s often take a brief afternoon pause before re-opening for lively evening hours.", city),
				SeasonalityNote: fmt.Sprintf("Visiting in %s offers great weather for sidewalk cafes and evening walks.", month),
			},
			{
				Title:           fmt.Sprintf("Public Transit & Walking in %s", city),
				Fact:            fmt.Sprintf("%s is exceptionally walkable with dedicated pedestrian pathways connecting major cultural quarters.", city),
				SeasonalityNote: "Wear comfortable walking shoes with good support on cobblestones.",
			},
		},
	}
}
