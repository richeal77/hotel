package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestService(t *testing.T) *Service {
	t.Helper()

	repo, err := NewJSONRepository(t.TempDir())
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	return NewService(repo, WithClock(func() time.Time {
		return time.Date(2026, 9, 13, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}))
}

func requireAppCode(t *testing.T, err error, code int) {
	t.Helper()

	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError code %d, got %T %v", code, err, err)
	}
	if appErr.Code != code {
		t.Fatalf("expected code %d, got %d (%s)", code, appErr.Code, appErr.Message)
	}
}

func countLedgers(ledgers []WalletLedger, orderID string, ledgerType WalletLedgerType) int {
	total := 0
	for _, ledger := range ledgers {
		if ledger.OrderID == orderID && ledger.Type == ledgerType {
			total++
		}
	}
	return total
}

func countRewards(rewards []ReferralReward, orderID string) int {
	total := 0
	for _, reward := range rewards {
		if reward.OrderID == orderID {
			total++
		}
	}
	return total
}

func ledgerAmount(ledgers []WalletLedger, orderID string, ledgerType WalletLedgerType) (int, bool) {
	for _, ledger := range ledgers {
		if ledger.OrderID == orderID && ledger.Type == ledgerType {
			return ledger.AmountCent, true
		}
	}
	return 0, false
}

func rewardAmount(rewards []ReferralReward, orderID string) (int, bool) {
	for _, reward := range rewards {
		if reward.OrderID == orderID {
			return reward.RewardAmountCent, true
		}
	}
	return 0, false
}

func TestUpdateBookingStatusEnforcesStateMachine(t *testing.T) {
	ctx := context.Background()
	service := newTestService(t)

	_, err := service.UpdateBookingStatus(ctx, "stf_boss", "ord_001", UpdateBookingStatusRequest{Status: BookingStatusCheckedIn})
	requireAppCode(t, err, CodeConflict)

	order, err := service.UpdateBookingStatus(ctx, "stf_boss", "ord_001", UpdateBookingStatusRequest{Status: BookingStatusConfirmed})
	if err != nil {
		t.Fatalf("confirm order: %v", err)
	}
	if order.Status != BookingStatusConfirmed {
		t.Fatalf("expected confirmed, got %s", order.Status)
	}
}

func TestJoinMemberWithSamePhoneIsIdempotent(t *testing.T) {
	ctx := context.Background()
	service := newTestService(t)

	login, err := service.MockLogin(ctx, MockLoginRequest{Scene: "frontdesk_qr"})
	if err != nil {
		t.Fatalf("mock login: %v", err)
	}
	_, err = service.BindPhone(ctx, login.Member.ID, BindPhoneRequest{Phone: "13900000001", Code: "666666"})
	if err != nil {
		t.Fatalf("bind phone: %v", err)
	}

	req := JoinMemberRequest{
		Nickname:           "新会员",
		Phone:              "13900000001",
		Source:             "frontdesk",
		Scene:              "frontdesk_qr",
		AcceptedPrivacy:    true,
		AcceptedMemberRule: true,
		GiftChoice:         GiftChoiceLaundry,
	}
	firstMember, firstCoupons, err := service.JoinMember(ctx, login.Member.ID, req)
	if err != nil {
		t.Fatalf("first join: %v", err)
	}
	secondMember, secondCoupons, err := service.JoinMember(ctx, login.Member.ID, req)
	if err != nil {
		t.Fatalf("repeat join: %v", err)
	}
	if firstMember.ID != secondMember.ID {
		t.Fatalf("repeat join returned different member: %s vs %s", firstMember.ID, secondMember.ID)
	}
	if len(firstCoupons) != 1 || len(secondCoupons) != 0 {
		t.Fatalf("repeat join should not issue another gift coupon: first=%+v second=%+v", firstCoupons, secondCoupons)
	}

	coupons, err := service.ListMyCoupons(ctx, login.Member.ID, CouponStatusAvailable)
	if err != nil {
		t.Fatalf("list coupons: %v", err)
	}
	newMemberCoupons := 0
	for _, coupon := range coupons {
		if coupon.MemberID == login.Member.ID {
			newMemberCoupons++
		}
	}
	if newMemberCoupons != 1 {
		t.Fatalf("expected one issued coupon after repeated join, got %d", newMemberCoupons)
	}
}

