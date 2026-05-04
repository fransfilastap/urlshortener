package domain

import (
	"time"
)

type Click struct {
	ID        int64     `json:"id" db:"id"`
	URLID     int64     `json:"url_id" db:"url_id"`
	URLShort  string    `json:"url_short" db:"url_short"`
	IP        string    `json:"ip" db:"ip"`
	Location  string    `json:"location,omitempty" db:"location"`
	Browser   string    `json:"browser,omitempty" db:"browser"`
	Device    string    `json:"device,omitempty" db:"device"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
}

func NewClick(urlID int64, urlShort, ip, location, browser, device string) *Click {
	return &Click{
		URLID:     urlID,
		URLShort:  urlShort,
		IP:        ip,
		Location:  location,
		Browser:   browser,
		Device:    device,
		Timestamp: time.Now(),
	}
}