package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var requestUUID = regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")

func configuredRespond(c *gin.Context, data any, err error) {
	var problem configuredError
	if errors.As(err, &problem) {
		fields := map[string]any{"reason": problem.reason}
		for key, value := range problem.fields {
			fields[key] = value
		}
		c.JSON(problem.status, APIResponse{Code: problem.status * 100, Message: problem.reason, Data: fields})
		return
	}
	respond(c, data, err)
}

func configuredAuth(s *Service, staff bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		token := bearerToken(c.GetHeader("Authorization"))
		var id, key string
		var ok bool
		if staff {
			id, ok = s.ResolveStaffToken(ctx, token)
			ok = ok && s.StaffExists(ctx, id)
			key = "staff_id"
		} else {
			id, ok = s.ResolveMemberToken(ctx, token)
			ok = ok && s.MemberExists(ctx, id)
			key = "member_id"
		}
		if !ok {
			configuredRespond(c, nil, issue(401, "UNAUTHENTICATED"))
			c.Abort()
			return
		}
		c.Set(key, id)
		c.Next()
	}
}

type configuredOperation func(*gin.Context, *Store, map[string]json.RawMessage, time.Time) (any, error)

func configuredHandler(s *Service, write bool, op configuredOperation) gin.HandlerFunc {
	return func(c *gin.Context) {
		var result any
		var body map[string]json.RawMessage
		if !write {
			err := s.repo.View(func(store *Store) error {
				var err error
				result, err = op(c, store, nil, s.now().UTC())
				return err
			})
			configuredRespond(c, result, err)
			return
		}
		key := strings.ToLower(c.GetHeader("Idempotency-Key"))
		if !requestUUID.MatchString(key) {
			configuredRespond(c, nil, invalid("Idempotency-Key"))
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
		decoder := json.NewDecoder(c.Request.Body)
		if err := decoder.Decode(&body); err != nil || body == nil {
			configuredRespond(c, nil, invalid("body"))
			return
		}
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			configuredRespond(c, nil, invalid("body"))
			return
		}
		raw, _ := json.Marshal(body)
		decoder = json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		var normalized any
		if err := decoder.Decode(&normalized); err != nil {
			configuredRespond(c, nil, invalid("body"))
			return
		}
		canonical, _ := json.Marshal(normalized)
		digest := fmt.Sprintf("%x", sha256.Sum256(canonical))
		scope := strings.Join([]string{staffID(c), memberID(c), c.Request.Method, c.FullPath(), c.Param("id"), key}, "|")
		err := s.repo.Update(func(store *Store) error {
			now := s.now().UTC()
			store.Configured.initialize()
			requests := store.Configured.Requests
			if entry, ok := requests[scope]; ok && now.Before(entry.ExpiresAt) {
				if entry.Digest != digest {
					return issue(409, "IDEMPOTENCY_CONFLICT")
				}
				result = entry.Response
				return nil
			}
			var err error
			result, err = op(c, store, body, now)
			if err != nil {
				return err
			}
			raw, err := json.Marshal(result)
			if err != nil {
				return err
			}
			for request, record := range requests {
				if !now.Before(record.ExpiresAt) {
					delete(requests, request)
				}
			}
			requests[scope] = configuredRequest{digest, raw, now.Add(7 * 24 * time.Hour)}
			return nil
		})
		configuredRespond(c, result, err)
	}
}

func decodeConfigured(body map[string]json.RawMessage, target any, required ...string) error {
	for _, field := range required {
		value, ok := body[field]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return invalid(field)
		}
	}
	for field, value := range body {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return invalid(field)
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return invalid("body")
	}
	return nil
}

func configuredPage[T any](c *gin.Context, items []T) Page[T] {
	p, size := page(c), max(1, min(100, pageSize(c)))
	p = max(1, p)
	start := len(items)
	if p-1 <= len(items)/size {
		start = min(len(items), (p-1)*size)
	}
	end := min(start+size, len(items))
	return Page[T]{Items: items[start:end], Page: p, PageSize: size, Total: len(items)}
}

func switchEnabled(name string) bool { return strings.EqualFold(os.Getenv(name), "true") }

func legacyWrite(s *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.mode != "local-demo" {
			configuredRespond(c, nil, issue(409, "FEATURE_DISABLED"))
			c.Abort()
			return
		}
		c.Next()
	}
}

func attachConfigured(api, admin *gin.RouterGroup, s *Service) {
	bookingEnabled := switchEnabled("CONFIGURED_BOOKING_ENABLED")
	api.GET("/mini/capabilities", func(c *gin.Context) {
		respond(c, gin.H{"configured_booking": bookingEnabled, "configured_join": false, "campaign_issue": false}, nil)
	})
	api.GET("/mini/room-types", configuredHandler(s, false, roomSearch(bookingEnabled)))
	member := api.Group("", configuredAuth(s, false))
	member.POST("/bookings/quotes", configuredHandler(s, true, createQuote(bookingEnabled)))
	member.POST("/bookings/configured", configuredHandler(s, true, submitConfigured(bookingEnabled)))
	member.POST("/bookings/:id/cancel", configuredHandler(s, true, cancelConfigured))
	staff := admin.Group("", configuredAuth(s, true))
	staff.GET("/room-types", configuredHandler(s, false, listRooms))
	staff.POST("/room-types", configuredHandler(s, true, writeRoom))
	staff.PATCH("/room-types/:id", configuredHandler(s, true, writeRoom))
	staff.GET("/room-types/:id/rates", configuredHandler(s, false, listRates))
	staff.POST("/room-types/:id/rates", configuredHandler(s, true, writeRate))
	staff.PATCH("/room-rates/:id", configuredHandler(s, true, writeRate))
	staff.GET("/room-types/:id/inventory", configuredHandler(s, false, roomInventory))
	staff.PUT("/room-types/:id/inventory", configuredHandler(s, true, roomInventory))
	staff.GET("/bookings/:id", configuredHandler(s, false, configuredDetail))
	staff.PATCH("/bookings/:id/configured-status", configuredHandler(s, true, checkinConfigured))
}

func listRooms(c *gin.Context, store *Store, _ map[string]json.RawMessage, _ time.Time) (any, error) {
	items := []roomType{}
	for _, room := range store.Configured.Rooms {
		if status := c.Query("status"); status != "" && status != room.Status {
			continue
		}
		if !strings.Contains(strings.ToLower(room.Name), strings.ToLower(c.Query("q"))) {
			continue
		}
		items = append(items, room)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].SortOrder == items[j].SortOrder {
			return items[i].ID < items[j].ID
		}
		return items[i].SortOrder < items[j].SortOrder
	})
	type listItem struct {
		roomType
		ActiveRates []roomRate `json:"active_rates"`
	}
	result := make([]listItem, 0, len(items))
	for _, room := range items {
		result = append(result, listItem{room, store.Configured.activeRates(room.ID)})
	}
	return configuredPage(c, result), nil
}

func listRates(c *gin.Context, store *Store, _ map[string]json.RawMessage, _ time.Time) (any, error) {
	id := c.Param("id")
	if _, ok := store.Configured.Rooms[id]; !ok {
		return nil, issue(404, "NOT_FOUND")
	}
	items := []roomRate{}
	for _, rate := range store.Configured.Rates {
		if rate.RoomID == id {
			items = append(items, rate)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return gin.H{"items": items}, nil
}
