# Concepts

Shared domain vocabulary for this project — entities, named processes, and status concepts with project-specific meaning. Seeded with core domain vocabulary, then accretes as ce-compound and ce-compound-refresh process learnings; direct edits are fine. Glossary only, not a spec or catch-all.

## Trip Planning & Domain Models

### Trip
A planned journey to a destination city, encapsulating traveler preferences (travel month, duration, exploration pace, core interests, mobility mode) and an associated structured itinerary.
*Avoid:* Journey, vacation plan

A Trip is initially requested via the multi-step wizard and saved with a permanent shareable identifier. It contains one or more alternative Itineraries (themed plans). Its `DurationDays` scales the number of Stops in each Itinerary (more days → longer flat stop sequences, not day buckets).

### Itinerary
A themed or alternative plan synthesized for a Trip — for example a "Relaxed foodie" route versus a "Packed museums" route. An Itinerary is not tied to a calendar day.
*Avoid:* Schedule, tour guide, day plan

An Itinerary is a flat, ordered sequence of Stops joined by Segments that carry the transition (mode, distance, duration) between consecutive stops. It is referenced and viewed repeatedly via the Trip's permanent link.

### Day Plan
*Removed.* The old single-day breakdown (Morning / Afternoon / Evening) was eliminated in the model rebuild; an Itinerary is now a themed plan with no day concept.

### Stop
A highlight card within an Itinerary — a physical location, landmark, venue, or location-anchored trivia point.
*Avoid:* waypoint, point of interest

A Stop is discriminated by `kind` from the set {Landmark, History, FoodDrink, HiddenGem, ParkNature, Neighborhood, Trivia}. Every Stop always carries an image and text (title + body) plus an optional `tip` for short practical advice distinct from the narrative body. Stops contain location coordinates, a recommended visit duration, neighborhood, and a wiki query for deep links. Consecutive Stops within an Itinerary are joined by a Segment carrying mode, distance, duration, and instruction.

### Highlight Card
*Removed.* The old rich visual profile was folded into the unified Stop type in the model rebuild.

### Cool Fact Card
*Removed.* City trivia was folded into the `Trivia` Stop kind (location-anchored) in the model rebuild.

### Trip Generator
The synthesis engine responsible for translating traveler preferences into a Trip with one or more structured Itineraries, each containing Stops and Segments.

In development and automated testing environments, generation is provided by deterministic fixtures; in production, generation is backed by generative language models.
