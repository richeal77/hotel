package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	repo Repository
	now  func() time.Time
	live *LiveAuth
	mode string
}

type ServiceOption func(*Service)

func NewService(repo Repository, options ...ServiceOption) *Service {
	service := &Service{
		repo: repo,
		now:  time.Now,
		mode: "local-demo",
	}
	for _, option := range options {
		option(service)
	}
	return service
}

func WithLiveAuth(auth *LiveAuth) ServiceOption {
	return func(service *Service) { service.live = auth; service.mode = "integration" }
}

func WithMode(mode string) ServiceOption {
	return func(service *Service) { service.mode = mode }
}

func WithClock(clock func() time.Time) ServiceOption {
	return func(service *Service) {
		service.now = clock
	}
}

func (s *Service) MockLogin(ctx context.Context, req MockLoginRequest) (MockLoginResult, error) {
	var result MockLoginResult
	err := s.repo.Update(func(store *Store) error {
		now := formatTime(s.now())
		member := Member{
			ID:          nextID("mbr", memberIDs(store.Members)),
			OpenID:      nextID("mock_openid", memberOpenIDs(store.Members)),
			Phone:       "",
			Nickname:    "",
			Source:      sourceFromScene(req.Scene),
			SourceScene: req.Scene,
			Status:      MemberStatusActive,
			StayCount:   0,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if req.ReferrerMemberID != "" {
			member.ReferrerMemberID = req.ReferrerMemberID
		}
		store.Members = append(store.Members, member)
		result = MockLoginResult{
			MemberToken: memberToken(member.ID),
			Member:      authProfile(member),
		}
		return nil
	})
	return result, err
}

func (s *Service) LoginByOpenID(ctx context.Context, openID, scene, referrerID string) (MemberAuthProfile, error) {
	var result MemberAuthProfile
	if openID == "" {
		return result, badInput("微信登录失败")
	}
	err := s.repo.Update(func(store *Store) error {
		for _, member := range store.Members {
			if member.OpenID == openID {
				result = authProfile(member)
				return nil
			}
		}
		now := formatTime(s.now())
		member := Member{ID: nextID("mbr", memberIDs(store.Members)), OpenID: openID, Source: sourceFromScene(scene), SourceScene: scene, ReferrerMemberID: referrerID, Status: MemberStatusActive, CreatedAt: now, UpdatedAt: now}
		store.Members = append(store.Members, member)
		result = authProfile(member)
		return nil
	})
	return result, err
}

func (s *Service) SendCode(ctx context.Context, req SendCodeRequest) (map[string]interface{}, error) {
	if strings.TrimSpace(req.Phone) == "" {
		return nil, badInput("手机号不能为空")
	}
	if req.Purpose != "bind_phone" {
		return nil, badInput("验证码用途不支持")
	}
	return map[string]interface{}{"mock_code": "666666", "expire_seconds": 300}, nil
}

func (s *Service) BindPhone(ctx context.Context, memberID string, req BindPhoneRequest) (MemberAuthProfile, error) {
	if req.Code != "666666" {
		return MemberAuthProfile{}, badCode("验证码错误")
	}
	return s.BindVerifiedPhone(ctx, memberID, req.Phone)
}

func (s *Service) BindVerifiedPhone(ctx context.Context, memberID, phone string) (MemberAuthProfile, error) {
	var result MemberAuthProfile
	err := s.repo.Update(func(store *Store) error {
		if strings.TrimSpace(phone) == "" {
			return badInput("手机号不能为空")
		}
		index := findMemberIndex(store.Members, memberID)
		if index < 0 {
			return unauthorized("未登录或 token 失效")
		}
		if phoneOwnedByAnotherMember(store.Members, phone, memberID) {
			return conflict("手机号已绑定其他会员")
		}
		store.Members[index].Phone = phone
		store.Members[index].UpdatedAt = formatTime(s.now())
		result = authProfile(store.Members[index])
		return nil
	})
	return result, err
}

func (s *Service) JoinMember(ctx context.Context, memberID string, req JoinMemberRequest) (Member, []CouponSummary, error) {
	var member Member
	coupons := []CouponSummary{}
	err := s.repo.Update(func(store *Store) error {
		if strings.TrimSpace(req.Nickname) == "" || strings.TrimSpace(req.Phone) == "" || strings.TrimSpace(req.Source) == "" || strings.TrimSpace(req.Scene) == "" {
			return badInput("入会参数不完整")
		}
		if !req.AcceptedPrivacy || !req.AcceptedMemberRule {
			return badInput("请先同意隐私政策和会员规则")
		}
		if !giftAllowed(store.RuleConfig, req.GiftChoice) {
			return badInput("入会礼不支持")
		}

		currentIndex := findMemberIndex(store.Members, memberID)
		if currentIndex < 0 {
			return unauthorized("未登录或 token 失效")
		}
		if phoneOwnedByAnotherMember(store.Members, req.Phone, memberID) {
			return conflict("手机号已绑定其他会员")
		}
		if existingIndex := findMemberByPhone(store.Members, req.Phone); existingIndex >= 0 && store.Members[existingIndex].Phone != "" {
			if existingIndex != currentIndex {
				return conflict("手机号已绑定其他会员")
			}
			if store.Members[existingIndex].Nickname != "" {
				member = store.Members[existingIndex]
				return nil
			}
		}
		if store.Members[currentIndex].Phone == "" || store.Members[currentIndex].Phone != req.Phone {
			return badInput("未绑定手机号不能入会")
		}

		now := formatTime(s.now())
		store.Members[currentIndex].Nickname = req.Nickname
		store.Members[currentIndex].Source = req.Source
		store.Members[currentIndex].SourceScene = req.Scene
		store.Members[currentIndex].Status = MemberStatusActive
		store.Members[currentIndex].UpdatedAt = now
		if store.Members[currentIndex].JoinTime == "" {
			store.Members[currentIndex].JoinTime = now
		}
		if req.ReferrerMemberID != "" {
			if err := bindReferralLocked(store, store.Members[currentIndex].ID, req.ReferrerMemberID, now); err != nil {
				return err
			}
		}

		templateID := giftTemplateID(req.GiftChoice)
		templateIndex := findCouponTemplateIndex(store.CouponTemplates, templateID)
		if templateIndex < 0 || !store.CouponTemplates[templateIndex].Enabled {
			return badInput("入会礼不支持")
		}
		coupon := issueCouponLocked(store, store.Members[currentIndex].ID, store.CouponTemplates[templateIndex], s.now())
		coupons = append(coupons, coupon.Summary())
		member = store.Members[currentIndex]
		appendAuditLocked(store, "system", "member.join", "member", member.ID, "join member", nil, toMap(member), auditIP(ctx), now)
		return nil
	})
	return member, coupons, err
}

func (s *Service) GetMiniHome(ctx context.Context, memberID string) (MiniHome, error) {
	var result MiniHome
	err := s.repo.View(func(store *Store) error {
		member, err := requireMember(store, memberID)
		if err != nil {
			return err
		}
		wallet := walletForMember(store, memberID)
		couponCount := 0
		for _, coupon := range store.Coupons {
			if coupon.MemberID == memberID && coupon.Status == CouponStatusAvailable {
				couponCount++
			}
		}
		bookingCount := 0
		for _, order := range store.Orders {
			if order.MemberID == memberID {
				bookingCount++
			}
		}
		result = MiniHome{
			Hotel: Hotel{
				Name:      "高碑店尚客优",
				Address:   "河北省保定市高碑店市核心商业街，近老火车站",
				Phone:     "0312-0000000",
				Tags:      []string{"停车", "洗衣", "发票", "棋牌"},
				Latitude:  39.326,
				Longitude: 115.873,
			},
			TodayMemberPriceCent: 8000,
			WalletBalanceCent:    wallet.BalanceCent,
			CouponCount:          couponCount,
			BookingCount:         bookingCount,
			ReferralSummary:      referralSummary(store, member.ID),
		}
		return nil
	})
	return result, err
}

func (s *Service) GetMemberMe(ctx context.Context, memberID string) (Member, string, error) {
	var member Member
	err := s.repo.View(func(store *Store) error {
		found, err := requireMember(store, memberID)
		if err != nil {
			return err
		}
		member = found
		return nil
	})
	return member, member.ID, err
}

func (s *Service) ListMyCoupons(ctx context.Context, memberID string, status CouponStatus) ([]Coupon, error) {
	items := []Coupon{}
	err := s.repo.View(func(store *Store) error {
		if status != "" && !validCouponStatus(status) {
			return badInput("优惠券状态不支持")
		}
		if _, err := requireMember(store, memberID); err != nil {
			return err
		}
		for _, coupon := range store.Coupons {
			if coupon.MemberID != memberID {
				continue
			}
			if status != "" && coupon.Status != status {
				continue
			}
			items = append(items, coupon)
		}
		return nil
	})
	return items, err
}

func (s *Service) GetWallet(ctx context.Context, memberID string) (WalletAccount, error) {
	var result WalletAccount
	err := s.repo.View(func(store *Store) error {
		if _, err := requireMember(store, memberID); err != nil {
			return err
		}
		result = walletForMember(store, memberID)
		return nil
	})
	return result, err
}

func (s *Service) CreateBooking(ctx context.Context, memberID string, req CreateBookingRequest) (BookingOrderSummary, error) {
	var summary BookingOrderSummary
	err := s.repo.Update(func(store *Store) error {
		member, err := requireMember(store, memberID)
		if err != nil {
			return err
		}
		if member.Status == MemberStatusBlocked {
			return forbidden("黑名单会员不能下单")
		}
		if strings.TrimSpace(req.CheckinDate) == "" || strings.TrimSpace(req.CheckoutDate) == "" || strings.TrimSpace(req.RoomType) == "" || strings.TrimSpace(req.ArrivalTime) == "" || strings.TrimSpace(req.ContactName) == "" || strings.TrimSpace(req.Phone) == "" {
			return badInput("订房参数不完整")
		}
		amount := roomPriceCent(req.RoomType)
		now := formatTime(s.now())
		order := BookingOrder{
			ID:                  nextID("ord", orderIDs(store.Orders)),
			Status:              BookingStatusPendingConfirm,
			StatusText:          statusText(BookingStatusPendingConfirm),
			MemberID:            member.ID,
			Phone:               req.Phone,
			ContactName:         req.ContactName,
			RoomType:            req.RoomType,
			CheckinDate:         req.CheckinDate,
			CheckoutDate:        req.CheckoutDate,
			ArrivalTime:         req.ArrivalTime,
			Channel:             "miniapp",
			Source:              member.Source,
			Remark:              req.Remark,
			EstimatedAmountCent: amount,
			ActualAmountCent:    amount,
			PaidStatus:          PaidStatusPayAtHotel,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		store.Orders = append(store.Orders, order)
		summary = order.Summary()
		return nil
	})
	return summary, err
}

func (s *Service) ListMyBookings(ctx context.Context, memberID string) ([]BookingOrder, error) {
	items := []BookingOrder{}
	err := s.repo.View(func(store *Store) error {
		if _, err := requireMember(store, memberID); err != nil {
			return err
		}
		for _, order := range store.Orders {
			if order.MemberID == memberID {
				items = append(items, order)
			}
		}
		sortOrders(items)
		return nil
	})
	return items, err
}

func (s *Service) GetReferralPage(ctx context.Context, memberID string) (ReferralPage, error) {
	var page ReferralPage
	err := s.repo.View(func(store *Store) error {
		if _, err := requireMember(store, memberID); err != nil {
			return err
		}
		rewards := []ReferralReward{}
		for _, reward := range store.ReferralRewards {
			if reward.ReferrerMemberID == memberID {
				rewards = append(rewards, reward)
			}
		}
		page = ReferralPage{
			ShareScene:  fmt.Sprintf("referrer=%s", memberID),
			ShareQRText: fmt.Sprintf("demo://miniapp?referrer=%s", memberID),
			Summary:     referralSummary(store, memberID),
			Rewards:     rewards,
		}
		return nil
	})
	return page, err
}

func (s *Service) BindReferral(ctx context.Context, memberID string, referrerID string) (ReferralRelation, error) {
	var relation ReferralRelation
	err := s.repo.Update(func(store *Store) error {
		if strings.TrimSpace(referrerID) == "" {
			return badInput("推荐人不能为空")
		}
		now := formatTime(s.now())
		if err := bindReferralLocked(store, memberID, referrerID, now); err != nil {
			return err
		}
		for _, item := range store.ReferralRelations {
			if item.ReferredMemberID == memberID {
				relation = item
				return nil
			}
		}
		return notFound("推荐关系不存在")
	})
	return relation, err
}

func (s *Service) AdminLogin(ctx context.Context, req AdminLoginRequest) (AdminLoginResult, error) {
	var result AdminLoginResult
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" {
		return result, badInput("账号和密码不能为空")
	}
	err := s.repo.View(func(store *Store) error {
		for _, staff := range store.StaffUsers {
			if staff.Username == req.Username && staff.Password == req.Password {
				result = AdminLoginResult{
					AccessToken: staffToken(staff.ID),
					Staff:       staff.Public(),
				}
				return nil
			}
		}
		return unauthorized("账号或密码错误")
	})
	return result, err
}

func (s *Service) StaffByID(ctx context.Context, staffID string) (StaffUser, error) {
	var result StaffUser
	err := s.repo.View(func(store *Store) error {
		staff, err := requireStaff(store, staffID)
		if err != nil {
			return err
		}
		result = staff.Public()
		return nil
	})
	return result, err
}

func (s *Service) MemberExists(ctx context.Context, memberID string) bool {
	exists := false
	_ = s.repo.View(func(store *Store) error {
		exists = findMemberIndex(store.Members, memberID) >= 0
		return nil
	})
	return exists
}

func (s *Service) StaffExists(ctx context.Context, staffID string) bool {
	exists := false
	_ = s.repo.View(func(store *Store) error {
		exists = findStaffIndex(store.StaffUsers, staffID) >= 0
		return nil
	})
	return exists
}

func (s *Service) GetDashboardSummary(ctx context.Context, date string) (DashboardSummary, error) {
	var result DashboardSummary
	if strings.TrimSpace(date) == "" {
		return result, badInput("日期不能为空")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return result, badInput("日期格式错误")
	}
	err := s.repo.View(func(store *Store) error {
		month := monthPrefix(date)
		sourceCounts := map[string]int{}
		todayMembers := 0
		for _, member := range store.Members {
			if memberIsJoined(member) && strings.HasPrefix(member.JoinTime, date) {
				todayMembers++
				sourceCounts[member.Source]++
			}
		}
		todayOrders := 0
		monthReferralRoomNights := 0
		monthRepeatRoomNights := 0
		for _, order := range store.Orders {
			if strings.HasPrefix(order.CreatedAt, date) {
				todayOrders++
			}
			if strings.HasPrefix(order.CheckinDate, month) && order.Status != BookingStatusCancelled {
				if isReferralMember(store, order.MemberID) {
					monthReferralRoomNights++
				}
				if memberStayCount(store, order.MemberID) > 1 {
					monthRepeatRoomNights++
				}
			}
		}
		todayCouponUsed := 0
		for _, coupon := range store.Coupons {
			if coupon.Status == CouponStatusUsed && strings.HasPrefix(coupon.UsedAt, date) {
				todayCouponUsed++
			}
		}
		monthGranted := 0
		monthUsed := 0
		for _, ledger := range store.WalletLedgers {
			if !strings.HasPrefix(ledger.CreatedAt, month) {
				continue
			}
			if ledger.AmountCent > 0 {
				monthGranted += ledger.AmountCent
			}
			if ledger.AmountCent < 0 || ledger.Type == WalletLedgerTypeRedeem {
				monthUsed += int(math.Abs(float64(ledger.AmountCent)))
			}
		}
		result = DashboardSummary{
			TodayMembers:            todayMembers,
			TodayOrders:             todayOrders,
			TodayCouponUsed:         todayCouponUsed,
			MonthRepeatRoomNights:   monthRepeatRoomNights,
			MonthReferralRoomNights: monthReferralRoomNights,
			MonthWalletGrantedCent:  monthGranted,
			MonthWalletUsedCent:     monthUsed,
			FrontdeskRank:           []FrontdeskMetric{{StaffName: "前台A", JoinedMembers: sourceCounts["frontdesk"]}},
			SourceRatio:             sourceMetrics(sourceCounts),
		}
		return nil
	})
	return result, err
}

func (s *Service) ListBookings(ctx context.Context, status BookingStatus, page int, pageSize int) (Page[BookingOrder], error) {
	var result Page[BookingOrder]
	err := s.repo.View(func(store *Store) error {
		if status != "" && !validBookingStatus(status) {
			return badInput("订单状态不支持")
		}
		items := []BookingOrder{}
		for _, order := range store.Orders {
			if status != "" && order.Status != status {
				continue
			}
			items = append(items, order)
		}
		sortOrders(items)
		result = paginate(items, page, pageSize)
		return nil
	})
	return result, err
}

func (s *Service) UpdateBookingStatus(ctx context.Context, staffID string, orderID string, req UpdateBookingStatusRequest) (BookingOrder, error) {
	var result BookingOrder
	err := s.repo.Update(func(store *Store) error {
		staff, err := requireStaff(store, staffID)
		if err != nil {
			return err
		}
		orderIndex := findOrderIndex(store.Orders, orderID)
		if orderIndex < 0 {
			return notFound("订单不存在")
		}
		if !validBookingStatus(req.Status) {
			return badInput("订单状态不支持")
		}
		current := store.Orders[orderIndex]
		if current.Status == req.Status {
			result = current
			return nil
		}
		if !canTransit(current.Status, req.Status) {
			return conflict("订单状态不允许流转")
		}
		before := toMap(current)
		now := formatTime(s.now())
		current.Status = req.Status
		current.StatusText = statusText(req.Status)
		current.OperatorStaffID = staffID
		current.UpdatedAt = now
		if req.ActualAmountCent > 0 {
			current.ActualAmountCent = req.ActualAmountCent
		}
		if req.Remark != "" {
			current.Remark = req.Remark
		}
		if req.Status == BookingStatusCheckedOut {
			current.PaidStatus = PaidStatusPaid
			grantCheckoutBenefitsLocked(store, &current, staffID, s.now())
			incrementStayLocked(store, current.MemberID, current.CheckoutDate, now)
		}
		store.Orders[orderIndex] = current
		appendAuditLocked(store, staff.ID, "booking.status.update", "booking_order", current.ID, fmt.Sprintf("%s -> %s", before["status"], current.Status), before, toMap(current), auditIP(ctx), now)
		result = current
		return nil
	})
	return result, err
}

func (s *Service) ListMembers(ctx context.Context, keyword string, page int, pageSize int) (Page[Member], error) {
	var result Page[Member]
	err := s.repo.View(func(store *Store) error {
		items := []Member{}
		keyword = strings.TrimSpace(keyword)
		for _, member := range store.Members {
			if keyword == "" || strings.Contains(member.ID, keyword) || strings.Contains(member.Phone, keyword) || strings.Contains(member.Nickname, keyword) {
				items = append(items, member)
			}
		}
		result = paginate(items, page, pageSize)
		return nil
	})
	return result, err
}

func (s *Service) UpdateMemberStatus(ctx context.Context, staffID string, memberID string, req UpdateMemberStatusRequest) (Member, error) {
	var result Member
	err := s.repo.Update(func(store *Store) error {
		staff, err := requireStaff(store, staffID)
		if err != nil {
			return err
		}
		if staff.Role == RoleFrontdesk {
			return forbidden("无权限")
		}
		if req.Status != MemberStatusActive && req.Status != MemberStatusBlocked {
			return badInput("会员状态不支持")
		}
		if strings.TrimSpace(req.Reason) == "" {
			return badInput("原因不能为空")
		}
		index := findMemberIndex(store.Members, memberID)
		if index < 0 {
			return notFound("会员不存在")
		}
		before := toMap(store.Members[index])
		now := formatTime(s.now())
		store.Members[index].Status = req.Status
		store.Members[index].UpdatedAt = now
		result = store.Members[index]
		appendAuditLocked(store, staff.ID, "member.status.update", "member", memberID, req.Reason, before, toMap(result), auditIP(ctx), now)
		return nil
	})
	return result, err
}

func (s *Service) LookupRedemption(ctx context.Context, keyword string) (RedemptionLookup, error) {
	var result RedemptionLookup
	if strings.TrimSpace(keyword) == "" {
		return result, badInput("查询关键词不能为空")
	}
	err := s.repo.View(func(store *Store) error {
		memberIndex := -1
		for i, member := range store.Members {
			if strings.Contains(member.ID, keyword) || strings.Contains(member.Phone, keyword) || strings.Contains(member.Nickname, keyword) {
				memberIndex = i
				break
			}
		}
		if memberIndex < 0 {
			return notFound("会员不存在")
		}
		member := store.Members[memberIndex]
		coupons := []Coupon{}
		for _, coupon := range store.Coupons {
			if coupon.MemberID == member.ID && coupon.Status == CouponStatusAvailable {
				coupons = append(coupons, coupon)
			}
		}
		orders := []BookingOrder{}
		for _, order := range store.Orders {
			if order.MemberID == member.ID {
				orders = append(orders, order)
			}
		}
		sortOrders(orders)
		if len(orders) > 5 {
			orders = orders[:5]
		}
		result = RedemptionLookup{
			Member:            member,
			Coupons:           coupons,
			WalletBalanceCent: walletForMember(store, member.ID).BalanceCent,
			RecentOrders:      orders,
		}
		return nil
	})
	return result, err
}

func (s *Service) RedeemCoupon(ctx context.Context, staffID string, couponID string, req RedeemCouponRequest) (Coupon, error) {
	var result Coupon
	if strings.TrimSpace(req.OrderID) == "" {
		return result, badInput("订单不能为空")
	}
	err := s.repo.Update(func(store *Store) error {
		staff, err := requireStaff(store, staffID)
		if err != nil {
			return err
		}
		couponIndex := findCouponIndex(store.Coupons, couponID)
		if couponIndex < 0 {
			return notFound("优惠券不存在")
		}
		orderIndex := findOrderIndex(store.Orders, req.OrderID)
		if orderIndex < 0 {
			return notFound("订单不存在")
		}
		coupon := store.Coupons[couponIndex]
		if coupon.Status != CouponStatusAvailable {
			return conflict("优惠券不可核销")
		}
		if coupon.MemberID != store.Orders[orderIndex].MemberID {
			return conflict("优惠券和订单会员不一致")
		}
		member, err := requireMember(store, coupon.MemberID)
		if err != nil {
			return err
		}
		if member.Status == MemberStatusBlocked {
			return conflict("黑名单会员不能核销券")
		}
		nowTime := s.now()
		if isExpired(coupon.ExpireAt, nowTime) {
			return conflict("优惠券已过期")
		}
		before := toMap(coupon)
		now := formatTime(nowTime)
		coupon.Status = CouponStatusUsed
		coupon.UsedAt = now
		coupon.UsedOrderID = req.OrderID
		coupon.UpdatedAt = now
		store.Coupons[couponIndex] = coupon
		result = coupon
		appendAuditLocked(store, staff.ID, "coupon.redeem", "coupon", coupon.ID, fmt.Sprintf("redeem for order %s", req.OrderID), before, toMap(coupon), auditIP(ctx), now)
		return nil
	})
	return result, err
}

func (s *Service) AdjustWallet(ctx context.Context, staffID string, memberID string, req AdjustWalletRequest) (WalletAccount, WalletLedger, error) {
	var wallet WalletAccount
	var ledger WalletLedger
	err := s.repo.Update(func(store *Store) error {
		staff, err := requireStaff(store, staffID)
		if err != nil {
			return err
		}
		if staff.Role == RoleFrontdesk {
			return forbidden("无权限")
		}
		if req.Type != WalletLedgerTypeManualGrant && req.Type != WalletLedgerTypeReverse {
			return badInput("调整类型不支持")
		}
		if req.AmountCent == 0 {
			return badInput("调整金额不能为 0")
		}
		if strings.TrimSpace(req.Reason) == "" {
			return badInput("原因不能为空")
		}
		if _, err := requireMember(store, memberID); err != nil {
			return err
		}
		now := formatTime(s.now())
		status := WalletLedgerStatusAvailable
		if req.Type == WalletLedgerTypeReverse {
			status = WalletLedgerStatusReversed
		}
		ledger = WalletLedger{
			ID:              nextID("wlg", walletLedgerIDs(store.WalletLedgers)),
			MemberID:        memberID,
			Type:            req.Type,
			AmountCent:      req.AmountCent,
			Status:          status,
			Reason:          req.Reason,
			OperatorStaffID: staffID,
			CreatedAt:       now,
		}
		store.WalletLedgers = append(store.WalletLedgers, ledger)
		applyWalletLedgerLocked(store, ledger, now)
		wallet = walletForMember(store, memberID)
		appendAuditLocked(store, staff.ID, "wallet.adjust", "wallet", memberID, req.Reason, nil, toMap(ledger), auditIP(ctx), now)
		return nil
	})
	return wallet, ledger, err
}

func (s *Service) ListReferralRewards(ctx context.Context, status RewardStatus, page int, pageSize int) (Page[ReferralReward], error) {
	var result Page[ReferralReward]
	err := s.repo.View(func(store *Store) error {
		if status != "" && !validRewardStatus(status) {
			return badInput("推荐奖励状态不支持")
		}
		items := []ReferralReward{}
		for _, reward := range store.ReferralRewards {
			if status != "" && reward.Status != status {
				continue
			}
			items = append(items, reward)
		}
		result = paginate(items, page, pageSize)
		return nil
	})
	return result, err
}

func (s *Service) GetRules(ctx context.Context) (RuleConfig, error) {
	var result RuleConfig
	err := s.repo.View(func(store *Store) error {
		result = store.RuleConfig
		return nil
	})
	return result, err
}

func (s *Service) UpdateRules(ctx context.Context, staffID string, req RuleConfig) (RuleConfig, error) {
	var result RuleConfig
	err := s.repo.Update(func(store *Store) error {
		staff, err := requireStaff(store, staffID)
		if err != nil {
			return err
		}
		if staff.Role != RoleBoss {
			return forbidden("无权限")
		}
		if req.CashbackRate < 0 || req.ReferralRate < 0 || req.CashbackValidDays <= 0 || req.CashbackOrderLimitCent < 0 || req.ReferralOrderLimitCent < 0 {
			return badInput("规则参数不合法")
		}
		if req.ReferralSettleMode == "" {
			req.ReferralSettleMode = "immediate"
		}
		if req.ReferralSettleMode != "immediate" && req.ReferralSettleMode != "t_plus_3" {
			return badInput("推荐结算模式不支持")
		}
		before := toMap(store.RuleConfig)
		store.RuleConfig = req
		result = req
		now := formatTime(s.now())
		appendAuditLocked(store, staff.ID, "rules.update", "rule_config", "rule_config", "update rules", before, toMap(req), auditIP(ctx), now)
		return nil
	})
	return result, err
}

func (s *Service) ListAuditLogs(ctx context.Context, page int, pageSize int) (Page[AuditLog], error) {
	var result Page[AuditLog]
	err := s.repo.View(func(store *Store) error {
		items := append([]AuditLog(nil), store.AuditLogs...)
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].CreatedAt > items[j].CreatedAt
		})
		result = paginate(items, page, pageSize)
		return nil
	})
	return result, err
}

