package app

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func NewRouter(service *Service) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	api := router.Group("/api/v1")
	api.POST("/auth/mock-login", func(c *gin.Context) {
		var req MockLoginRequest
		if !bindJSON(c, &req) {
			return
		}
		result, err := service.MockLogin(c.Request.Context(), req)
		respond(c, result, err)
	})
	api.POST("/auth/send-code", func(c *gin.Context) {
		var req SendCodeRequest
		if !bindJSON(c, &req) {
			return
		}
		result, err := service.SendCode(c.Request.Context(), req)
		respond(c, result, err)
	})

	member := api.Group("")
	member.Use(memberAuth(service))
	member.POST("/auth/bind-phone", func(c *gin.Context) {
		var req BindPhoneRequest
		if !bindJSON(c, &req) {
			return
		}
		result, err := service.BindPhone(c.Request.Context(), memberID(c), req)
		respond(c, gin.H{"member": result}, err)
	})
	member.POST("/members/join", func(c *gin.Context) {
		var req JoinMemberRequest
		if !bindJSON(c, &req) {
			return
		}
		member, coupons, err := service.JoinMember(c.Request.Context(), memberID(c), req)
		respond(c, gin.H{"member": member, "issued_coupons": coupons}, err)
	})
	member.GET("/mini/home", func(c *gin.Context) {
		result, err := service.GetMiniHome(c.Request.Context(), memberID(c))
		respond(c, result, err)
	})
	member.GET("/members/me", func(c *gin.Context) {
		member, code, err := service.GetMemberMe(c.Request.Context(), memberID(c))
		respond(c, gin.H{"member": member, "member_code": code}, err)
	})
	member.GET("/coupons/my", func(c *gin.Context) {
		result, err := service.ListMyCoupons(c.Request.Context(), memberID(c), CouponStatus(c.Query("status")))
		respond(c, gin.H{"items": result}, err)
	})
	member.GET("/wallet/me", func(c *gin.Context) {
		result, err := service.GetWallet(c.Request.Context(), memberID(c))
		respond(c, result, err)
	})
	member.POST("/bookings", func(c *gin.Context) {
		var req CreateBookingRequest
		if !bindJSON(c, &req) {
			return
		}
		result, err := service.CreateBooking(c.Request.Context(), memberID(c), req)
		respond(c, gin.H{"order": result}, err)
	})
	member.GET("/bookings/my", func(c *gin.Context) {
		result, err := service.ListMyBookings(c.Request.Context(), memberID(c))
		respond(c, gin.H{"items": result}, err)
	})
	member.GET("/referrals/me", func(c *gin.Context) {
		result, err := service.GetReferralPage(c.Request.Context(), memberID(c))
		respond(c, result, err)
	})
	member.POST("/referrals/bind", func(c *gin.Context) {
		var req struct {
			ReferrerMemberID string `json:"referrer_member_id"`
		}
		if !bindJSON(c, &req) {
			return
		}
		result, err := service.BindReferral(c.Request.Context(), memberID(c), req.ReferrerMemberID)
		respond(c, gin.H{"relation": result}, err)
	})

	admin := api.Group("/admin")
	admin.POST("/auth/login", func(c *gin.Context) {
		var req AdminLoginRequest
		if !bindJSON(c, &req) {
			return
		}
		result, err := service.AdminLogin(c.Request.Context(), req)
		respond(c, result, err)
	})

	adminAuthed := admin.Group("")
	adminAuthed.Use(staffAuth(service))
	adminAuthed.GET("/dashboard/summary", func(c *gin.Context) {
		result, err := service.GetDashboardSummary(c.Request.Context(), c.Query("date"))
		respond(c, result, err)
	})
	adminAuthed.GET("/bookings", func(c *gin.Context) {
		result, err := service.ListBookings(c.Request.Context(), BookingStatus(c.Query("status")), page(c), pageSize(c))
		respond(c, result, err)
	})
	adminAuthed.PATCH("/bookings/:order_id/status", func(c *gin.Context) {
		var req UpdateBookingStatusRequest
		if !bindJSON(c, &req) {
			return
		}
		result, err := service.UpdateBookingStatus(c.Request.Context(), staffID(c), c.Param("order_id"), req)
		respond(c, gin.H{"order": result}, err)
	})
	adminAuthed.GET("/members", func(c *gin.Context) {
		result, err := service.ListMembers(c.Request.Context(), c.Query("keyword"), page(c), pageSize(c))
		respond(c, result, err)
	})
	adminAuthed.PATCH("/members/:member_id/status", func(c *gin.Context) {
		var req UpdateMemberStatusRequest
		if !bindJSON(c, &req) {
			return
		}
		result, err := service.UpdateMemberStatus(c.Request.Context(), staffID(c), c.Param("member_id"), req)
		respond(c, gin.H{"member": result}, err)
	})
	adminAuthed.GET("/redemption/lookup", func(c *gin.Context) {
		result, err := service.LookupRedemption(c.Request.Context(), c.Query("keyword"))
		respond(c, result, err)
	})
	adminAuthed.POST("/coupons/:coupon_id/redeem", func(c *gin.Context) {
		var req RedeemCouponRequest
		if !bindJSON(c, &req) {
			return
		}
		result, err := service.RedeemCoupon(c.Request.Context(), staffID(c), c.Param("coupon_id"), req)
		respond(c, gin.H{"coupon": result}, err)
	})
	adminAuthed.POST("/wallets/:member_id/adjust", func(c *gin.Context) {
		var req AdjustWalletRequest
		if !bindJSON(c, &req) {
			return
		}
		wallet, ledger, err := service.AdjustWallet(c.Request.Context(), staffID(c), c.Param("member_id"), req)
		respond(c, gin.H{"wallet": wallet, "ledger": ledger}, err)
	})
	adminAuthed.GET("/referrals/rewards", func(c *gin.Context) {
		result, err := service.ListReferralRewards(c.Request.Context(), RewardStatus(c.Query("status")), page(c), pageSize(c))
		respond(c, result, err)
	})
	adminAuthed.GET("/rules", func(c *gin.Context) {
		result, err := service.GetRules(c.Request.Context())
		respond(c, result, err)
	})
	adminAuthed.PATCH("/rules", func(c *gin.Context) {
		var req RuleConfig
		if !bindJSON(c, &req) {
			return
		}
		result, err := service.UpdateRules(c.Request.Context(), staffID(c), req)
		respond(c, result, err)
	})
	adminAuthed.GET("/audit-logs", func(c *gin.Context) {
		result, err := service.ListAuditLogs(c.Request.Context(), page(c), pageSize(c))
		respond(c, result, err)
	})
	adminAuthed.GET("/demo/state", func(c *gin.Context) {
		result, err := service.GetDemoState(c.Request.Context())
		respond(c, result, err)
	})
	adminAuthed.POST("/demo/reset", func(c *gin.Context) {
		var req struct {
			ConfirmText string `json:"confirm_text"`
		}
		if !bindJSON(c, &req) {
			return
		}
		result, err := service.ResetDemo(c.Request.Context(), staffID(c), req.ConfirmText)
		respond(c, result, err)
	})

	return router
}

