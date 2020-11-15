package provider

import (
	"time"
)

type Availability struct {
	SiteType string
	Date     time.Time
	URL      string
}

// Result is supposed to be a vendor neutral result of results
type Result struct {
	ID string

	Name     string
	Distance float64

	States []string

	Availability []Availability
	Amenities    string
}
