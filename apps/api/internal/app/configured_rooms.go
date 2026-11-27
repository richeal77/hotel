package app

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

func validPublishStatus(status string) bool {
	return status == "draft" || status == "active" || status == "inactive"
}

func writeRoom(c *gin.Context, store *Store, body map[string]json.RawMessage, now time.Time) (any, error) {
	var req struct {
		Name     *string         `json:"name"`
		Sort     *int            `json:"sort_order"`
		Quantity *int            `json:"default_sellable_quantity"`
		Details  json.RawMessage `json:"details"`
		Version  int             `json:"version"`
		Status   *string         `json:"status"`
	}
	id := c.Param("id")
	room, exists := store.Configured.Rooms[id]
	if id == "" {
		if err := decodeConfigured(body, &req, "name", "sort_order", "default_sellable_quantity", "details"); err != nil {
			return nil, err
		}
		if body["version"] != nil || req.Status != nil {
			return nil, invalid("status")
		}
		room = roomType{ID: newConfiguredID(), Status: "draft"}
	} else {
		if !exists {
			return nil, issue(404, "NOT_FOUND")
		}
		if err := decodeConfigured(body, &req, "version"); err != nil {
			return nil, err
		}
		if err := checkVersion(req.Version, room.Version); err != nil {
			return nil, err
		}
	}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" || utf8.RuneCountInString(*req.Name) > 80 {
			return nil, invalid("name")
		}
		room.Name = *req.Name
	}
	if req.Sort != nil {
		room.SortOrder = *req.Sort
	}
	if req.Quantity != nil {
		if *req.Quantity < 0 {
			return nil, invalid("default_sellable_quantity")
		}
		for _, day := range store.Configured.Days[room.ID] {
			if day.Date >= shopDate(now) && day.Override == nil && day.Reserved > *req.Quantity {
				return nil, issue(409, "ROOM_SOLD_OUT")
			}
		}
		room.DefaultQuantity = *req.Quantity
	}
	if req.Details != nil {
		if err := validateRoomDetails(req.Details); err != nil {
			return nil, err
		}
		room.Details = req.Details
	}
	if req.Status != nil {
		if !validPublishStatus(*req.Status) {
			return nil, invalid("status")
		}
		room.Status = *req.Status
	}
	room.Version++
	room.UpdatedAt = now
	store.Configured.Rooms[room.ID] = room
	return gin.H{"room": room}, nil
}

func validateRoomDetails(raw json.RawMessage) error {
	var details struct {
		Photos []string `json:"photo_urls"`
		Beds   []struct {
			Type  string `json:"type"`
			Count int    `json:"count"`
			Width *int   `json:"width_cm"`
		} `json:"beds"`
		Amenities []string `json:"amenities"`
		Guests    *int     `json:"max_guests"`
		Area      *int     `json:"area_m2"`
		Window    *string  `json:"window"`
		Notes     string   `json:"notes"`
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || decodeConfigured(fields, &details, "photo_urls", "beds", "amenities") != nil {
		return invalid("details")
	}
	for _, photo := range details.Photos {
		u, err := url.Parse(photo)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
			return invalid("details.photo_urls")
		}
	}
	for _, bed := range details.Beds {
		if strings.TrimSpace(bed.Type) == "" || bed.Count < 1 || (bed.Width != nil && *bed.Width < 1) {
			return invalid("details.beds")
		}
	}
	if details.Guests != nil && *details.Guests < 1 || details.Area != nil && *details.Area < 1 {
		return invalid("details")
	}
	if details.Window != nil && *details.Window != "yes" && *details.Window != "no" && *details.Window != "partial" {
		return invalid("details.window")
	}
	return nil
}