func bindJSON(c *gin.Context, dest interface{}) bool {
	if err := c.ShouldBindJSON(dest); err != nil {
		respond(c, nil, badInput("请求参数错误"))
		return false
	}
	return true
}

func respond(c *gin.Context, data interface{}, err error) {
	if err == nil {
		c.JSON(http.StatusOK, APIResponse{Code: CodeOK, Message: "ok", Data: data})
		return
	}

	var appErr *AppError
	if !errors.As(err, &appErr) {
		appErr = &AppError{Code: CodeInternal, HTTPStatus: http.StatusInternalServerError, Message: "服务器错误"}
	}
	c.JSON(appErr.HTTPStatus, APIResponse{Code: appErr.Code, Message: appErr.Message, Data: nil})
}

func memberAuth(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		id, ok := MemberIDFromToken(token)
		if !ok || !service.MemberExists(c.Request.Context(), id) {
			respond(c, nil, unauthorized("未登录或 token 失效"))
			c.Abort()
			return
		}
		c.Set("member_id", id)
		c.Next()
	}
}

func staffAuth(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		id, ok := StaffIDFromToken(token)
		if !ok || !service.StaffExists(c.Request.Context(), id) {
			respond(c, nil, unauthorized("未登录或 token 失效"))
			c.Abort()
			return
		}
		c.Set("staff_id", id)
		c.Next()
	}
}

func bearerToken(header string) string {
	token, ok := strings.CutPrefix(header, "Bearer ")
	if !ok {
		return ""
	}
	return strings.TrimSpace(token)
}

func memberID(c *gin.Context) string {
	value, _ := c.Get("member_id")
	id, _ := value.(string)
	return id
}

func staffID(c *gin.Context) string {
	value, _ := c.Get("staff_id")
	id, _ := value.(string)
	return id
}

func page(c *gin.Context) int {
	return intQuery(c, "page", 1)
}

func pageSize(c *gin.Context) int {
	return intQuery(c, "page_size", 20)
}

func intQuery(c *gin.Context, name string, fallback int) int {
	value := c.Query(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