func (s *Service) GetDemoState(ctx context.Context) (DemoState, error) {
	var result DemoState
	err := s.repo.View(func(store *Store) error {
		result = demoState(store)
		return nil
	})
	return result, err
}

func (s *Service) ResetDemo(ctx context.Context, staffID string, confirmText string) (DemoState, error) {
	if confirmText != "RESET" {
		return DemoState{}, badInput("确认文本错误")
	}

	var state DemoState
	store := SeedStore()
	now := formatTime(s.now())
	staff, err := s.staffFromStore(ctx, staffID)
	if err != nil {
		return DemoState{}, err
	}
	if staff.Role != RoleBoss {
		return DemoState{}, forbidden("无权限")
	}
	appendAuditLocked(&store, staff.ID, "demo.reset", "demo", "demo", "reset demo data", nil, nil, auditIP(ctx), now)
	state = demoState(&store)
	if err := s.repo.Replace(store); err != nil {
		return DemoState{}, err
	}
	return state, nil
}

func (s *Service) AuditStaffEvent(ctx context.Context, staffID, action, summary string) error {
	return s.repo.Update(func(store *Store) error {
		if _, err := requireStaff(store, staffID); err != nil {
			return err
		}
		appendAuditLocked(store, staffID, action, "staff", staffID, summary, nil, nil, auditIP(ctx), formatTime(s.now()))
		return nil
	})
}

