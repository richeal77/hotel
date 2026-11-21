package app

import (
	"fmt"
	"net/http"
)

const (
	CodeOK           = 0
	CodeBadInput     = 40000
	CodeBadCode      = 40001
	CodeUnauthorized = 40100
	CodeForbidden    = 40300
	CodeNotFound     = 40400
	CodeConflict     = 40900
	CodeTooMany      = 42900
	CodeInternal     = 50000
)

type AppError struct {
	Code       int
	HTTPStatus int
	Message    string
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

func badInput(message string) *AppError {
	return &AppError{Code: CodeBadInput, HTTPStatus: http.StatusBadRequest, Message: message}
}

func badCode(message string) *AppError {
	return &AppError{Code: CodeBadCode, HTTPStatus: http.StatusBadRequest, Message: message}
}

func unauthorized(message string) *AppError {
	return &AppError{Code: CodeUnauthorized, HTTPStatus: http.StatusUnauthorized, Message: message}
}

func forbidden(message string) *AppError {
	return &AppError{Code: CodeForbidden, HTTPStatus: http.StatusForbidden, Message: message}
}

func notFound(message string) *AppError {
	return &AppError{Code: CodeNotFound, HTTPStatus: http.StatusNotFound, Message: message}
}

func conflict(message string) *AppError {
	return &AppError{Code: CodeConflict, HTTPStatus: http.StatusConflict, Message: message}
}

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type Role string

const (
	RoleBoss      Role = "boss"
	RoleManager   Role = "manager"
	RoleFrontdesk Role = "frontdesk"
	RoleFinance   Role = "finance"
)

type MemberStatus string

const (
	MemberStatusActive  MemberStatus = "active"
	MemberStatusBlocked MemberStatus = "blocked"
)

type BookingStatus string

const (
	BookingStatusPendingConfirm BookingStatus = "pending_confirm"
	BookingStatusConfirmed      BookingStatus = "confirmed"
	BookingStatusCheckedIn      BookingStatus = "checked_in"
	BookingStatusCheckedOut     BookingStatus = "checked_out"
	BookingStatusCancelled      BookingStatus = "cancelled"
)

type PaidStatus string

const (
	PaidStatusPayAtHotel PaidStatus = "pay_at_hotel"
	PaidStatusPending    PaidStatus = "pending"
	PaidStatusPaid       PaidStatus = "paid"
)

type CouponStatus string

const (
	CouponStatusAvailable CouponStatus = "available"
	CouponStatusUsed      CouponStatus = "used"
	CouponStatusExpired   CouponStatus = "expired"
	CouponStatusVoided    CouponStatus = "voided"
)

type CouponType string

const (
	CouponTypeCashDiscount  CouponType = "cash_discount"
	CouponTypeServiceCoupon CouponType = "service_coupon"
	CouponTypeLateCheckout  CouponType = "late_checkout"
)

type WalletLedgerStatus string

const (
	WalletLedgerStatusFrozen    WalletLedgerStatus = "frozen"
	WalletLedgerStatusAvailable WalletLedgerStatus = "available"
	WalletLedgerStatusUsed      WalletLedgerStatus = "used"
	WalletLedgerStatusExpired   WalletLedgerStatus = "expired"
	WalletLedgerStatusReversed  WalletLedgerStatus = "reversed"
)

type WalletLedgerType string

const (
	WalletLedgerTypeStayCashback   WalletLedgerType = "stay_cashback"
	WalletLedgerTypeReferralReward WalletLedgerType = "referral_reward"
	WalletLedgerTypeManualGrant    WalletLedgerType = "manual_grant"
	WalletLedgerTypeRedeem         WalletLedgerType = "redeem"
	WalletLedgerTypeExpire         WalletLedgerType = "expire"
	WalletLedgerTypeReverse        WalletLedgerType = "reverse"
)

type RewardStatus string

const (
	RewardStatusPending   RewardStatus = "pending"
	RewardStatusFrozen    RewardStatus = "frozen"
	RewardStatusAvailable RewardStatus = "available"
	RewardStatusReversed  RewardStatus = "reversed"
)

type GiftChoice string

const (
	GiftChoiceLaundry      GiftChoice = "laundry"
	GiftChoiceLateCheckout GiftChoice = "late_checkout"
	GiftChoiceNextStay5    GiftChoice = "next_stay_5"
)

type MockLoginRequest struct {
	Scene            string `json:"scene"`
	ReferrerMemberID string `json:"referrer_member_id"`
}

type WxLoginRequest struct {
	WxLoginCode      string `json:"wx_login_code"`
	Scene            string `json:"scene"`
	ReferrerMemberID string `json:"referrer_member_id"`
}

type WxBindPhoneRequest struct {
	PhoneCode string `json:"phone_code"`
}

type SendCodeRequest struct {
	Phone   string `json:"phone"`
	Purpose string `json:"purpose"`
}

type BindPhoneRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

type JoinMemberRequest struct {
	Nickname           string     `json:"nickname"`
	Phone              string     `json:"phone"`
	Source             string     `json:"source"`
	Scene              string     `json:"scene"`
	ReferrerMemberID   string     `json:"referrer_member_id"`
	AcceptedPrivacy    bool       `json:"accepted_privacy"`
	AcceptedMemberRule bool       `json:"accepted_member_rule"`
	GiftChoice         GiftChoice `json:"gift_choice"`
}

type AdminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type MemberAuthProfile struct {
	ID           string `json:"id"`
	OpenID       string `json:"openid"`
	Phone        string `json:"phone"`
	IsPhoneBound bool   `json:"is_phone_bound"`
}

type Member struct {
	ID               string       `json:"id"`
	OpenID           string       `json:"openid"`
	Phone            string       `json:"phone"`
	Nickname         string       `json:"nickname"`
	Source           string       `json:"source"`
	SourceScene      string       `json:"source_scene"`
	ReferrerMemberID string       `json:"referrer_member_id,omitempty"`
	Status           MemberStatus `json:"status"`
	JoinTime         string       `json:"join_time"`
	StayCount        int          `json:"stay_count"`
	LastStayDate     string       `json:"last_stay_date,omitempty"`
	CreatedAt        string       `json:"created_at"`
	UpdatedAt        string       `json:"updated_at"`
}

type StaffUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role Role   `json:"role"`
}