func TestDefaultAcceptancePhoneCanJoinAfterResetAndReceiveLaundryCoupon(t *testing.T) {
	ctx := context.Background()
	service := newTestService(t)

	if _, err := service.ResetDemo(ctx, "stf_boss", "RESET"); err != nil {
		t.Fatalf("reset demo: %v", err)
	}

	login, err := service.MockLogin(ctx, MockLoginRequest{Scene: "frontdesk_qr"})
	if err != nil {
		t.Fatalf("mock login: %v", err)
	}
	if _, err := service.BindPhone(ctx, login.Member.ID, BindPhoneRequest{Phone: "13800138901", Code: "666666"}); err != nil {
		t.Fatalf("bind default acceptance phone: %v", err)
	}

	member, coupons, err := service.JoinMember(ctx, login.Member.ID, JoinMemberRequest{
		Nickname:           "验收住客",
		Phone:              "13800138901",
		Source:             "frontdesk",
		Scene:              "frontdesk_qr",
		AcceptedPrivacy:    true,
		AcceptedMemberRule: true,
		GiftChoice:         GiftChoiceLaundry,
	})
	if err != nil {
		t.Fatalf("join with default acceptance phone: %v", err)
	}
	if member.ID != login.Member.ID || member.Phone != "13800138901" {
		t.Fatalf("join should keep the logged-in member identity, got login=%s member=%+v", login.Member.ID, member)
	}
	if len(coupons) != 1 || coupons[0].Name != "洗衣券 1 次" || coupons[0].Status != CouponStatusAvailable {
		t.Fatalf("expected one available laundry coupon, got %+v", coupons)
	}
}

func TestBindPhoneRejectsPhoneOwnedByAnotherMember(t *testing.T) {
	ctx := context.Background()
	service := newTestService(t)

	login, err := service.MockLogin(ctx, MockLoginRequest{Scene: "frontdesk_qr"})
	if err != nil {
		t.Fatalf("mock login: %v", err)
	}

	_, err = service.BindPhone(ctx, login.Member.ID, BindPhoneRequest{Phone: "13800138001", Code: "666666"})
	requireAppCode(t, err, CodeConflict)
}

func TestJoinMemberRejectsPhoneOwnedByAnotherMember(t *testing.T) {
	ctx := context.Background()
	service := newTestService(t)

	login, err := service.MockLogin(ctx, MockLoginRequest{Scene: "frontdesk_qr"})
	if err != nil {
		t.Fatalf("mock login: %v", err)
	}

	_, _, err = service.JoinMember(ctx, login.Member.ID, JoinMemberRequest{
		Nickname:           "冲突用户",
		Phone:              "13800138001",
		Source:             "frontdesk",
		Scene:              "frontdesk_qr",
		AcceptedPrivacy:    true,
		AcceptedMemberRule: true,
		GiftChoice:         GiftChoiceLaundry,
	})
	requireAppCode(t, err, CodeConflict)
}

func TestMockLoginDoesNotIncreaseTodayMembersUntilJoin(t *testing.T) {
	ctx := context.Background()
	service := newTestService(t)

	before, err := service.GetDashboardSummary(ctx, "2026-09-13")
	if err != nil {
		t.Fatalf("dashboard before login: %v", err)
	}
	login, err := service.MockLogin(ctx, MockLoginRequest{Scene: "frontdesk_qr"})
	if err != nil {
		t.Fatalf("mock login: %v", err)
	}
	afterLogin, err := service.GetDashboardSummary(ctx, "2026-09-13")
	if err != nil {
		t.Fatalf("dashboard after login: %v", err)
	}
	if afterLogin.TodayMembers != before.TodayMembers {
		t.Fatalf("mock-login should not count as joined member: before=%d after=%d", before.TodayMembers, afterLogin.TodayMembers)
	}

	if _, err := service.BindPhone(ctx, login.Member.ID, BindPhoneRequest{Phone: "13900000002", Code: "666666"}); err != nil {
		t.Fatalf("bind phone: %v", err)
	}
	if _, _, err := service.JoinMember(ctx, login.Member.ID, JoinMemberRequest{
		Nickname:           "真正入会",
		Phone:              "13900000002",
		Source:             "frontdesk",
		Scene:              "frontdesk_qr",
		AcceptedPrivacy:    true,
		AcceptedMemberRule: true,
		GiftChoice:         GiftChoiceLaundry,
	}); err != nil {
		t.Fatalf("join member: %v", err)
	}
	afterJoin, err := service.GetDashboardSummary(ctx, "2026-09-13")
	if err != nil {
		t.Fatalf("dashboard after join: %v", err)
	}
	if afterJoin.TodayMembers != before.TodayMembers+1 {
		t.Fatalf("join-member should count as joined member: before=%d after_join=%d", before.TodayMembers, afterJoin.TodayMembers)
	}
}