func (s *Service) staffFromStore(ctx context.Context, staffID string) (StaffUser, error) {
	var result StaffUser
	err := s.repo.View(func(store *Store) error {
		staff, err := requireStaff(store, staffID)
		if err != nil {
			return err
		}
		result = staff.Public()
		return nil
	})
	return result, err
}

func grantCheckoutBenefitsLocked(store *Store, order *BookingOrder, staffID string, nowTime time.Time) {
	amount := order.ActualAmountCent
	if amount <= 0 {
		amount = order.EstimatedAmountCent
	}
	now := formatTime(nowTime)
	expireAt := endOfDay(nowTime.AddDate(0, 0, store.RuleConfig.CashbackValidDays))
	status := WalletLedgerStatusAvailable
	rewardStatus := RewardStatusAvailable
	settledAt := now
	if store.RuleConfig.ReferralSettleMode == "t_plus_3" {
		status = WalletLedgerStatusFrozen
		rewardStatus = RewardStatusFrozen
		settledAt = ""
	}

	if !walletLedgerExists(store.WalletLedgers, order.ID, order.MemberID, WalletLedgerTypeStayCashback) {
		cashback := cappedRateAmount(amount, store.RuleConfig.CashbackRate, store.RuleConfig.CashbackOrderLimitCent)
		if cashback > 0 {
			ledger := WalletLedger{
				ID:              nextID("wlg", walletLedgerIDs(store.WalletLedgers)),
				MemberID:        order.MemberID,
				OrderID:         order.ID,
				Type:            WalletLedgerTypeStayCashback,
				AmountCent:      cashback,
				Status:          status,
				Reason:          "离店后按房费 3% 发放",
				ExpireAt:        expireAt,
				OperatorStaffID: "system",
				CreatedAt:       now,
			}
			store.WalletLedgers = append(store.WalletLedgers, ledger)
			applyWalletLedgerLocked(store, ledger, now)
		}
	}

	relationIndex := findReferralRelationByReferred(store.ReferralRelations, order.MemberID)
	if relationIndex < 0 || rewardExists(store.ReferralRewards, order.ID) {
		return
	}
	relation := store.ReferralRelations[relationIndex]
	referrer, referrerErr := requireMember(store, relation.ReferrerMemberID)
	referred, referredErr := requireMember(store, relation.ReferredMemberID)
	if referrerErr != nil || referredErr != nil || referrer.Phone == referred.Phone {
		return
	}
	rewardAmount := cappedRateAmount(amount, store.RuleConfig.ReferralRate, store.RuleConfig.ReferralOrderLimitCent)
	if rewardAmount <= 0 {
		return
	}
	reward := ReferralReward{
		ID:               nextID("rwd", referralRewardIDs(store.ReferralRewards)),
		ReferrerMemberID: relation.ReferrerMemberID,
		ReferredMemberID: relation.ReferredMemberID,
		OrderID:          order.ID,
		ActualAmountCent: amount,
		RewardRate:       store.RuleConfig.ReferralRate,
		RewardAmountCent: rewardAmount,
		Status:           rewardStatus,
		CreatedAt:        now,
		SettledAt:        settledAt,
	}
	store.ReferralRewards = append(store.ReferralRewards, reward)
	ledger := WalletLedger{
		ID:              nextID("wlg", walletLedgerIDs(store.WalletLedgers)),
		MemberID:        relation.ReferrerMemberID,
		OrderID:         order.ID,
		Type:            WalletLedgerTypeReferralReward,
		AmountCent:      rewardAmount,
		Status:          status,
		Reason:          "推荐好友离店奖励",
		ExpireAt:        expireAt,
		OperatorStaffID: "system",
		CreatedAt:       now,
	}
	store.WalletLedgers = append(store.WalletLedgers, ledger)
	applyWalletLedgerLocked(store, ledger, now)
}

