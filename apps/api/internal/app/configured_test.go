package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func configuredCall(router http.Handler, method, path, actor, body string, key int) (int, map[string]any) {
	req := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+actor)
	req.Header.Set("Idempotency-Key", fmt.Sprintf("00000000-0000-4000-8000-%012d", key))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var response map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &response)
	return rec.Code, response
}

func TestConfiguredBookingCancellationReleasesEachNight(t *testing.T) {
	testConfiguredBooking(t, "cancel")
}

func TestConfiguredBookingRejectsRoomClosedAfterQuote(t *testing.T) {
	testConfiguredBooking(t, "closed")
}

func TestConfiguredBookingConcurrentLastRoom(t *testing.T) {
	testConfiguredBooking(t, "race")
}

func TestConfiguredQuoteExpiresAtDeadline(t *testing.T) {
	testConfiguredBooking(t, "expired")
}

func TestConfiguredNoShowReleasesAtDeadlineOnce(t *testing.T) {
	testConfiguredBooking(t, "no_show")
}

func TestConfiguredCheckinPreventsNoShowAndCancellation(t *testing.T) {
	testConfiguredBooking(t, "checkin")
}

func TestConfiguredMySQLFirstInitialization(t *testing.T) {
	dsn := os.Getenv("API_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("API_TEST_MYSQL_DSN must point to an empty disposable database")
	}
	repo, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.DB.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var tables int
	if err := repo.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()").Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 0 {
		t.Fatal("test requires an empty database")
	}
	if err := MigrateMySQL(ctx, repo.DB); err != nil {
		t.Fatal(err)
	}
	if err := CheckMySQLSchema(ctx, repo.DB); err != nil {
		t.Fatal(err)
	}
	err = repo.View(func(store *Store) error {
		if len(store.Members) != 0 || len(store.StaffUsers) != 0 || len(store.Orders) != 0 || len(store.Configured.Orders) != 0 || len(store.WalletLedgers) != 0 {
			return fmt.Errorf("first initialization seeded demo data")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Update(func(store *Store) error {
		store.Configured.initialize()
		store.Configured.Rooms["test_room"] = roomType{ID: "test_room", Name: "Test room"}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := MigrateMySQL(ctx, repo.DB); err != nil {
		t.Fatal(err)
	}
	err = repo.View(func(store *Store) error {
		if store.Configured.Rooms["test_room"].Name != "Test room" {
			return fmt.Errorf("rerunning migration lost the room")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func testConfiguredBooking(t *testing.T, scenario string) {
	t.Setenv("CONFIGURED_BOOKING_ENABLED", "true")
	dir := t.TempDir()
	repo, err := NewJSONRepository(dir)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 12, 1, 9, 0, 0, 0, shopZone)
	service := NewService(repo, WithClock(func() time.Time { return now }))
	router := NewRouter(service)
	const staff = "mock_staff_token_stf_boss"
	const member = "mock_member_token_mbr_001"
	call := func(method, path, actor, body string, key int) map[string]any {
		t.Helper()
		status, response := configuredCall(router, method, path, actor, body, key)
		if status != 200 {
			t.Fatalf("%s %s: status %d, response %v", method, path, status, response)
		}
		return response["data"].(map[string]any)
	}
	if items := call("GET", "/admin/room-types", staff, "", 0)["items"].([]any); len(items) != 0 {
		t.Fatalf("configured rooms must start empty, got %v", items)
	}
	room := call("POST", "/admin/room-types", staff, `{"name":"Standard","sort_order":1,"default_sellable_quantity":1,"details":{"photo_urls":[],"beds":[],"amenities":[]}}`, 1)["room"].(map[string]any)
	roomPath := "/admin/room-types/" + room["id"].(string)
	rate := call("POST", roomPath+"/rates", staff, `{"name":"Flexible","base_nightly_cent":7900,"breakfast_count":0,"cancellation_text":""}`, 2)["rate"].(map[string]any)
	call("PATCH", "/admin/room-rates/"+rate["id"].(string), staff, `{"version":1,"status":"active"}`, 3)
	call("PATCH", roomPath, staff, `{"version":1,"status":"active"}`, 4)
	inventoryPath := roomPath + "/inventory?from=2026-12-01&to=2026-12-03"
	readInventory := func(want float64) {
		t.Helper()
		days := call("GET", inventoryPath, staff, "", 0)["items"].([]any)
		if len(days) != 2 {
			t.Fatalf("expected 2 inventory nights, got %v", days)
		}
		for _, day := range days {
			if day.(map[string]any)["reserved_quantity"] != want {
				t.Fatalf("expected reserved quantity %v, got %v", want, day)
			}
		}
	}
	quote := call("POST", "/bookings/quotes", member, fmt.Sprintf(`{"rate_id":%q,"checkin_date":"2026-12-01","checkout_date":"2026-12-03"}`, rate["id"]), 5)["quote"].(map[string]any)
	if quote["amount_due_cent"] != float64(15800) {
		t.Fatalf("unexpected quote: %v", quote)
	}
	body := fmt.Sprintf(`{"quote_id":%q,"arrival_time":"18:00","contact_name":"Guest","phone":"13800138001"}`, quote["id"])
	if scenario == "closed" {
		version := call("GET", inventoryPath, staff, "", 0)["version"]
		call("PUT", roomPath+"/inventory", staff, fmt.Sprintf(`{"version":%.0f,"from":"2026-12-01","to":"2026-12-03","items":[{"stay_date":"2026-12-02","sellable_quantity":null,"closed":true}]}`, version), 6)
		status, response := configuredCall(router, "POST", "/bookings/configured", member, body, 7)
		if status != 409 || response["data"].(map[string]any)["reason"] != "ROOM_CLOSED" {
			t.Fatalf("closed night: %d %v", status, response)
		}
		readInventory(0)
		return
	}
	if scenario == "expired" {
		now = now.Add(10 * time.Minute)
		status, response := configuredCall(router, "POST", "/bookings/configured", member, body, 7)
		if status != 409 || response["data"].(map[string]any)["reason"] != "QUOTE_EXPIRED" {
			t.Fatalf("expired quote: %d %v", status, response)
		}
		readInventory(0)
		return
	}
	if scenario == "race" {
		second := call("POST", "/bookings/quotes", member, fmt.Sprintf(`{"rate_id":%q,"checkin_date":"2026-12-01","checkout_date":"2026-12-03"}`, rate["id"]), 10)["quote"].(map[string]any)
		secondBody := fmt.Sprintf(`{"quote_id":%q,"arrival_time":"18:00","contact_name":"Guest","phone":"13800138001"}`, second["id"])
		results := make(chan int, 2)
		for i, payload := range []string{body, secondBody} {
			go func() {
				status, _ := configuredCall(router, "POST", "/bookings/configured", member, payload, 20+i)
				results <- status
			}()
		}
		first, last := <-results, <-results
		if !((first == 200 && last == 409) || (first == 409 && last == 200)) {
			t.Fatalf("last room results %d %d", first, last)
		}
		readInventory(1)
		return
	}
	order := call("POST", "/bookings/configured", member, body, 7)["order"].(map[string]any)
	readInventory(1)
	replayed := call("POST", "/bookings/configured", member, body, 7)["order"].(map[string]any)
	if replayed["id"] != order["id"] {
		t.Fatalf("booking replay returned a different order: %v", replayed)
	}
	changed := strings.Replace(body, "Guest", "Other guest", 1)
	status, response := configuredCall(router, "POST", "/bookings/configured", member, changed, 7)
	if status != 409 || response["data"].(map[string]any)["reason"] != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("changed booking replay: %d %v", status, response)
	}
	orders, err := NewJSONRepository(dir)
	if err != nil {
		t.Fatal(err)
	}
	router = NewRouter(NewService(orders, WithClock(func() time.Time { return now })))
	readInventory(1)
	orderID := order["id"].(string)
	detailPath := "/admin/bookings/" + orderID
	if got := call("GET", detailPath, staff, "", 0)["order"].(map[string]any)["status"]; got != "booked" {
		t.Fatalf("persisted status %v", got)
	}
	if scenario == "no_show" || scenario == "checkin" {
		service = NewService(orders, WithClock(func() time.Time { return now }))
		now = time.Date(2026, 12, 2, 12, 59, 0, 0, shopZone)
		if scenario == "checkin" {
			data := call("PATCH", detailPath+"/configured-status", staff, `{"expected_status":"booked","status":"checked_in","selected_campaigns":[]}`, 31)
			if data["order"].(map[string]any)["inventory_state"] != "consumed" {
				t.Fatalf("unexpected check-in: %v", data)
			}
			now = now.Add(time.Minute)
			if err := service.RunNoShows(); err != nil {
				t.Fatal(err)
			}
			readInventory(1)
			status, _ := configuredCall(router, "POST", "/bookings/"+orderID+"/cancel", member, `{}`, 32)
			if status != 409 {
				t.Fatalf("checked-in cancellation status %d", status)
			}
			return
		}
		if err := service.RunNoShows(); err != nil {
			t.Fatal(err)
		}
		readInventory(1)
		now = now.Add(time.Minute)
		for i := 0; i < 2; i++ {
			if err := service.RunNoShows(); err != nil {
				t.Fatal(err)
			}
		}
		readInventory(0)
		if got := call("GET", detailPath, staff, "", 0)["order"].(map[string]any)["status"]; got != "no_show" {
			t.Fatalf("no-show status %v", got)
		}
		return
	}
	cancelPath := "/bookings/" + orderID + "/cancel"
	call("POST", cancelPath, member, `{}`, 8)
	readInventory(0)
	call("POST", cancelPath, member, `{}`, 8)
	call("POST", cancelPath, member, `{}`, 9)
	readInventory(0)
	status, response = configuredCall(router, "POST", "/bookings/configured", member, body, 23)
	if status != 409 || response["data"].(map[string]any)["reason"] != "QUOTE_CHANGED" {
		t.Fatalf("cancelled order made its consumed quote reusable: %d %v", status, response)
	}

	// Exercise formal list and write restrictions with the same persisted order.
	t.Setenv("CONFIGURED_BOOKING_ENABLED", "false")
	router = NewRouter(NewService(orders, WithMode("integration"), WithClock(func() time.Time { return now })))
	for _, path := range []string{"/bookings/my", "/admin/bookings?status=cancelled"} {
		actor := member
		if strings.HasPrefix(path, "/admin/") {
			actor = staff
		}
		items := call("GET", path, actor, "", 0)["items"].([]any)
		if len(items) != 1 || items[0].(map[string]any)["id"] != orderID {
			t.Fatalf("%s: unexpected formal order list %v", path, items)
		}
	}
	for _, path := range []string{"/bookings", "/members/join", "/bookings/configured"} {
		status, response := configuredCall(router, "POST", path, member, body, 21)
		if status != 409 || response["data"].(map[string]any)["reason"] != "FEATURE_DISABLED" {
			t.Fatalf("%s: %d %v", path, status, response)
		}
	}
	call("POST", cancelPath, member, `{}`, 22)
}
