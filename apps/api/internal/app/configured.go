package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"time"
)

type configuredStore struct {
	Rooms             map[string]roomType
	Rates             map[string]roomRate
	Days              map[string]map[string]inventoryDay
	InventoryVersions map[string]int
	Quotes            map[string]savedQuote
	Orders            map[string]configuredOrder
	Requests          map[string]configuredRequest
}

func (s *configuredStore) initialize() {
	if s.Rooms == nil {
		s.Rooms = make(map[string]roomType)
	}
	if s.Rates == nil {
		s.Rates = make(map[string]roomRate)
	}
	if s.Days == nil {
		s.Days = make(map[string]map[string]inventoryDay)
	}
	if s.InventoryVersions == nil {
		s.InventoryVersions = make(map[string]int)
	}
	if s.Quotes == nil {
		s.Quotes = make(map[string]savedQuote)
	}
	if s.Orders == nil {
		s.Orders = make(map[string]configuredOrder)
	}
	if s.Requests == nil {
		s.Requests = make(map[string]configuredRequest)
	}
}

type roomType struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	SortOrder       int             `json:"sort_order"`
	Status          string          `json:"status"`
	DefaultQuantity int             `json:"default_sellable_quantity"`
	Details         json.RawMessage `json:"details"`
	Version         int             `json:"version"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type roomRate struct {
	ID           string    `json:"id"`
	RoomID       string    `json:"room_type_id"`
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	Nightly      int       `json:"base_nightly_cent"`
	Breakfast    int       `json:"breakfast_count"`
	Cancellation string    `json:"cancellation_text"`
	Version      int       `json:"version"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type inventoryDay struct {
	Date      string `json:"stay_date"`
	Override  *int   `json:"sellable_quantity"`
	Effective int    `json:"effective_quantity"`
	Reserved  int    `json:"reserved_quantity"`
	Closed    bool   `json:"closed"`
}

type bookingQuote struct {
	ID          string    `json:"id"`
	ExpiresAt   time.Time `json:"expires_at"`
	RoomID      string    `json:"room_type_id"`
	RoomName    string    `json:"room_name"`
	RateID      string    `json:"rate_id"`
	Checkin     string    `json:"checkin_date"`
	Checkout    string    `json:"checkout_date"`
	Nights      int       `json:"night_count"`
	Nightly     int       `json:"nightly_cent"`
	Total       int       `json:"room_total_cent"`
	Due         int       `json:"amount_due_cent"`
	Usable      []any     `json:"usable_benefits"`
	Selected    []string  `json:"selected_benefits"`
	Deductions  []any     `json:"deductions"`
	Fulfillment []any     `json:"fulfillment_items"`
	Potential   []string  `json:"potential_rewards"`
}

type savedQuote struct {
	Quote       bookingQuote
	MemberID    string
	RoomVersion int
	RateVersion int
	Consumed    bool
}

type bookingHistory struct {
	Status    string    `json:"status"`
	At        time.Time `json:"occurred_at"`
	ActorKind string    `json:"actor_kind"`
	ActorID   string    `json:"actor_id"`
}

type configuredOrder struct {
	ID             string           `json:"id"`
	RoomID         string           `json:"room_type_id"`
	RateID         string           `json:"rate_id"`
	QuoteID        string           `json:"quote_id"`
	Status         string           `json:"status"`
	Checkin        string           `json:"checkin_date"`
	Checkout       string           `json:"checkout_date"`
	Snapshot       bookingQuote     `json:"booking_snapshot"`
	InventoryState string           `json:"inventory_state"`
	Due            int              `json:"amount_due_cent"`
	MemberID       string           `json:"member_id"`
	Arrival        string           `json:"arrival_time"`
	Contact        string           `json:"contact_name"`
	Phone          string           `json:"phone"`
	Remark         string           `json:"remark,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	History        []bookingHistory `json:"history"`
}

type configuredRequest struct {
	Digest    string
	Response  json.RawMessage
	ExpiresAt time.Time
}

type configuredError struct {
	status int
	reason string
	fields map[string]any
}

func (e configuredError) Error() string { return e.reason }

func issue(status int, reason string) error { return configuredError{status: status, reason: reason} }

func invalid(field string) error {
	return configuredError{400, "VALIDATION_ERROR", map[string]any{"field_errors": []map[string]string{{"field": field, "message": "Invalid or missing value"}}}}
}

func checkVersion(got, current int) error {
	if got < 1 {
		return invalid("version")
	}
	if got != current {
		return configuredError{409, "STALE_VERSION", map[string]any{"current_version": current}}
	}
	return nil
}

func newConfiguredID() string {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(data[:])
}

func stayDates(from, to string) ([]string, error) {
	start, startErr := time.Parse(time.DateOnly, from)
	end, endErr := time.Parse(time.DateOnly, to)
	if startErr != nil || endErr != nil || !start.Before(end) || start.AddDate(1, 0, 0).Before(end) {
		return nil, invalid("date_range")
	}
	var dates []string
	for date := start; date.Before(end); date = date.AddDate(0, 0, 1) {
		dates = append(dates, date.Format(time.DateOnly))
	}
	return dates, nil
}

func (s *configuredStore) day(room roomType, date string) inventoryDay {
	day := s.Days[room.ID][date]
	day.Date = date
	day.Effective = room.DefaultQuantity
	if day.Override != nil {
		day.Effective = *day.Override
	}
	if day.Closed {
		day.Effective = 0
	}
	return day
}

func (s *configuredStore) inventoryVersion(roomID string) int {
	return max(1, s.InventoryVersions[roomID])
}

func (s *configuredStore) inventory(room roomType, dates []string) map[string]any {
	items := make([]inventoryDay, 0, len(dates))
	for _, date := range dates {
		items = append(items, s.day(room, date))
	}
	return map[string]any{"items": items, "version": s.inventoryVersion(room.ID)}
}

func (s *configuredStore) activeRates(roomID string) []roomRate {
	items := []roomRate{}
	for _, rate := range s.Rates {
		if rate.RoomID == roomID && rate.Status == "active" {
			items = append(items, rate)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (s *configuredStore) availability(room roomType, dates []string) (int, error) {
	available := int(^uint(0) >> 1)
	for _, date := range dates {
		day := s.day(room, date)
		if day.Closed {
			return 0, issue(409, "ROOM_CLOSED")
		}
		available = min(available, max(0, day.Effective-day.Reserved))
	}
	if available == 0 {
		return 0, issue(409, "ROOM_SOLD_OUT")
	}
	return available, nil
}

func (s *configuredStore) reserve(roomID string, dates []string, delta int) error {
	if s.Days[roomID] == nil {
		s.Days[roomID] = make(map[string]inventoryDay)
	}
	for _, date := range dates {
		day := s.Days[roomID][date]
		if day.Reserved+delta < 0 {
			return errors.New("configured room reservation is inconsistent")
		}
		day.Date = date
		day.Reserved += delta
		s.Days[roomID][date] = day
	}
	return nil
}

func (s *configuredStore) release(order configuredOrder, status, actorKind, actorID string, now time.Time) (configuredOrder, error) {
	dates, err := stayDates(order.Checkin, order.Checkout)
	if err != nil {
		return order, err
	}
	if err := s.reserve(order.RoomID, dates, -1); err != nil {
		return order, err
	}
	order.Status = status
	order.InventoryState = "released"
	order.History = append(order.History, bookingHistory{status, now, actorKind, actorID})
	s.Orders[order.ID] = order
	return order, nil
}
