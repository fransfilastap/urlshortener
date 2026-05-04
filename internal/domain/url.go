package domain

import (
	"time"
)

type URL struct {
	ID               int64      `json:"id" db:"id"`
	Original         string     `json:"original" db:"original"`
	Short            string     `json:"short" db:"short"`
	Title            string     `json:"title" db:"title"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	ExpiresAt        time.Time  `json:"expires_at,omitempty" db:"expires_at"`
	Clicks           int64      `json:"clicks" db:"clicks"`
	CreatorReference string     `json:"creator_reference,omitempty" db:"creator_reference"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

func NewURL(original, short, title string, expiresAt time.Time, creatorReference string) *URL {
	return &URL{
		Original:         original,
		Short:            short,
		Title:            title,
		CreatedAt:        time.Now(),
		ExpiresAt:        expiresAt,
		Clicks:           0,
		CreatorReference: creatorReference,
	}
}