type StaffAccount struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     Role   `json:"role"`
}

func (s StaffAccount) Public() StaffUser {
	return StaffUser{ID: s.ID, Name: s.Name, Role: s.Role}
}

type Hotel struct {
	Name      string   `json:"name"`
	Address   string   `json:"address"`
	Phone     string   `json:"phone"`
	Tags      []string `json:"tags"`
	Latitude  float64  `json:"latitude,omitempty"`
	Longitude float64  `json:"longitude,omitempty"`
}

type MiniHome struct {
	Hotel                Hotel           `json:"hotel"`
	TodayMemberPriceCent int             `json:"today_member_price_cent"`
	WalletBalanceCent    int             `json:"wallet_balance_cent"`
	CouponCount          int             `json:"coupon_count"`
	BookingCount         int             `json:"booking_count"`
	ReferralSummary      ReferralSummary `json:"referral_summary"`
}

type CouponTemplate struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Type      CouponType `json:"type"`
	ValueCent int        `json:"value_cent"`
	ValidDays int        `json:"valid_days"`
	UseRule   string     `json:"use_rule"`
	Enabled   bool       `json:"enabled"`
}

type CouponSummary struct {
	ID     string       `json:"id"`
	Name   string       `json:"name"`
	Status CouponStatus `json:"status"`
}

type Coupon struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Status      CouponStatus `json:"status"`
	TemplateID  string       `json:"template_id,omitempty"`
	MemberID    string       `json:"member_id,omitempty"`
	Type        CouponType   `json:"type"`
	ValueCent   int          `json:"value_cent,omitempty"`
	ReceivedAt  string       `json:"received_at,omitempty"`
	ExpireAt    string       `json:"expire_at"`
	UsedAt      string       `json:"used_at,omitempty"`
	UsedOrderID string       `json:"used_order_id,omitempty"`
	UseRule     string       `json:"use_rule"`
	CreatedAt   string       `json:"created_at,omitempty"`
	UpdatedAt   string       `json:"updated_at,omitempty"`
}