func TestRedeemCouponIsSingleUseAndBlockedMembersCannotRedeem(t *testing.T) {
	ctx := context.Background()
	service := newTestService(t)

	coupon, err := service.RedeemCoupon(ctx, "stf_boss", "cpn_001", RedeemCouponRequest{OrderID: "ord_001", Remark: "front desk redeem"})
	if err != nil {
		t.Fatalf("redeem coupon: %v", err)
	}
	if coupon.Status != CouponStatusUsed || coupon.UsedOrderID != "ord_001" {
		t.Fatalf("coupon was not marked used for the order: %+v", coupon)
	}

	_, err = service.RedeemCoupon(ctx, "stf_boss", "cpn_001", RedeemCouponRequest{OrderID: "ord_001"})
	requireAppCode(t, err, CodeConflict)

	_, err = service.UpdateMemberStatus(ctx, "stf_boss", "mbr_005", UpdateMemberStatusRequest{Status: MemberStatusBlocked, Reason: "risk"})
	if err != nil {
		t.Fatalf("block member: %v", err)
	}

	_, err = service.RedeemCoupon(ctx, "stf_boss", "cpn_004", RedeemCouponRequest{OrderID: "ord_004"})
	requireAppCode(t, err, CodeConflict)
}

func TestCheckoutGrantsCashbackAndReferralRewardOnce(t *testing.T) {
	ctx := context.Background()
	service := newTestService(t)

	summary, err := service.CreateBooking(ctx, "mbr_005", CreateBookingRequest{
		CheckinDate:  "2026-09-20",
		CheckoutDate: "2026-09-21",
		RoomType:     "标准大床房",
		ArrivalTime:  "20:00",
		ContactName:  "被推荐人",
		Phone:        "13800138005",
		Remark:       "quiet room",
	})
	if err != nil {
		t.Fatalf("create booking: %v", err)
	}

	for _, status := range []BookingStatus{BookingStatusConfirmed, BookingStatusCheckedIn, BookingStatusCheckedOut} {
		_, err = service.UpdateBookingStatus(ctx, "stf_boss", summary.ID, UpdateBookingStatusRequest{
			Status:           status,
			ActualAmountCent: 8000,
		})
		if err != nil {
			t.Fatalf("transition to %s: %v", status, err)
		}
	}

	memberWallet, err := service.GetWallet(ctx, "mbr_005")
	if err != nil {
		t.Fatalf("get member wallet: %v", err)
	}
	if got := countLedgers(memberWallet.Ledgers, summary.ID, WalletLedgerTypeStayCashback); got != 1 {
		t.Fatalf("expected one cashback ledger, got %d in %+v", got, memberWallet.Ledgers)
	}
	if amount, ok := ledgerAmount(memberWallet.Ledgers, summary.ID, WalletLedgerTypeStayCashback); !ok || amount != 240 {
		t.Fatalf("expected 3%% cashback of 8000 cents to be 240, got amount=%d ok=%v ledgers=%+v", amount, ok, memberWallet.Ledgers)
	}

	referrerWallet, err := service.GetWallet(ctx, "mbr_004")
	if err != nil {
		t.Fatalf("get referrer wallet: %v", err)
	}
	if got := countLedgers(referrerWallet.Ledgers, summary.ID, WalletLedgerTypeReferralReward); got != 1 {
		t.Fatalf("expected one referral reward ledger, got %d in %+v", got, referrerWallet.Ledgers)
	}
	if amount, ok := ledgerAmount(referrerWallet.Ledgers, summary.ID, WalletLedgerTypeReferralReward); !ok || amount != 400 {
		t.Fatalf("expected 5%% referral reward of 8000 cents to be 400, got amount=%d ok=%v ledgers=%+v", amount, ok, referrerWallet.Ledgers)
	}

	rewards, err := service.ListReferralRewards(ctx, "", 1, 100)
	if err != nil {
		t.Fatalf("list rewards: %v", err)
	}
	if got := countRewards(rewards.Items, summary.ID); got != 1 {
		t.Fatalf("expected one referral reward row, got %d in %+v", got, rewards.Items)
	}
	if amount, ok := rewardAmount(rewards.Items, summary.ID); !ok || amount != 400 {
		t.Fatalf("expected referral reward row amount 400, got amount=%d ok=%v rewards=%+v", amount, ok, rewards.Items)
	}

	_, err = service.UpdateBookingStatus(ctx, "stf_boss", summary.ID, UpdateBookingStatusRequest{
		Status:           BookingStatusCheckedOut,
		ActualAmountCent: 8000,
	})
	if err != nil {
		t.Fatalf("repeat checkout should be idempotent: %v", err)
	}

	memberWallet, _ = service.GetWallet(ctx, "mbr_005")
	referrerWallet, _ = service.GetWallet(ctx, "mbr_004")
	rewards, _ = service.ListReferralRewards(ctx, "", 1, 100)
	if got := countLedgers(memberWallet.Ledgers, summary.ID, WalletLedgerTypeStayCashback); got != 1 {
		t.Fatalf("repeat checkout duplicated cashback ledger: %d", got)
	}
	if got := countLedgers(referrerWallet.Ledgers, summary.ID, WalletLedgerTypeReferralReward); got != 1 {
		t.Fatalf("repeat checkout duplicated referral wallet ledger: %d", got)
	}
	if got := countRewards(rewards.Items, summary.ID); got != 1 {
		t.Fatalf("repeat checkout duplicated referral reward row: %d", got)
	}
}

