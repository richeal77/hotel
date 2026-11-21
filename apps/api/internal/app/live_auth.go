package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type LiveAuth struct {
	repo *MySQLRepository
	wx   *WeChatClient
}

func NewLiveAuth(repo *MySQLRepository, appID, secret string) (*LiveAuth, error) {
	if appID == "" || secret == "" {
		return nil, errors.New("WECHAT_APP_ID and WECHAT_APP_SECRET are required")
	}
	return &LiveAuth{repo: repo, wx: &WeChatClient{AppID: appID, Secret: secret, BaseURL: "https://api.weixin.qq.com", Client: &http.Client{Timeout: 8 * time.Second}}}, nil
}

func (a *LiveAuth) LoginWeChat(ctx context.Context, service *Service, req WxLoginRequest) (MockLoginResult, error) {
	if strings.TrimSpace(req.WxLoginCode) == "" {
		return MockLoginResult{}, badInput("微信登录 code 不能为空")
	}
	openID, err := a.wx.ExchangeLogin(ctx, req.WxLoginCode)
	if err != nil {
		return MockLoginResult{}, unauthorized("微信登录失败")
	}
	member, err := service.LoginByOpenID(ctx, openID, req.Scene, req.ReferrerMemberID)
	if err != nil {
		return MockLoginResult{}, err
	}
	token, err := a.newSession(ctx, "member", member.ID, 7*24*time.Hour)
	if err != nil {
		return MockLoginResult{}, err
	}
	return MockLoginResult{MemberToken: token, Member: member}, nil
}

func (a *LiveAuth) BindWeChatPhone(ctx context.Context, service *Service, memberID, code string) (MemberAuthProfile, error) {
	if strings.TrimSpace(code) == "" {
		return MemberAuthProfile{}, badInput("手机号授权 code 不能为空")
	}
	phone, err := a.wx.ExchangePhone(ctx, code)
	if err != nil {
		return MemberAuthProfile{}, badCode("手机号授权失败")
	}
	return service.BindVerifiedPhone(ctx, memberID, phone)
}

func (a *LiveAuth) LoginStaff(ctx context.Context, service *Service, req AdminLoginRequest) (AdminLoginResult, error) {
	var account StaffAccount
	err := service.repo.View(func(store *Store) error {
		for _, staff := range store.StaffUsers {
			if staff.Username == req.Username {
				account = staff
				break
			}
		}
		return nil
	})
	if err != nil {
		return AdminLoginResult{}, err
	}
	if account.ID == "" || bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(req.Password)) != nil {
		return AdminLoginResult{}, unauthorized("账号或密码错误")
	}
	token, err := a.newSession(ctx, "staff", account.ID, 12*time.Hour)
	if err != nil {
		return AdminLoginResult{}, err
	}
	if err := service.AuditStaffEvent(ctx, account.ID, "staff.login", "staff session created"); err != nil {
		_ = a.Revoke(ctx, token)
		return AdminLoginResult{}, err
	}
	return AdminLoginResult{AccessToken: token, Staff: account.Public()}, nil
}

func (a *LiveAuth) newSession(ctx context.Context, principalType, principalID string, duration time.Duration) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(bytes)
	hash := sha256.Sum256([]byte(token))
	_, err := a.repo.DB.ExecContext(ctx, "INSERT INTO auth_sessions (token_hash, principal_type, principal_id, expires_at) VALUES (?, ?, ?, DATE_ADD(UTC_TIMESTAMP(6), INTERVAL ? SECOND))", hex.EncodeToString(hash[:]), principalType, principalID, int(duration.Seconds()))
	return token, err
}

func (a *LiveAuth) Resolve(ctx context.Context, token, principalType string) (string, bool) {
	if len(token) != 64 {
		return "", false
	}
	hash := sha256.Sum256([]byte(token))
	var id string
	err := a.repo.DB.QueryRowContext(ctx, "SELECT principal_id FROM auth_sessions WHERE token_hash=? AND principal_type=? AND revoked_at IS NULL AND expires_at>UTC_TIMESTAMP(6)", hex.EncodeToString(hash[:]), principalType).Scan(&id)
	return id, err == nil
}

func (a *LiveAuth) Revoke(ctx context.Context, token string) error {
	if len(token) != 64 {
		return nil
	}
	hash := sha256.Sum256([]byte(token))
	_, err := a.repo.DB.ExecContext(ctx, "UPDATE auth_sessions SET revoked_at=UTC_TIMESTAMP(6) WHERE token_hash=? AND revoked_at IS NULL", hex.EncodeToString(hash[:]))
	return err
}

func validStrongPassword(password string) bool {
	if len(password) < 12 {
		return false
	}
	var upper, lower, digit, special bool
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			upper = true
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= '0' && r <= '9':
			digit = true
		default:
			special = true
		}
	}
	return upper && lower && digit && special
}

