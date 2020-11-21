package backend

import (
	"encoding/json"
	"fmt"
	"net/http/cookiejar"
	"time"

	"github.com/tstromberg/campwiz/pkg/cache"
	"github.com/tstromberg/campwiz/pkg/campwiz"
	"k8s.io/klog/v2"
)

// RCalifornia handles RCalifornia queries
type RCalifornia struct {
	store cache.Store
	jar   *cookiejar.Jar
}

// Name is a human readable name
func (b *RCalifornia) Name() string {
	return "RCalifornia"
}

// List lists available sites
func (b *RCalifornia) List(q campwiz.Query) ([]campwiz.Result, error) {
	return nil, nil
}

// raURL is the search URL to request reservation information from.
var rcURL = "https://" + "www." + "reserve" + "california.com"

type rcRequest struct {
	PlaceID             int    `json:"PlaceId"`
	Latitude            string `json:"Latitude"`
	Longitude           string `json:"Longitude"`
	HighlightedPlaceId  int    `json:"HighlightedPlaceId"`
	StartDate           string `json:"StartDate"`
	Nights              string `json:"Nights"`
	CountNearby         bool   `json:"CountNearby"`
	NearbyLimit         int    `json:"NearbyLimit"`
	NearbyOnlyAvailable bool   `json:"NearbyOnlyAvailable"`
	NearbyCountLimit    int    `json:"NearbyCountLimit"`
	Sort                string `json:"Sort"`
	CustomerID          string `json:"CustomerID"`
	RefreshFavourites   bool   `json:"RefreshFavourites"`
	IsADA               bool   `json:"IsADA"`
	UnitCategoryID      int    `json:"UnitCategoryId"`
	SleepingUnitID      int    `json:"SleepingUnitId"`
	MinVehicleLength    int    `json:"MinVehicleLength"`
	UnitTypesGroupIDs   []int  `json:"UnitTypeGroupIds"`
}

type rcPlace struct {
	AllHighlights     string  `json:"Allhighlights"`
	Available         bool    `json:"Available"`
	Description       string  `json:"Description"`
	Latitude          float64 `json:"Latitude"`
	Longitude         float64 `json:"Longitude"`
	MilesFromSelected int     `json:"MilesFromSelected"`
	Name              string  `json:"Name"`
	PlaceID           int     `json:"PlaceId"`
	URL               string  `json:"Url"`
}

type rcResponse struct {
	NearbyPlaces []rcPlace
}

// rcPageRequest creates the request object for a search.
func rcPageRequest(q campwiz.Query, arrival time.Time) (cache.Request, error) {
	rcr := rcRequest{
		Latitude:            fmt.Sprintf("%.4f", q.Lat),
		Longitude:           fmt.Sprintf("%.4f", q.Lon),
		StartDate:           arrival.Format("01-02-2006"),
		CountNearby:         true,
		CustomerID:          "0",
		Nights:              fmt.Sprintf("%d", q.StayLength),
		PlaceID:             0,
		RefreshFavourites:   true,
		Sort:                "Distance",
		NearbyLimit:         q.MaxDistance,
		NearbyOnlyAvailable: true,
		NearbyCountLimit:    100,
	}

	body, err := json.Marshal(&rcr)
	if err != nil {
		return cache.Request{}, fmt.Errorf("marshal: %w", err)
	}

	r := cache.Request{
		Method:      "POST",
		URL:         "https://calirdr.usedirect.com/rdr/rdr/search/place",
		Referrer:    rcURL,
		MaxAge:      searchPageExpiry,
		ContentType: "application/json",
		Body:        body,
	}

	return r, nil
}

func parseRCSearchPage(bs []byte, date time.Time, q campwiz.Query) ([]campwiz.Result, error) {
	klog.Infof("parse rc page: %s", bs)

	var rr rcResponse
	err := json.Unmarshal(bs, &rr)
	if err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	klog.V(2).Infof("unmarshalled data: %+v", rr)

	var results []campwiz.Result
	for _, r := range rr.NearbyPlaces {
		if !r.Available {
			continue
		}

		a := campwiz.Availability{
			SiteType: "campsite",
			Date:     date,
			URL:      rcURL + "/CaliforniaWebHome/Facilities/SearchViewUnitAvailabity.aspx",
		}

		rr := campwiz.Result{
			ID:           fmt.Sprintf("/rc/%d", r.PlaceID),
			Name:         r.Name,
			Description:  r.Description,
			Features:     r.AllHighlights,
			Distance:     float64(r.MilesFromSelected),
			Availability: []campwiz.Availability{a},
			URL:          r.URL,
		}

		klog.Infof("%s is available: %+v", r.Name, rr)
		results = append(results, rr)
	}

	return results, nil
}

// searchRC runs a search for a single date
func searchRC(q campwiz.Query, date time.Time, cs cache.Store) ([]campwiz.Result, error) {
	req, err := rcPageRequest(q, date)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}

	resp, err := cache.Fetch(req, cs)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	results, err := parseRCSearchPage(resp.Body, date, q)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	klog.Infof("returning %d results", len(results))
	return results, nil
}