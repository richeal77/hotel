package app

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var shopZone = time.FixedZone("Asia/Shanghai", 8*60*60)

func shopDate(now time.Time) string { return now.In(shopZone).Format(time.DateOnly) }

func roomSearch(enabled bool) configuredOperation {
	return func(c *gin.Context, store *Store, _ map[string]json.RawMessage, now time.Time) (any, error) {
		dates, err := stayDates(c.Query("checkin_date"), c.Query("checkout_date"))
		if err != nil {
			return nil, err
		}
		if dates[0] < shopDate(now) {
			return nil, invalid("checkin_date")
		}
		type publicRate struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Nightly   int    `json:"nightly_cent"`
			Available int    `json:"max_available_rooms"`
		}
		type publicRoom struct {
			ID      string          `json:"id"`
			Name    string          `json:"name"`
			Details json.RawMessage `json:"details"`
			Rates   []publicRate    `json:"rates"`
		}
		items := []publicRoom{}
		rooms := []roomType{}
		if enabled {
			for _, room := range store.Configured.Rooms {
				if room.Status == "active" {
					rooms = append(rooms, room)
				}
			}
		}
		sort.Slice(rooms, func(i, j int) bool {
			if rooms[i].SortOrder == rooms[j].SortOrder {
				return rooms[i].ID < rooms[j].ID
			}
			return rooms[i].SortOrder < rooms[j].SortOrder
		})
		for _, room := range rooms {
			available, err := store.Configured.availability(room, dates)
			if err != nil {
				continue
			}
			rates := []publicRate{}
			for _, rate := range store.Configured.activeRates(room.ID) {
				rates = append(rates, publicRate{rate.ID, rate.Name, rate.Nightly, available})
			}
			if len(rates) > 0 {
				items = append(items, publicRoom{room.ID, room.Name, room.Details, rates})
			}
		}
		return gin.H{"configured_booking": enabled, "items": items}, nil
	}
}

func createQuote(enabled bool) configuredOperation {
	return func(c *gin.Context, store *Store, body map[string]json.RawMessage, now time.Time) (any, error) {
		if !enabled {
			return nil, issue(409, "FEATURE_DISABLED")
		}
		var req struct {
			Rate     string            `json:"rate_id"`
			Checkin  string            `json:"checkin_date"`
			Checkout string            `json:"checkout_date"`
			Selected []json.RawMessage `json:"selected_benefits"`
		}
		if err := decodeConfigured(body, &req, "rate_id", "checkin_date", "checkout_date"); err != nil {
			return nil, err
		}
		if len(req.Selected) > 0 {
			return nil, issue(409, "BENEFIT_UNAVAILABLE")
		}
		dates, err := stayDates(req.Checkin, req.Checkout)
		if err != nil {
			return nil, err
		}
		if req.Checkin < shopDate(now) {
			return nil, invalid("checkin_date")
		}
		member, err := requireMember(store, memberID(c))
		if err != nil || member.Status == MemberStatusBlocked {
			return nil, issue(403, "FORBIDDEN")
		}
		s := &store.Configured
		rate, ok := s.Rates[req.Rate]
		if !ok {
			return nil, issue(404, "NOT_FOUND")
		}
		room := s.Rooms[rate.RoomID]
		if room.Status != "active" || rate.Status != "active" {
			return nil, issue(409, "ROOM_CLOSED")
		}
		if _, err := s.availability(room, dates); err != nil {
			return nil, err
		}
		total := rate.Nightly * len(dates)
		quote := bookingQuote{
			ID: newConfiguredID(), ExpiresAt: now.Add(10 * time.Minute), RoomID: room.ID, RoomName: room.Name, RateID: rate.ID,
			Checkin: req.Checkin, Checkout: req.Checkout, Nights: len(dates), Nightly: rate.Nightly, Total: total, Due: total,
			Usable: []any{}, Selected: []string{}, Deductions: []any{}, Fulfillment: []any{}, Potential: []string{},
		}
		s.Quotes[quote.ID] = savedQuote{Quote: quote, MemberID: member.ID, RoomVersion: room.Version, RateVersion: rate.Version}
		return gin.H{"quote": quote}, nil
	}
}

