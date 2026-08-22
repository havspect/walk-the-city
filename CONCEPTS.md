# Concepts

Shared domain vocabulary for this project — entities, named processes, and status concepts with project-specific meaning. Seeded with core domain vocabulary, then accretes as ce-compound and ce-compound-refresh process learnings; direct edits are fine. Glossary only, not a spec or catch-all.

## Trip Planning & Domain Models

### Trip
A planned journey to a destination city, encapsulating traveler preferences (travel month, duration, exploration pace, core interests, mobility mode) and an associated structured itinerary.
*Avoid:* Journey, vacation plan

A Trip is initially requested via the multi-step wizard and saved with a permanent shareable identifier. It contains zero or one generated Itineraries.

### Itinerary
The comprehensive schedule synthesized for a Trip, organized into chronological daily routes, curated highlight cards, and city trivia.
*Avoid:* Schedule, tour guide

An Itinerary is immutable once generated and persisted, but can be referenced and viewed repeatedly via the Trip's permanent link.

### Day Plan
The single-day breakdown within an Itinerary, organizing recommended stops across chronological time segments (Morning, Afternoon, and Evening).

Each Day Plan includes estimated walking or public transit logistics between consecutive stops.

### Stop
A specific physical location, landmark, or venue visit within a Day Plan.

Stops contain location coordinates, approximate visit duration, transit instructions, and direct links to mapping providers.

### Highlight Card
A rich visual profile highlighting an architectural landmark, historical event, or local culinary experience.

Highlight Cards pair historical lore and architectural insights with practical insider tips, neighborhood categorization, and deep links to map search and encyclopedia references.

### Cool Fact Card
A concise contextual trivia item detailing local customs, architectural quirks, or seasonal advice for the destination city.

### Trip Generator
The synthesis engine responsible for translating traveler preferences into a structured Itinerary.

In development and automated testing environments, generation is provided by deterministic fixtures; in production, generation is backed by generative language models.
