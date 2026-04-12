package model

import "time"

type GitHubRepository struct {
	ID             int64
	FullName       string
	Owner          string
	Name           string
	LastSeenTag    *string
	LastReleaseURL *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
