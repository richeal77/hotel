package app

const (
	demoDate = "2026-09-12"
	seedNow  = "2026-09-12T10:00:00+08:00"
)

func SeedStore() Store {
	return Store{
		Members: []Member{
			{ID: "mbr_001", OpenID: "mock_openid_001", Phone: "13800138001", Nickname: "张三", Source: "frontdesk", SourceScene: "frontdesk_qr", Status: MemberStatusActive, JoinTime: "2026-09-12T09:00:00+08:00", StayCount: 1, LastStayDate: "2026-09-12", CreatedAt: "2026-09-12T09:00:00+08:00", UpdatedAt: "2026-09-12T09:00:00+08:00"},
			{ID: "mbr_002", OpenID: "mock_openid_002", Phone: "13800138002", Nickname: "李四", Source: "meituan", SourceScene: "meituan_qr", Status: MemberStatusActive, JoinTime: "2026-09-12T09:10:00+08:00", StayCount: 2, LastStayDate: "2026-09-10", CreatedAt: "2026-09-12T09:10:00+08:00", UpdatedAt: "2026-09-12T09:10:00+08:00"},
			{ID: "mbr_003", OpenID: "mock_openid_003", Phone: "13800138003", Nickname: "王五", Source: "frontdesk", SourceScene: "frontdesk_qr", Status: MemberStatusActive, JoinTime: "2026-09-12T09:20:00+08:00", StayCount: 1, LastStayDate: "2026-09-11", CreatedAt: "2026-09-12T09:20:00+08:00", UpdatedAt: "2026-09-12T09:20:00+08:00"},
			{ID: "mbr_004", OpenID: "mock_openid_004", Phone: "13800138004", Nickname: "推荐官", Source: "frontdesk", SourceScene: "frontdesk_qr", Status: MemberStatusActive, JoinTime: "2026-09-12T09:30:00+08:00", StayCount: 3, LastStayDate: "2026-09-08", CreatedAt: "2026-09-12T09:30:00+08:00", UpdatedAt: "2026-09-12T09:30:00+08:00"},
			{ID: "mbr_005", OpenID: "mock_openid_005", Phone: "13800138005", Nickname: "被推荐人", Source: "referral", SourceScene: "referrer=mbr_004", ReferrerMemberID: "mbr_004", Status: MemberStatusActive, JoinTime: "2026-09-12T09:40:00+08:00", StayCount: 1, LastStayDate: "2026-09-12", CreatedAt: "2026-09-12T09:40:00+08:00", UpdatedAt: "2026-09-12T09:40:00+08:00"},
		},
		StaffUsers: []StaffAccount{
			{ID: "stf_boss", Username: "boss", Password: "666666", Name: "老板", Role: RoleBoss},
			{ID: "stf_manager", Username: "manager", Password: "666666", Name: "店长", Role: RoleManager},
			{ID: "stf_frontdesk1", Username: "frontdesk1", Password: "666666", Name: "前台A", Role: RoleFrontdesk},
			{ID: "stf_finance", Username: "finance", Password: "666666", Name: "财务", Role: RoleFinance},
		},
		Orders: []BookingOrder{
			{ID: "ord_001", Status: BookingStatusPendingConfirm, StatusText: statusText(BookingStatusPendingConfirm), MemberID: "mbr_001", Phone: "13800138001", ContactName: "张三", RoomType: "标准大床房", CheckinDate: "2026-09-20", CheckoutDate: "2026-09-21", ArrivalTime: "20:00", Channel: "miniapp", Source: "frontdesk", Remark: "想要安静房间", EstimatedAmountCent: 8000, ActualAmountCent: 8000, PaidStatus: PaidStatusPayAtHotel, CreatedAt: "2026-09-12T10:20:00+08:00", UpdatedAt: "2026-09-12T10:20:00+08:00"},
			{ID: "ord_002", Status: BookingStatusConfirmed, StatusText: statusText(BookingStatusConfirmed), MemberID: "mbr_002", Phone: "13800138002", ContactName: "李四", RoomType: "标准双床房", CheckinDate: "2026-09-20", CheckoutDate: "2026-09-21", ArrivalTime: "19:00", Channel: "miniapp", Source: "meituan", EstimatedAmountCent: 8800, ActualAmountCent: 8800, PaidStatus: PaidStatusPayAtHotel, OperatorStaffID: "stf_boss", CreatedAt: "2026-09-12T10:30:00+08:00", UpdatedAt: "2026-09-12T10:35:00+08:00"},
			{ID: "ord_003", Status: BookingStatusCheckedIn, StatusText: statusText(BookingStatusCheckedIn), MemberID: "mbr_003", Phone: "13800138003", ContactName: "王五", RoomType: "特惠大床房", CheckinDate: "2026-09-12", CheckoutDate: "2026-09-13", ArrivalTime: "18:00", Channel: "miniapp", Source: "frontdesk", EstimatedAmountCent: 6900, ActualAmountCent: 6900, PaidStatus: PaidStatusPayAtHotel, OperatorStaffID: "stf_frontdesk1", CreatedAt: "2026-09-12T10:40:00+08:00", UpdatedAt: "2026-09-12T12:00:00+08:00"},
			{ID: "ord_004", Status: BookingStatusCheckedOut, StatusText: statusText(BookingStatusCheckedOut), MemberID: "mbr_005", Phone: "13800138005", ContactName: "被推荐人", RoomType: "标准大床房", CheckinDate: "2026-09-11", CheckoutDate: "2026-09-12", ArrivalTime: "21:00", Channel: "miniapp", Source: "referral", EstimatedAmountCent: 8000, ActualAmountCent: 8000, PaidStatus: PaidStatusPaid, OperatorStaffID: "stf_boss", CreatedAt: "2026-09-11T10:20:00+08:00", UpdatedAt: "2026-09-12T10:20:00+08:00"},
		},
		CouponTemplates: []CouponTemplate{
			{ID: "cpt_001", Name: "洗衣券 1 次", Type: CouponTypeServiceCoupon, ValueCent: 0, ValidDays: 90, UseRule: "住店期间可用", Enabled: true},
			{ID: "cpt_002", Name: "延迟退房券", Type: CouponTypeLateCheckout, ValueCent: 0, ValidDays: 90, UseRule: "下次入住可用，不可叠加", Enabled: true},
			{ID: "cpt_003", Name: "下次住店 5 元券", Type: CouponTypeCashDiscount, ValueCent: 500, ValidDays: 90, UseRule: "下次入住可用，不可叠加", Enabled: true},
			{ID: "cpt_004", Name: "淡季 8 元券", Type: CouponTypeCashDiscount, ValueCent: 800, ValidDays: 60, UseRule: "淡季入住可用", Enabled: true},
			{ID: "cpt_005", Name: "连住 10 元券", Type: CouponTypeCashDiscount, ValueCent: 1000, ValidDays: 60, UseRule: "连住两晚可用", Enabled: true},
		},
		Coupons: []Coupon{
			{ID: "cpn_001", Name: "洗衣券 1 次", TemplateID: "cpt_001", MemberID: "mbr_001", Type: CouponTypeServiceCoupon, Status: CouponStatusAvailable, ReceivedAt: "2026-09-12T09:00:00+08:00", ExpireAt: "2026-12-11T23:59:59+08:00", UseRule: "住店期间可用", CreatedAt: "2026-09-12T09:00:00+08:00", UpdatedAt: "2026-09-12T09:00:00+08:00"},
			{ID: "cpn_002", Name: "下次住店 5 元券", TemplateID: "cpt_003", MemberID: "mbr_001", Type: CouponTypeCashDiscount, ValueCent: 500, Status: CouponStatusExpired, ReceivedAt: "2026-06-01T09:00:00+08:00", ExpireAt: "2026-08-30T23:59:59+08:00", UseRule: "下次入住可用，不可叠加", CreatedAt: "2026-06-01T09:00:00+08:00", UpdatedAt: "2026-08-30T23:59:59+08:00"},
			{ID: "cpn_003", Name: "淡季 8 元券", TemplateID: "cpt_004", MemberID: "mbr_002", Type: CouponTypeCashDiscount, ValueCent: 800, Status: CouponStatusUsed, ReceivedAt: "2026-09-12T09:10:00+08:00", ExpireAt: "2026-11-11T23:59:59+08:00", UsedAt: "2026-09-12T11:00:00+08:00", UsedOrderID: "ord_002", UseRule: "淡季入住可用", CreatedAt: "2026-09-12T09:10:00+08:00", UpdatedAt: "2026-09-12T11:00:00+08:00"},
			{ID: "cpn_004", Name: "洗衣券 1 次", TemplateID: "cpt_001", MemberID: "mbr_005", Type: CouponTypeServiceCoupon, Status: CouponStatusAvailable, ReceivedAt: "2026-09-12T09:40:00+08:00", ExpireAt: "2026-12-11T23:59:59+08:00", UseRule: "住店期间可用", CreatedAt: "2026-09-12T09:40:00+08:00", UpdatedAt: "2026-09-12T09:40:00+08:00"},
		},
		WalletAccounts: []WalletAccountRecord{
			{MemberID: "mbr_001", BalanceCent: 240, TotalGrantedCent: 240, TotalUsedCent: 0, CreatedAt: seedNow, UpdatedAt: seedNow},
			{MemberID: "mbr_002", BalanceCent: 0, TotalGrantedCent: 200, TotalUsedCent: 0, CreatedAt: seedNow, UpdatedAt: seedNow},
			{MemberID: "mbr_003", BalanceCent: 0, TotalGrantedCent: 100, TotalUsedCent: 100, CreatedAt: seedNow, UpdatedAt: seedNow},
			{MemberID: "mbr_004", BalanceCent: 400, TotalGrantedCent: 400, TotalUsedCent: 0, CreatedAt: seedNow, UpdatedAt: seedNow},
			{MemberID: "mbr_005", BalanceCent: 0, TotalGrantedCent: 0, TotalUsedCent: 0, CreatedAt: seedNow, UpdatedAt: seedNow},
		},
		WalletLedgers: []WalletLedger{
			{ID: "wlg_001", MemberID: "mbr_001", OrderID: "ord_001", Type: WalletLedgerTypeStayCashback, AmountCent: 240, Status: WalletLedgerStatusAvailable, Reason: "离店后按房费 3% 发放", ExpireAt: "2026-12-11T23:59:59+08:00", OperatorStaffID: "system", CreatedAt: "2026-09-12T10:00:00+08:00"},
			{ID: "wlg_002", MemberID: "mbr_002", OrderID: "ord_002", Type: WalletLedgerTypeStayCashback, AmountCent: 200, Status: WalletLedgerStatusFrozen, Reason: "离店后按房费 3% 发放", ExpireAt: "2026-12-11T23:59:59+08:00", OperatorStaffID: "system", CreatedAt: "2026-09-12T10:00:00+08:00"},
			{ID: "wlg_003", MemberID: "mbr_003", OrderID: "ord_003", Type: WalletLedgerTypeRedeem, AmountCent: -100, Status: WalletLedgerStatusUsed, Reason: "住店抵扣使用", OperatorStaffID: "stf_boss", CreatedAt: "2026-09-12T10:00:00+08:00"},
			{ID: "wlg_004", MemberID: "mbr_004", OrderID: "ord_004", Type: WalletLedgerTypeReferralReward, AmountCent: 400, Status: WalletLedgerStatusAvailable, Reason: "推荐好友离店奖励", ExpireAt: "2026-12-11T23:59:59+08:00", OperatorStaffID: "system", CreatedAt: "2026-09-12T10:00:00+08:00"},
		},
		ReferralRelations: []ReferralRelation{
			{ID: "rfr_001", ReferrerMemberID: "mbr_004", ReferredMemberID: "mbr_005", CreatedAt: "2026-09-12T09:40:00+08:00"},
		},
		ReferralRewards: []ReferralReward{
			{ID: "rwd_001", ReferrerMemberID: "mbr_004", ReferredMemberID: "mbr_005", OrderID: "ord_pending_demo", ActualAmountCent: 8000, RewardRate: 0.05, RewardAmountCent: 400, Status: RewardStatusPending, CreatedAt: "2026-09-12T09:50:00+08:00"},
			{ID: "rwd_002", ReferrerMemberID: "mbr_004", ReferredMemberID: "mbr_005", OrderID: "ord_004", ActualAmountCent: 8000, RewardRate: 0.05, RewardAmountCent: 400, Status: RewardStatusAvailable, CreatedAt: "2026-09-12T10:00:00+08:00", SettledAt: "2026-09-12T10:00:00+08:00"},
		},
		AuditLogs:     []AuditLog{},
		Notifications: []Notification{},
		RuleConfig: RuleConfig{
			CashbackRate:              0.03,
			CashbackValidDays:         90,
			CashbackOrderLimitCent:    1000,
			ReferralRate:              0.05,
			ReferralOrderLimitCent:    1000,
			NewMemberGifts:            []GiftChoice{GiftChoiceLaundry, GiftChoiceLateCheckout, GiftChoiceNextStay5},
			SpecialPriceCouponAllowed: true,
			ReferralSettleMode:        "immediate",
		},
	}
}
