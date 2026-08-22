package trip

import (
	"time"
)

// Trip represents a user-created or agent-generated city itinerary.
type Trip struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Destination  string    `gorm:"size:255;not null" json:"destination"`
	DurationDays int       `gorm:"not null;default:1" json:"duration_days"`
	Notes        string    `gorm:"type:text" json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