func ProvisionStaff(repo *MySQLRepository, username, name string, role Role, password string) error {
	if !validStrongPassword(password) {
		return errors.New("password needs 12+ characters with upper, lower, digit and special characters")
	}
	if username == "" || name == "" {
		return errors.New("username and name are required")
	}
	switch role {
	case RoleBoss, RoleManager, RoleFrontdesk, RoleFinance:
	default:
		return errors.New("invalid staff role")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	var staffID string
	err = repo.updateWithTx(func(store *Store) error {
		for i := range store.StaffUsers {
			if store.StaffUsers[i].Username == username {
				staffID = store.StaffUsers[i].ID
				store.StaffUsers[i].Name, store.StaffUsers[i].Role, store.StaffUsers[i].Password = name, role, string(hash)
				appendAuditLocked(store, "system", "staff.provision", "staff", staffID, "staff credentials updated", nil, nil, "", formatTime(time.Now()))
				return nil
			}
		}
		ids := make([]string, len(store.StaffUsers))
		for i, staff := range store.StaffUsers {
			ids[i] = staff.ID
		}
		staffID = nextID("stf", ids)
		store.StaffUsers = append(store.StaffUsers, StaffAccount{ID: staffID, Username: username, Name: name, Role: role, Password: string(hash)})
		appendAuditLocked(store, "system", "staff.provision", "staff", staffID, "staff account created", nil, nil, "", formatTime(time.Now()))
		return nil
	}, func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "UPDATE auth_sessions SET revoked_at=UTC_TIMESTAMP(6) WHERE principal_type='staff' AND principal_id=? AND revoked_at IS NULL", staffID)
		return err
	})
	return err
}

type WeChatClient struct {
	AppID, Secret, BaseURL string
	Client                 *http.Client
	mu                     sync.Mutex
	accessToken            string
	accessExpiry           time.Time
}

func (w *WeChatClient) getJSON(ctx context.Context, path string, params url.Values, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.BaseURL+path+"?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	return w.do(req, dest)
}

func (w *WeChatClient) do(req *http.Request, dest any) error {
	resp, err := w.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("WeChat HTTP %d", resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(dest); err != nil {
		return err
	}
	return nil
}

func (w *WeChatClient) ExchangeLogin(ctx context.Context, code string) (string, error) {
	var result struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"errcode"`
	}
	err := w.getJSON(ctx, "/sns/jscode2session", url.Values{"appid": {w.AppID}, "secret": {w.Secret}, "js_code": {code}, "grant_type": {"authorization_code"}}, &result)
	if err != nil {
		return "", err
	}
	if result.ErrCode != 0 || result.OpenID == "" {
		return "", errors.New("WeChat rejected login code")
	}
	return result.OpenID, nil
}

func (w *WeChatClient) token(ctx context.Context) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.accessToken != "" && time.Now().Before(w.accessExpiry) {
		return w.accessToken, nil
	}
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
	}
	err := w.getJSON(ctx, "/cgi-bin/token", url.Values{"grant_type": {"client_credential"}, "appid": {w.AppID}, "secret": {w.Secret}}, &result)
	if err != nil {
		return "", err
	}
	if result.ErrCode != 0 || result.AccessToken == "" || result.ExpiresIn < 120 {
		return "", errors.New("WeChat access token unavailable")
	}
	w.accessToken = result.AccessToken
	w.accessExpiry = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second)
	return w.accessToken, nil
}

func (w *WeChatClient) ExchangePhone(ctx context.Context, code string) (string, error) {
	token, err := w.token(ctx)
	if err != nil {
		return "", err
	}
	data, _ := json.Marshal(map[string]string{"code": code})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.BaseURL+"/wxa/business/getuserphonenumber?access_token="+url.QueryEscape(token), strings.NewReader(string(data)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	var result struct {
		ErrCode   int `json:"errcode"`
		PhoneInfo struct {
			PhoneNumber     string `json:"phoneNumber"`
			PurePhoneNumber string `json:"purePhoneNumber"`
		} `json:"phone_info"`
	}
	if err := w.do(req, &result); err != nil {
		return "", err
	}
	if result.ErrCode != 0 || result.PhoneInfo.PurePhoneNumber == "" {
		return "", errors.New("WeChat rejected phone code")
	}
	return result.PhoneInfo.PurePhoneNumber, nil
}

func (s *Service) ResolveMemberToken(ctx context.Context, token string) (string, bool) {
	if s.live != nil {
		return s.live.Resolve(ctx, token, "member")
	}
	return MemberIDFromToken(token)
}
func (s *Service) ResolveStaffToken(ctx context.Context, token string) (string, bool) {
	if s.live != nil {
		return s.live.Resolve(ctx, token, "staff")
	}
	return StaffIDFromToken(token)
}
