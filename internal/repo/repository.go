package repo

import "time"

type Repository struct {
	ID             int64
	FullName       string
	Owner          string
	Name           string
	LastSeenTag    *string
	LastReleaseURL *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
