package model

import "time"

type Subscription struct {
	ID               int64
	Email            string
	RepositoryID     int64
	Confirmed        bool
	Active           bool
	ConfirmToken     string
	UnsubscribeToken string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