func (c Coupon) Summary() CouponSummary {
	return CouponSummary{ID: c.ID, Name: c.Name, Status: c.Status}
}

type WalletAccount struct {
	BalanceCent      int            `json:"balance_cent"`
	TotalGrantedCent int            `json:"total_granted_cent"`
	TotalUsedCent    int            `json:"total_used_cent"`
	Ledgers          []WalletLedger `json:"ledgers"`
}

type WalletAccountRecord struct {
	MemberID         string `json:"member_id"`
	BalanceCent      int    `json:"balance_cent"`
	TotalGrantedCent int    `json:"total_granted_cent"`
	TotalUsedCent    int    `json:"total_used_cent"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type WalletLedger struct {
	ID              string             `json:"id"`
	MemberID        string             `json:"member_id"`
	OrderID         string             `json:"order_id,omitempty"`
	Type            WalletLedgerType   `json:"type"`
	AmountCent      int                `json:"amount_cent"`
	Status          WalletLedgerStatus `json:"status"`
	Reason          string             `json:"reason"`
	ExpireAt        string             `json:"expire_at,omitempty"`
	OperatorStaffID string             `json:"operator_staff_id"`
	CreatedAt       string             `json:"created_at"`
}

type CreateBookingRequest struct {
	CheckinDate  string `json:"checkin_date"`
	CheckoutDate string `json:"checkout_date"`
	RoomType     string `json:"room_type"`
	ArrivalTime  string `json:"arrival_time"`
	ContactName  string `json:"contact_name"`
	Phone        string `json:"phone"`
	Remark       string `json:"remark"`
}

type BookingOrderSummary struct {
	ID         string        `json:"id"`
	Status     BookingStatus `json:"status"`
	StatusText string        `json:"status_text"`
}

type BookingOrder struct {
	ID                  string        `json:"id"`
	Status              BookingStatus `json:"status"`
	StatusText          string        `json:"status_text"`
	MemberID            string        `json:"member_id"`
	Phone               string        `json:"phone"`
	ContactName         string        `json:"contact_name"`
	RoomType            string        `json:"room_type"`
	CheckinDate         string        `json:"checkin_date"`
	CheckoutDate        string        `json:"checkout_date"`
	ArrivalTime         string        `json:"arrival_time"`
	Channel             string        `json:"channel"`
	Source              string        `json:"source"`
	Remark              string        `json:"remark,omitempty"`
	EstimatedAmountCent int           `json:"estimated_amount_cent"`
	ActualAmountCent    int           `json:"actual_amount_cent,omitempty"`
	PaidStatus          PaidStatus    `json:"paid_status"`
	OperatorStaffID     string        `json:"operator_staff_id,omitempty"`
	CreatedAt           string        `json:"created_at"`
	UpdatedAt           string        `json:"updated_at"`
}

func (o BookingOrder) Summary() BookingOrderSummary {
	return BookingOrderSummary{ID: o.ID, Status: o.Status, StatusText: o.StatusText}
}

type UpdateBookingStatusRequest struct {
	Status           BookingStatus `json:"status"`
	ActualAmountCent int           `json:"actual_amount_cent"`
	Remark           string        `json:"remark"`
}

type ReferralSummary struct {
	ReferredCount       int `json:"referred_count"`
	ReferredMembers     int `json:"referred_members,omitempty"`
	ReferredOrders      int `json:"referred_orders,omitempty"`
	PendingRewardCent   int `json:"pending_reward_cent,omitempty"`
	AvailableRewardCent int `json:"available_reward_cent"`
}

type ReferralPage struct {
	ShareScene  string           `json:"share_scene"`
	ShareQRText string           `json:"share_qr_text"`
	Summary     ReferralSummary  `json:"summary"`
	Rewards     []ReferralReward `json:"rewards"`
}

type ReferralRelation struct {
	ID               string `json:"id"`
	ReferrerMemberID string `json:"referrer_member_id"`
	ReferredMemberID string `json:"referred_member_id"`
	CreatedAt        string `json:"created_at"`
}

type ReferralReward struct {
	ID               string       `json:"id"`
	ReferrerMemberID string       `json:"referrer_member_id"`
	ReferredMemberID string       `json:"referred_member_id"`
	OrderID          string       `json:"order_id"`
	ActualAmountCent int          `json:"actual_amount_cent"`
	RewardRate       float64      `json:"reward_rate"`
	RewardAmountCent int          `json:"reward_amount_cent"`
	Status           RewardStatus `json:"status"`
	CreatedAt        string       `json:"created_at"`
	SettledAt        string       `json:"settled_at,omitempty"`
}

type UpdateMemberStatusRequest struct {
	Status MemberStatus `json:"status"`
	Reason string       `json:"reason"`
}

type RedemptionLookup struct {
	Member            Member         `json:"member"`
	Coupons           []Coupon       `json:"coupons"`
	WalletBalanceCent int            `json:"wallet_balance_cent"`
	RecentOrders      []BookingOrder `json:"recent_orders"`
}

type RedeemCouponRequest struct {
	OrderID string `json:"order_id"`
	Remark  string `json:"remark"`
}

type AdjustWalletRequest struct {
	AmountCent int              `json:"amount_cent"`
	Type       WalletLedgerType `json:"type"`
	Reason     string           `json:"reason"`
}

type RuleConfig struct {
	CashbackRate              float64      `json:"cashback_rate"`
	CashbackValidDays         int          `json:"cashback_valid_days"`
	CashbackOrderLimitCent    int          `json:"cashback_order_limit_cent"`
	ReferralRate              float64      `json:"referral_rate"`
	ReferralOrderLimitCent    int          `json:"referral_order_limit_cent"`
	NewMemberGifts            []GiftChoice `json:"new_member_gifts"`
	SpecialPriceCouponAllowed bool         `json:"special_price_coupon_allowed"`
	ReferralSettleMode        string       `json:"referral_settle_mode"`
}

type DashboardSummary struct {
	TodayMembers            int               `json:"today_members"`
	TodayOrders             int               `json:"today_orders"`
	TodayCouponUsed         int               `json:"today_coupon_used"`
	MonthRepeatRoomNights   int               `json:"month_repeat_room_nights"`
	MonthReferralRoomNights int               `json:"month_referral_room_nights"`
	MonthWalletGrantedCent  int               `json:"month_wallet_granted_cent"`
	MonthWalletUsedCent     int               `json:"month_wallet_used_cent"`
	FrontdeskRank           []FrontdeskMetric `json:"frontdesk_rank"`
	SourceRatio             []SourceMetric    `json:"source_ratio"`
}

type FrontdeskMetric struct {
	StaffName     string `json:"staff_name"`
	JoinedMembers int    `json:"joined_members"`
}

type SourceMetric struct {
	Source string `json:"source"`
	Count  int    `json:"count"`
}

type AuditLog struct {
	ID              string                 `json:"id"`
	OperatorStaffID string                 `json:"operator_staff_id"`
	OperatorName    string                 `json:"operator_name,omitempty"`
	Action          string                 `json:"action"`
	TargetType      string                 `json:"target_type"`
	TargetID        string                 `json:"target_id"`
	Summary         string                 `json:"summary"`
	Before          map[string]interface{} `json:"before,omitempty"`
	After           map[string]interface{} `json:"after,omitempty"`
	IP              string                 `json:"ip,omitempty"`
	CreatedAt       string                 `json:"created_at"`
}

type DemoState struct {
	MemberCount         int    `json:"member_count"`
	OrderCount          int    `json:"order_count"`
	CouponCount         int    `json:"coupon_count"`
	WalletLedgerCount   int    `json:"wallet_ledger_count"`
	ReferralRewardCount int    `json:"referral_reward_count"`
	AuditLogCount       int    `json:"audit_log_count"`
	SimulatedDate       string `json:"simulated_date"`
	ImmediateSettlement bool   `json:"immediate_settlement"`
}

type Notification struct {
	ID        string `json:"id"`
	MemberID  string `json:"member_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

type Page[T any] struct {
	Items    []T `json:"items"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type MockLoginResult struct {
	MemberToken string            `json:"member_token"`
	Member      MemberAuthProfile `json:"member"`
}

type AdminLoginResult struct {
	AccessToken string    `json:"access_token"`
	Staff       StaffUser `json:"staff"`
}
