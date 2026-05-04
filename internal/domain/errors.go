package domain

import "errors"

var (
	ErrURLNotFound = errors.New("url not found")
	ErrURLExists   = errors.New("url with this short code already exists")
	ErrInvalidURL  = errors.New("invalid url")
	ErrRecentClick = errors.New("recent click from the same visitor")
)