func writeRate(c *gin.Context, store *Store, body map[string]json.RawMessage, now time.Time) (any, error) {
	var req struct {
		Name         *string `json:"name"`
		Nightly      *int    `json:"base_nightly_cent"`
		Breakfast    *int    `json:"breakfast_count"`
		Cancellation *string `json:"cancellation_text"`
		Version      int     `json:"version"`
		Status       *string `json:"status"`
	}
	id := c.Param("id")
	var rate roomRate
	if c.Request.Method == "POST" {
		if _, ok := store.Configured.Rooms[id]; !ok {
			return nil, issue(404, "NOT_FOUND")
		}
		if err := decodeConfigured(body, &req, "name", "base_nightly_cent", "breakfast_count", "cancellation_text"); err != nil {
			return nil, err
		}
		if body["version"] != nil || req.Status != nil {
			return nil, invalid("status")
		}
		rate = roomRate{ID: newConfiguredID(), RoomID: id, Status: "draft"}
	} else {
		var ok bool
		rate, ok = store.Configured.Rates[id]
		if !ok {
			return nil, issue(404, "NOT_FOUND")
		}
		if err := decodeConfigured(body, &req, "version"); err != nil {
			return nil, err
		}
		if err := checkVersion(req.Version, rate.Version); err != nil {
			return nil, err
		}
	}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" || utf8.RuneCountInString(*req.Name) > 80 {
			return nil, invalid("name")
		}
		rate.Name = *req.Name
	}
	if req.Nightly != nil {
		if *req.Nightly < 0 || *req.Nightly > int(^uint(0)>>1)/366 {
			return nil, invalid("base_nightly_cent")
		}
		rate.Nightly = *req.Nightly
	}
	if req.Breakfast != nil {
		if *req.Breakfast < 0 {
			return nil, invalid("breakfast_count")
		}
		rate.Breakfast = *req.Breakfast
	}
	if req.Cancellation != nil {
		if utf8.RuneCountInString(*req.Cancellation) > 500 {
			return nil, invalid("cancellation_text")
		}
		rate.Cancellation = *req.Cancellation
	}
	if req.Status != nil {
		if !validPublishStatus(*req.Status) {
			return nil, invalid("status")
		}
		rate.Status = *req.Status
	}
	rate.Version++
	rate.UpdatedAt = now
	store.Configured.Rates[rate.ID] = rate
	return gin.H{"rate": rate}, nil
}

func roomInventory(c *gin.Context, store *Store, body map[string]json.RawMessage, _ time.Time) (any, error) {
	s := &store.Configured
	id := c.Param("id")
	room, ok := s.Rooms[id]
	if !ok {
		return nil, issue(404, "NOT_FOUND")
	}
	if c.Request.Method == "GET" {
		dates, err := stayDates(c.Query("from"), c.Query("to"))
		if err != nil {
			return nil, err
		}
		return s.inventory(room, dates), nil
	}
	var req struct {
		Version int                          `json:"version"`
		From    string                       `json:"from"`
		To      string                       `json:"to"`
		Items   []map[string]json.RawMessage `json:"items"`
	}
	if err := decodeConfigured(body, &req, "version", "from", "to", "items"); err != nil {
		return nil, err
	}
	if err := checkVersion(req.Version, s.inventoryVersion(id)); err != nil {
		return nil, err
	}
	dates, err := stayDates(req.From, req.To)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	if s.Days[id] == nil {
		s.Days[id] = make(map[string]inventoryDay)
	}
	for _, item := range req.Items {
		var update struct {
			Date   string `json:"stay_date"`
			Closed bool   `json:"closed"`
		}
		override, hasOverride := item["sellable_quantity"]
		delete(item, "sellable_quantity")
		if !hasOverride || decodeConfigured(item, &update, "stay_date", "closed") != nil {
			return nil, invalid("items")
		}
		if update.Date < req.From || update.Date >= req.To || seen[update.Date] {
			return nil, invalid("items.stay_date")
		}
		if _, err := time.Parse(time.DateOnly, update.Date); err != nil {
			return nil, invalid("items.stay_date")
		}
		seen[update.Date] = true
		day := s.day(room, update.Date)
		if err := json.Unmarshal(override, &day.Override); err != nil || (day.Override != nil && *day.Override < 0) {
			return nil, invalid("items.sellable_quantity")
		}
		quantity := room.DefaultQuantity
		if day.Override != nil {
			quantity = *day.Override
		}
		if quantity < day.Reserved {
			return nil, issue(409, "ROOM_SOLD_OUT")
		}
		day.Closed = update.Closed
		s.Days[id][update.Date] = day
	}
	s.InventoryVersions[id] = req.Version + 1
	return s.inventory(room, dates), nil
}
