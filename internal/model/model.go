package model

import (
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
)

type Link struct {
	ID        uuid.UUID
	OriginURL string
	ShortCode string
	Created   time.Time
}

type Click struct {
	ID        uuid.UUID
	LinkID    uuid.UUID
	UserAgent string
	IPAddress string
	Clicked   time.Time
}

var (
	ErrAlreadyExists   = errors.New("record already exists")
	ErrCustomCodeTaken = errors.New("this custom short code is already taken")
	ErrLinkNotFound    = errors.New("link not found")
)
