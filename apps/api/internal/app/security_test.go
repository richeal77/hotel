package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMaskPhoneAndAudit(t *testing.T) {
	if got := maskPhone("13800138001"); got != "138****8001" {
		t.Fatalf("maskPhone = %s", got)
	}
	original := map[string]interface{}{"phone": "13800138001", "nested": map[string]interface{}{"phone": "13900139002"}}
	masked := maskAuditMap(original)
	if masked["phone"] != "138****8001" {
		t.Fatal(masked)
	}
	if original["phone"] != "13800138001" {
		t.Fatal("audit masking mutated source")
	}
	if got := maskPhoneText("联系电话 13800138001"); got != "联系电话 138****8001" {
		t.Fatal(got)
	}
	data := maskStaffData(Page[Member]{Items: []Member{{Phone: "13800138001"}}})
	raw, _ := json.Marshal(data)
	if strings.Contains(string(raw), "13800138001") || !strings.Contains(string(raw), "138****8001") {
		t.Fatal(string(raw))
	}
}

func TestStrongPassword(t *testing.T) {
	for _, weak := range []string{"666666", "lowercase123!", "UPPERCASE123!", "NoDigitsHere!"} {
		if validStrongPassword(weak) {
			t.Fatalf("accepted weak password %q", weak)
		}
	}
	if !validStrongPassword("StrongPass123!") {
		t.Fatal("rejected strong password")
	}
}

func TestRateLimiterWindow(t *testing.T) {
	l := newRateLimiter()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l.now = func() time.Time { return now }
	if !l.allow("ip|login", 2, time.Minute) || !l.allow("ip|login", 2, time.Minute) {
		t.Fatal("initial requests rejected")
	}
	if l.allow("ip|login", 2, time.Minute) {
		t.Fatal("third request allowed")
	}
	now = now.Add(time.Minute)
	if !l.allow("ip|login", 2, time.Minute) {
		t.Fatal("window did not reset")
	}
}

func TestWeChatCodeExchange(t *testing.T) {
	tokenCalls := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body string
		switch r.URL.Path {
		case "/sns/jscode2session":
			if r.URL.Query().Get("js_code") != "login-code" || r.URL.Query().Get("secret") != "secret" {
				t.Error("bad login query")
			}
			body = `{"openid":"openid-real","errcode":0}`
		case "/cgi-bin/token":
			tokenCalls++
			body = `{"access_token":"access","expires_in":7200,"errcode":0}`
		case "/wxa/business/getuserphonenumber":
			if r.URL.Query().Get("access_token") != "access" {
				t.Error("bad access token")
			}
			var payload map[string]string
			_ = json.NewDecoder(r.Body).Decode(&payload)
			if payload["code"] != "phone-code" {
				t.Error("bad phone code")
			}
			body = `{"errcode":0,"phone_info":{"purePhoneNumber":"13800138001"}}`
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	wx := &WeChatClient{AppID: "appid", Secret: "secret", BaseURL: "https://api.weixin.qq.com", Client: client}
	openID, err := wx.ExchangeLogin(context.Background(), "login-code")
	if err != nil || openID != "openid-real" {
		t.Fatalf("login: %s %v", openID, err)
	}
	for range 2 {
		phone, err := wx.ExchangePhone(context.Background(), "phone-code")
		if err != nil || phone != "13800138001" {
			t.Fatalf("phone: %s %v", phone, err)
		}
	}
	if tokenCalls != 1 {
		t.Fatalf("token calls = %d", tokenCalls)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDemoRoleAndLiveRouteIsolation(t *testing.T) {
	repo, err := NewJSONRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	demo := NewRouter(NewService(repo))
	request := httptest.NewRequest("GET", "/api/v1/admin/rules", nil)
	request.Header.Set("Authorization", "Bearer mock_staff_token_stf_frontdesk1")
	response := httptest.NewRecorder()
	demo.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("frontdesk rules status = %d", response.Code)
	}

	live := NewRouter(NewService(repo, WithLiveAuth(&LiveAuth{})))
	for _, path := range []string{"/api/v1/auth/mock-login", "/api/v1/admin/demo/state"} {
		response = httptest.NewRecorder()
		live.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("live route %s status = %d", path, response.Code)
		}
	}
}