func bindReferralLocked(store *Store, memberID string, referrerID string, now string) error {
	if memberID == referrerID {
		return conflict("不能推荐自己")
	}
	memberIndex := findMemberIndex(store.Members, memberID)
	if memberIndex < 0 {
		return unauthorized("未登录或 token 失效")
	}
	referrerIndex := findMemberIndex(store.Members, referrerID)
	if referrerIndex < 0 {
		return notFound("推荐人不存在")
	}
	for _, relation := range store.ReferralRelations {
		if relation.ReferredMemberID == memberID {
			return nil
		}
	}
	if store.Members[memberIndex].Phone != "" && store.Members[memberIndex].Phone == store.Members[referrerIndex].Phone {
		return conflict("同手机号不能绑定推荐关系")
	}
	relation := ReferralRelation{
		ID:               nextID("rfr", referralRelationIDs(store.ReferralRelations)),
		ReferrerMemberID: referrerID,
		ReferredMemberID: memberID,
		CreatedAt:        now,
	}
	store.ReferralRelations = append(store.ReferralRelations, relation)
	store.Members[memberIndex].ReferrerMemberID = referrerID
	store.Members[memberIndex].UpdatedAt = now
	return nil
}

func issueCouponLocked(store *Store, memberID string, template CouponTemplate, nowTime time.Time) Coupon {
	now := formatTime(nowTime)
	coupon := Coupon{
		ID:         nextID("cpn", couponIDs(store.Coupons)),
		Name:       template.Name,
		TemplateID: template.ID,
		MemberID:   memberID,
		Type:       template.Type,
		ValueCent:  template.ValueCent,
		Status:     CouponStatusAvailable,
		ReceivedAt: now,
		ExpireAt:   endOfDay(nowTime.AddDate(0, 0, template.ValidDays)),
		UseRule:    template.UseRule,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	store.Coupons = append(store.Coupons, coupon)
	return coupon
}

func applyWalletLedgerLocked(store *Store, ledger WalletLedger, now string) {
	index := findWalletAccountIndex(store.WalletAccounts, ledger.MemberID)
	if index < 0 {
		store.WalletAccounts = append(store.WalletAccounts, WalletAccountRecord{
			MemberID:  ledger.MemberID,
			CreatedAt: now,
			UpdatedAt: now,
		})
		index = len(store.WalletAccounts) - 1
	}
	if ledger.Status == WalletLedgerStatusAvailable || ledger.Type == WalletLedgerTypeReverse {
		store.WalletAccounts[index].BalanceCent += ledger.AmountCent
		if ledger.AmountCent > 0 {
			store.WalletAccounts[index].TotalGrantedCent += ledger.AmountCent
		} else {
			store.WalletAccounts[index].TotalUsedCent += -ledger.AmountCent
		}
	}
	store.WalletAccounts[index].UpdatedAt = now
}

func appendAuditLocked(store *Store, staffID string, action string, targetType string, targetID string, summary string, before map[string]interface{}, after map[string]interface{}, ip string, now string) {
	operatorName := staffName(store, staffID)
	store.AuditLogs = append(store.AuditLogs, AuditLog{
		ID:              nextID("aud", auditLogIDs(store.AuditLogs)),
		OperatorStaffID: staffID,
		OperatorName:    operatorName,
		Action:          action,
		TargetType:      targetType,
		TargetID:        targetID,
		Summary:         maskPhoneText(summary),
		Before:          maskAuditMap(before),
		After:           maskAuditMap(after),
		IP:              ip,
		CreatedAt:       now,
	})
}

func walletForMember(store *Store, memberID string) WalletAccount {
	account := WalletAccount{Ledgers: []WalletLedger{}}
	if index := findWalletAccountIndex(store.WalletAccounts, memberID); index >= 0 {
		record := store.WalletAccounts[index]
		account.BalanceCent = record.BalanceCent
		account.TotalGrantedCent = record.TotalGrantedCent
		account.TotalUsedCent = record.TotalUsedCent
	}
	for _, ledger := range store.WalletLedgers {
		if ledger.MemberID == memberID {
			account.Ledgers = append(account.Ledgers, ledger)
		}
	}
	sort.SliceStable(account.Ledgers, func(i, j int) bool {
		return account.Ledgers[i].CreatedAt > account.Ledgers[j].CreatedAt
	})
	return account
}

func referralSummary(store *Store, memberID string) ReferralSummary {
	summary := ReferralSummary{}
	referred := map[string]bool{}
	for _, relation := range store.ReferralRelations {
		if relation.ReferrerMemberID == memberID {
			referred[relation.ReferredMemberID] = true
		}
	}
	summary.ReferredCount = len(referred)
	summary.ReferredMembers = len(referred)
	orders := map[string]bool{}
	for _, reward := range store.ReferralRewards {
		if reward.ReferrerMemberID != memberID {
			continue
		}
		orders[reward.OrderID] = true
		if reward.Status == RewardStatusPending || reward.Status == RewardStatusFrozen {
			summary.PendingRewardCent += reward.RewardAmountCent
		}
		if reward.Status == RewardStatusAvailable {
			summary.AvailableRewardCent += reward.RewardAmountCent
		}
	}
	summary.ReferredOrders = len(orders)
	return summary
}

func demoState(store *Store) DemoState {
	return DemoState{
		MemberCount:         len(store.Members),
		OrderCount:          len(store.Orders),
		CouponCount:         len(store.Coupons),
		WalletLedgerCount:   len(store.WalletLedgers),
		ReferralRewardCount: len(store.ReferralRewards),
		AuditLogCount:       len(store.AuditLogs),
		SimulatedDate:       demoDate,
		ImmediateSettlement: store.RuleConfig.ReferralSettleMode == "immediate",
	}
}

func requireMember(store *Store, id string) (Member, error) {
	index := findMemberIndex(store.Members, id)
	if index < 0 {
		return Member{}, unauthorized("未登录或 token 失效")
	}
	return store.Members[index], nil
}

func requireStaff(store *Store, id string) (StaffAccount, error) {
	index := findStaffIndex(store.StaffUsers, id)
	if index < 0 {
		return StaffAccount{}, unauthorized("未登录或 token 失效")
	}
	return store.StaffUsers[index], nil
}

func findMemberIndex(items []Member, id string) int {
	for i, item := range items {
		if item.ID == id {
			return i
		}
	}
	return -1
}

func findMemberByPhone(items []Member, phone string) int {
	for i, item := range items {
		if item.Phone == phone {
			return i
		}
	}
	return -1
}

func phoneOwnedByAnotherMember(items []Member, phone string, memberID string) bool {
	for _, item := range items {
		if item.Phone == phone && item.ID != memberID {
			return true
		}
	}
	return false
}

func findStaffIndex(items []StaffAccount, id string) int {
	for i, item := range items {
		if item.ID == id {
			return i
		}
	}
	return -1
}

func findOrderIndex(items []BookingOrder, id string) int {
	for i, item := range items {
		if item.ID == id {
			return i
		}
	}
	return -1
}

func findCouponIndex(items []Coupon, id string) int {
	for i, item := range items {
		if item.ID == id {
			return i
		}
	}
	return -1
}

func findCouponTemplateIndex(items []CouponTemplate, id string) int {
	for i, item := range items {
		if item.ID == id {
			return i
		}
	}
	return -1
}

func findWalletAccountIndex(items []WalletAccountRecord, memberID string) int {
	for i, item := range items {
		if item.MemberID == memberID {
			return i
		}
	}
	return -1
}

func findReferralRelationByReferred(items []ReferralRelation, memberID string) int {
	for i, item := range items {
		if item.ReferredMemberID == memberID {
			return i
		}
	}
	return -1
}

func canTransit(from BookingStatus, to BookingStatus) bool {
	transitions := map[BookingStatus][]BookingStatus{
		BookingStatusPendingConfirm: {BookingStatusConfirmed, BookingStatusCancelled},
		BookingStatusConfirmed:      {BookingStatusCheckedIn, BookingStatusCancelled},
		BookingStatusCheckedIn:      {BookingStatusCheckedOut, BookingStatusCancelled},
	}
	for _, allowed := range transitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

func validBookingStatus(status BookingStatus) bool {
	switch status {
	case BookingStatusPendingConfirm, BookingStatusConfirmed, BookingStatusCheckedIn, BookingStatusCheckedOut, BookingStatusCancelled:
		return true
	default:
		return false
	}
}

func validCouponStatus(status CouponStatus) bool {
	switch status {
	case CouponStatusAvailable, CouponStatusUsed, CouponStatusExpired, CouponStatusVoided:
		return true
	default:
		return false
	}
}

func validRewardStatus(status RewardStatus) bool {
	switch status {
	case RewardStatusPending, RewardStatusFrozen, RewardStatusAvailable, RewardStatusReversed:
		return true
	default:
		return false
	}
}

func statusText(status BookingStatus) string {
	switch status {
	case BookingStatusPendingConfirm:
		return "待前台确认"
	case BookingStatusConfirmed:
		return "已确认"
	case BookingStatusCheckedIn:
		return "已入住"
	case BookingStatusCheckedOut:
		return "已离店"
	case BookingStatusCancelled:
		return "已取消"
	default:
		return string(status)
	}
}

func roomPriceCent(roomType string) int {
	prices := map[string]int{
		"特惠大床房": 6900,
		"大床房":   8000,
		"标准大床房": 8000,
		"标准双床房": 8800,
		"棋牌房":   11800,
	}
	if price, ok := prices[roomType]; ok {
		return price
	}
	return 8000
}

func sourceFromScene(scene string) string {
	switch {
	case strings.Contains(scene, "meituan"):
		return "meituan"
	case strings.Contains(scene, "referrer"):
		return "referral"
	default:
		return "frontdesk"
	}
}

func authProfile(member Member) MemberAuthProfile {
	return MemberAuthProfile{ID: member.ID, OpenID: member.OpenID, Phone: member.Phone, IsPhoneBound: member.Phone != ""}
}

func memberToken(memberID string) string {
	return "mock_member_token_" + memberID
}

func staffToken(staffID string) string {
	return "mock_staff_token_" + staffID
}

func MemberIDFromToken(token string) (string, bool) {
	return strings.CutPrefix(token, "mock_member_token_")
}

func StaffIDFromToken(token string) (string, bool) {
	return strings.CutPrefix(token, "mock_staff_token_")
}

func giftAllowed(config RuleConfig, gift GiftChoice) bool {
	for _, allowed := range config.NewMemberGifts {
		if allowed == gift {
			return true
		}
	}
	return false
}

func giftTemplateID(gift GiftChoice) string {
	switch gift {
	case GiftChoiceLaundry:
		return "cpt_001"
	case GiftChoiceLateCheckout:
		return "cpt_002"
	case GiftChoiceNextStay5:
		return "cpt_003"
	default:
		return ""
	}
}

func cappedRateAmount(amountCent int, rate float64, limitCent int) int {
	amount := int(math.Round(float64(amountCent) * rate))
	if limitCent > 0 && amount > limitCent {
		return limitCent
	}
	return amount
}

func endOfDay(value time.Time) string {
	year, month, day := value.Date()
	location := value.Location()
	return time.Date(year, month, day, 23, 59, 59, 0, location).Format(time.RFC3339)
}

func formatTime(value time.Time) string {
	return value.Format(time.RFC3339)
}

func isExpired(expireAt string, now time.Time) bool {
	if expireAt == "" {
		return false
	}
	parsed, err := time.Parse(time.RFC3339, expireAt)
	if err != nil {
		return false
	}
	return parsed.Before(now)
}

func walletLedgerExists(items []WalletLedger, orderID string, memberID string, ledgerType WalletLedgerType) bool {
	for _, item := range items {
		if item.OrderID == orderID && item.MemberID == memberID && item.Type == ledgerType {
			return true
		}
	}
	return false
}

func rewardExists(items []ReferralReward, orderID string) bool {
	for _, item := range items {
		if item.OrderID == orderID {
			return true
		}
	}
	return false
}

func isReferralMember(store *Store, memberID string) bool {
	return findReferralRelationByReferred(store.ReferralRelations, memberID) >= 0
}

func memberStayCount(store *Store, memberID string) int {
	if index := findMemberIndex(store.Members, memberID); index >= 0 {
		return store.Members[index].StayCount
	}
	return 0
}

func memberIsJoined(member Member) bool {
	return member.Phone != "" && member.Nickname != "" && member.JoinTime != ""
}

func incrementStayLocked(store *Store, memberID string, checkoutDate string, now string) {
	index := findMemberIndex(store.Members, memberID)
	if index < 0 {
		return
	}
	store.Members[index].StayCount++
	store.Members[index].LastStayDate = checkoutDate
	store.Members[index].UpdatedAt = now
}

func sourceMetrics(counts map[string]int) []SourceMetric {
	items := make([]SourceMetric, 0, len(counts))
	for source, count := range counts {
		items = append(items, SourceMetric{Source: source, Count: count})
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Source < items[j].Source
	})
	return items
}

func monthPrefix(date string) string {
	if len(date) >= 7 {
		return date[:7]
	}
	return date
}

func sortOrders(items []BookingOrder) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
}