func TestReferralRewardIsCappedAtTenYuan(t *testing.T) {
	ctx := context.Background()
	service := newTestService(t)

	summary, err := service.CreateBooking(ctx, "mbr_005", CreateBookingRequest{
		CheckinDate:  "2026-09-22",
		CheckoutDate: "2026-09-23",
		RoomType:     "棋牌房",
		ArrivalTime:  "20:00",
		ContactName:  "被推荐人",
		Phone:        "13800138005",
	})
	if err != nil {
		t.Fatalf("create booking: %v", err)
	}

	for _, status := range []BookingStatus{BookingStatusConfirmed, BookingStatusCheckedIn, BookingStatusCheckedOut} {
		if _, err := service.UpdateBookingStatus(ctx, "stf_boss", summary.ID, UpdateBookingStatusRequest{
			Status:           status,
			ActualAmountCent: 50000,
		}); err != nil {
			t.Fatalf("transition to %s: %v", status, err)
		}
	}

	rewards, err := service.ListReferralRewards(ctx, "", 1, 100)
	if err != nil {
		t.Fatalf("list rewards: %v", err)
	}
	if amount, ok := rewardAmount(rewards.Items, summary.ID); !ok || amount != 1000 {
		t.Fatalf("expected referral reward capped at 1000 cents, got amount=%d ok=%v rewards=%+v", amount, ok, rewards.Items)
	}
}

func TestResetDemoRestoresSeedDataAndRecordsAuditLog(t *testing.T) {
	ctx := context.Background()
	service := newTestService(t)

	_, err := service.CreateBooking(ctx, "mbr_001", CreateBookingRequest{
		CheckinDate:  "2026-10-01",
		CheckoutDate: "2026-10-02",
		RoomType:     "棋牌房",
		ArrivalTime:  "18:00",
		ContactName:  "张三",
		Phone:        "13800138001",
	})
	if err != nil {
		t.Fatalf("create booking: %v", err)
	}

	state, err := service.ResetDemo(ctx, "stf_boss", "RESET")
	if err != nil {
		t.Fatalf("reset demo: %v", err)
	}
	if state.MemberCount != 5 || state.OrderCount != 4 || state.CouponCount != 4 || state.AuditLogCount != 1 {
		t.Fatalf("unexpected reset state: %+v", state)
	}

	state, err = service.GetDemoState(ctx)
	if err != nil {
		t.Fatalf("get demo state: %v", err)
	}
	if state.OrderCount != 4 || state.AuditLogCount != 1 {
		t.Fatalf("reset did not persist seed state: %+v", state)
	}
}