func submitConfigured(enabled bool) configuredOperation {
	return func(c *gin.Context, store *Store, body map[string]json.RawMessage, now time.Time) (any, error) {
		if !enabled {
			return nil, issue(409, "FEATURE_DISABLED")
		}
		var req struct {
			Quote   string `json:"quote_id"`
			Arrival string `json:"arrival_time"`
			Contact string `json:"contact_name"`
			Phone   string `json:"phone"`
			Remark  string `json:"remark"`
		}
		if err := decodeConfigured(body, &req, "quote_id", "arrival_time", "contact_name", "phone"); err != nil {
			return nil, err
		}
		if _, err := time.Parse("15:04", req.Arrival); err != nil {
			return nil, invalid("arrival_time")
		}
		if strings.TrimSpace(req.Contact) == "" {
			return nil, invalid("contact_name")
		}
		member, err := requireMember(store, memberID(c))
		if err != nil || member.Status == MemberStatusBlocked || member.Phone == "" || member.Phone != req.Phone {
			return nil, issue(403, "FORBIDDEN")
		}
		s := &store.Configured
		saved, ok := s.Quotes[req.Quote]
		if !ok || saved.MemberID != member.ID {
			return nil, issue(404, "NOT_FOUND")
		}
		quote := saved.Quote
		if saved.Consumed {
			return nil, issue(409, "QUOTE_CHANGED")
		}
		if !now.Before(quote.ExpiresAt) || quote.Checkin < shopDate(now) {
			return nil, issue(409, "QUOTE_EXPIRED")
		}
		room, rate := s.Rooms[quote.RoomID], s.Rates[quote.RateID]
		if room.Status != "active" || rate.Status != "active" {
			return nil, issue(409, "ROOM_CLOSED")
		}
		dates, err := stayDates(quote.Checkin, quote.Checkout)
		if err != nil {
			return nil, err
		}
		if _, err := s.availability(room, dates); err != nil {
			return nil, err
		}
		if room.Version != saved.RoomVersion || rate.Version != saved.RateVersion {
			return nil, issue(409, "QUOTE_CHANGED")
		}
		if err := s.reserve(room.ID, dates, 1); err != nil {
			return nil, err
		}
		order := configuredOrder{
			ID: newConfiguredID(), RoomID: room.ID, RateID: rate.ID, QuoteID: quote.ID, Status: "booked",
			Checkin: quote.Checkin, Checkout: quote.Checkout, Snapshot: quote, InventoryState: "held", Due: quote.Due,
			MemberID: member.ID, Arrival: req.Arrival, Contact: req.Contact, Phone: req.Phone, Remark: req.Remark, CreatedAt: now,
			History: []bookingHistory{{"booked", now, "member", member.ID}},
		}
		s.Orders[order.ID] = order
		saved.Consumed = true
		s.Quotes[quote.ID] = saved
		return gin.H{"order": order}, nil
	}
}

func cancelConfigured(c *gin.Context, store *Store, body map[string]json.RawMessage, now time.Time) (any, error) {
	if len(body) != 0 {
		return nil, invalid("body")
	}
	s := &store.Configured
	order, ok := s.Orders[c.Param("id")]
	if !ok || order.MemberID != memberID(c) {
		return nil, issue(404, "NOT_FOUND")
	}
	switch order.Status {
	case "cancelled":
		return gin.H{"order": order}, nil
	case "booked":
		order, err := s.release(order, "cancelled", "member", memberID(c), now)
		return gin.H{"order": order}, err
	default:
		return nil, issue(409, "INVALID_TRANSITION")
	}
}

func checkinConfigured(c *gin.Context, store *Store, body map[string]json.RawMessage, now time.Time) (any, error) {
	var req struct {
		Expected string `json:"expected_status"`
		Status   string `json:"status"`
		Selected []struct {
			CampaignID string `json:"campaign_id"`
			Version    int    `json:"campaign_version"`
		} `json:"selected_campaigns"`
	}
	if err := decodeConfigured(body, &req, "expected_status", "status", "selected_campaigns"); err != nil {
		return nil, err
	}
	if req.Expected != "booked" || req.Status != "checked_in" {
		return nil, invalid("status")
	}
	order, ok := store.Configured.Orders[c.Param("id")]
	if !ok {
		return nil, issue(404, "NOT_FOUND")
	}
	if order.Status != req.Expected {
		return nil, issue(409, "INVALID_TRANSITION")
	}
	if len(req.Selected) > 0 {
		return nil, issue(409, "FEATURE_DISABLED")
	}
	order.Status = "checked_in"
	order.InventoryState = "consumed"
	order.History = append(order.History, bookingHistory{order.Status, now, "staff", staffID(c)})
	store.Configured.Orders[order.ID] = order
	return gin.H{"order": publicStaffOrder(order), "issued_benefits": []any{}, "redeemed_benefits": []any{}, "skipped_campaigns": []any{}}, nil
}

func publicStaffOrder(order configuredOrder) configuredOrder {
	order.Phone = maskPhone(order.Phone)
	return order
}

func configuredDetail(c *gin.Context, store *Store, _ map[string]json.RawMessage, _ time.Time) (any, error) {
	order, ok := store.Configured.Orders[c.Param("id")]
	if !ok {
		return nil, issue(404, "NOT_FOUND")
	}
	return gin.H{"order": publicStaffOrder(order), "status_history": order.History}, nil
}

func configuredOrders(store *Store, member, status string) []configuredOrder {
	items := []configuredOrder{}
	for _, order := range store.Configured.Orders {
		if member != "" && order.MemberID != member || status != "" && order.Status != status {
			continue
		}
		if member == "" {
			order = publicStaffOrder(order)
		}
		items = append(items, order)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items
}

func (s *Service) RunNoShows() error {
	now := s.now().UTC()
	local := now.In(shopZone)
	days := 1
	if local.Hour() < 13 {
		days = 2
	}
	cutoff := local.AddDate(0, 0, -days).Format(time.DateOnly)
	noChanges := errors.New("no configured data to clean up")
	err := s.repo.Update(func(store *Store) error {
		changed := false
		for id, quote := range store.Configured.Quotes {
			if !now.Before(quote.Quote.ExpiresAt.Add(7 * 24 * time.Hour)) {
				delete(store.Configured.Quotes, id)
				changed = true
			}
		}
		for _, order := range store.Configured.Orders {
			if order.Status != "booked" || order.Checkin > cutoff {
				continue
			}
			if _, err := store.Configured.release(order, "no_show", "system", "no_show", now); err != nil {
				return err
			}
			changed = true
		}
		if !changed {
			return noChanges
		}
		return nil
	})
	if errors.Is(err, noChanges) {
		return nil
	}
	return err
}