func paginate[T any](items []T, page int, pageSize int) Page[T] {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return Page[T]{Items: items[start:end], Page: page, PageSize: pageSize, Total: total}
}

func nextID(prefix string, ids []string) string {
	maxID := 0
	searchPrefix := prefix + "_"
	for _, id := range ids {
		if !strings.HasPrefix(id, searchPrefix) {
			continue
		}
		number, err := strconv.Atoi(strings.TrimPrefix(id, searchPrefix))
		if err == nil && number > maxID {
			maxID = number
		}
	}
	return fmt.Sprintf("%s_%03d", prefix, maxID+1)
}

func memberIDs(items []Member) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func memberOpenIDs(items []Member) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.OpenID)
	}
	return ids
}

func orderIDs(items []BookingOrder) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func couponIDs(items []Coupon) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func walletLedgerIDs(items []WalletLedger) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func referralRelationIDs(items []ReferralRelation) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func referralRewardIDs(items []ReferralReward) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func auditLogIDs(items []AuditLog) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func staffName(store *Store, staffID string) string {
	if index := findStaffIndex(store.StaffUsers, staffID); index >= 0 {
		return store.StaffUsers[index].Name
	}
	if staffID == "system" {
		return "system"
	}
	return staffID
}

func toMap(value interface{}) map[string]interface{} {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return result
}