func TestAdjustWalletAppliesManualGrantAndReverse(t *testing.T) {
	ctx := context.Background()
	service := newTestService(t)

	wallet, ledger, err := service.AdjustWallet(ctx, "stf_boss", "mbr_001", AdjustWalletRequest{
		AmountCent: 500,
		Type:       WalletLedgerTypeManualGrant,
		Reason:     "guest compensation",
	})
	if err != nil {
		t.Fatalf("manual grant: %v", err)
	}
	if ledger.Status != WalletLedgerStatusAvailable || wallet.BalanceCent != 740 || wallet.TotalGrantedCent != 740 {
		t.Fatalf("manual grant did not update wallet correctly: wallet=%+v ledger=%+v", wallet, ledger)
	}

	wallet, ledger, err = service.AdjustWallet(ctx, "stf_boss", "mbr_001", AdjustWalletRequest{
		AmountCent: -200,
		Type:       WalletLedgerTypeReverse,
		Reason:     "reverse demo grant",
	})
	if err != nil {
		t.Fatalf("reverse wallet: %v", err)
	}
	if ledger.Status != WalletLedgerStatusReversed || wallet.BalanceCent != 540 || wallet.TotalUsedCent != 200 {
		t.Fatalf("reverse did not update wallet correctly: wallet=%+v ledger=%+v", wallet, ledger)
	}

	_, _, err = service.AdjustWallet(ctx, "stf_frontdesk1", "mbr_001", AdjustWalletRequest{
		AmountCent: 100,
		Type:       WalletLedgerTypeManualGrant,
		Reason:     "not allowed",
	})
	requireAppCode(t, err, CodeForbidden)
}

func TestRouterUsesContractResponseEnvelope(t *testing.T) {
	service := newTestService(t)
	router := NewRouter(service)

	body := bytes.NewBufferString(`{"phone":"13800138001","purpose":"bind_phone"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/send-code", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			MockCode      string `json:"mock_code"`
			ExpireSeconds int    `json:"expire_seconds"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Code != CodeOK || response.Message != "ok" || response.Data.MockCode != "666666" || response.Data.ExpireSeconds != 300 {
		t.Fatalf("unexpected response envelope: %+v", response)
	}
}

func TestJoinMemberResponseReturnsEmptyIssuedCouponsArrayOnRepeatJoin(t *testing.T) {
	ctx := context.Background()
	service := newTestService(t)
	router := NewRouter(service)

	login, err := service.MockLogin(ctx, MockLoginRequest{Scene: "frontdesk_qr"})
	if err != nil {
		t.Fatalf("mock login: %v", err)
	}
	if _, err := service.BindPhone(ctx, login.Member.ID, BindPhoneRequest{Phone: "13900000003", Code: "666666"}); err != nil {
		t.Fatalf("bind phone: %v", err)
	}

	body := `{"nickname":"数组测试","phone":"13900000003","source":"frontdesk","scene":"frontdesk_qr","accepted_privacy":true,"accepted_member_rule":true,"gift_choice":"laundry"}`
	for i := 0; i < 2; i++ {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/members/join", bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+login.MemberToken)
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("join attempt %d expected HTTP 200, got %d body=%s", i+1, recorder.Code, recorder.Body.String())
		}
		if i == 1 {
			var payload struct {
				Code int `json:"code"`
				Data struct {
					IssuedCoupons []CouponSummary `json:"issued_coupons"`
				} `json:"data"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode repeat join response: %v", err)
			}
			if payload.Code != CodeOK || payload.Data.IssuedCoupons == nil || len(payload.Data.IssuedCoupons) != 0 {
				t.Fatalf("repeat join must return issued_coupons as empty array, got code=%d coupons=%#v body=%s", payload.Code, payload.Data.IssuedCoupons, recorder.Body.String())
			}
		}
	}
}

func TestRouterRejectsMissingRequiredQueriesAndInvalidEnums(t *testing.T) {
	service := newTestService(t)
	router := NewRouter(service)

	cases := []struct {
		name   string
		method string
		path   string
		token  string
	}{
		{
			name:   "dashboard date is required",
			method: http.MethodGet,
			path:   "/api/v1/admin/dashboard/summary",
			token:  "mock_staff_token_stf_boss",
		},
		{
			name:   "redemption keyword is required",
			method: http.MethodGet,
			path:   "/api/v1/admin/redemption/lookup",
			token:  "mock_staff_token_stf_boss",
		},
		{
			name:   "coupon status must match enum",
			method: http.MethodGet,
			path:   "/api/v1/coupons/my?status=not_a_status",
			token:  "mock_member_token_mbr_001",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, nil)
			request.Header.Set("Authorization", "Bearer "+tt.token)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected HTTP 400, got %d with body %s", recorder.Code, recorder.Body.String())
			}
			var response APIResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.Code != CodeBadInput || response.Data != nil {
				t.Fatalf("expected bad input envelope with null data, got %+v", response)
			}
		})
	}
}
