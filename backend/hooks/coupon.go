package hooks

import (
	"facet/services"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterCouponHooks registers coupon-related API endpoints
func RegisterCouponHooks(app *pocketbase.PocketBase, rl *services.RateLimitService) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		// ──────────────────────────────────────────────────────────────────
		// ADMIN ENDPOINTS
		// ──────────────────────────────────────────────────────────────────

		// GET /api/admin/coupons - List all coupons
		se.Router.GET("/api/admin/coupons", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			records, err := app.FindRecordsByFilter(
				"coupons",
				"",
				"-created",
				500,
				0,
				nil,
			)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch coupons"})
			}

			coupons := make([]map[string]interface{}, 0, len(records))
			for _, record := range records {
				coupons = append(coupons, couponToMap(record))
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"coupons": coupons,
			})
		})

		// POST /api/admin/coupons - Create coupon
		se.Router.POST("/api/admin/coupons", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			var req struct {
				Code           string      `json:"code"`
				DiscountType   string      `json:"discount_type"`
				DiscountValue  float64     `json:"discount_value"`
				ApplicableType string      `json:"applicable_type"`
				ApplicableIDs  interface{} `json:"applicable_ids"`
				MaxUses        int         `json:"max_uses"`
				ExpiresAt      string      `json:"expires_at"`
				Active         *bool       `json:"active"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			if req.Code == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "code is required"})
			}
			if req.DiscountType != "percentage" && req.DiscountType != "fixed" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "discount_type must be 'percentage' or 'fixed'"})
			}
			if req.DiscountValue < 1 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "discount_value must be at least 1"})
			}
			if req.DiscountType == "percentage" && req.DiscountValue > 100 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "percentage discount cannot exceed 100"})
			}

			collection, err := app.FindCollectionByNameOrId("coupons")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "collection not found"})
			}

			coupon := core.NewRecord(collection)
			coupon.Set("code", strings.ToUpper(strings.TrimSpace(req.Code)))
			coupon.Set("discount_type", req.DiscountType)
			coupon.Set("discount_value", req.DiscountValue)
			if req.ApplicableType != "" {
				coupon.Set("applicable_type", req.ApplicableType)
			} else {
				coupon.Set("applicable_type", "all")
			}
			coupon.Set("applicable_ids", req.ApplicableIDs)
			coupon.Set("max_uses", req.MaxUses)
			coupon.Set("current_uses", 0)
			if req.ExpiresAt != "" {
				coupon.Set("expires_at", req.ExpiresAt)
			}
			if req.Active != nil {
				coupon.Set("active", *req.Active)
			} else {
				coupon.Set("active", true)
			}

			if err := app.Save(coupon); err != nil {
				if strings.Contains(err.Error(), "UNIQUE constraint failed") {
					return e.JSON(http.StatusConflict, map[string]string{"error": "coupon code already exists"})
				}
				app.Logger().Error("Failed to create coupon", "error", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create coupon"})
			}

			return e.JSON(http.StatusCreated, couponToMap(coupon))
		})

		// PATCH /api/admin/coupons/{id} - Update coupon
		se.Router.PATCH("/api/admin/coupons/{id}", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			couponID := e.Request.PathValue("id")
			coupon, err := app.FindRecordById("coupons", couponID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "coupon not found"})
			}

			var req map[string]interface{}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			if code, ok := req["code"].(string); ok {
				coupon.Set("code", strings.ToUpper(strings.TrimSpace(code)))
			}
			if discountType, ok := req["discount_type"].(string); ok {
				if discountType != "percentage" && discountType != "fixed" {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "discount_type must be 'percentage' or 'fixed'"})
				}
				coupon.Set("discount_type", discountType)
			}
			if discountValue, ok := req["discount_value"].(float64); ok {
				if discountValue < 1 {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "discount_value must be at least 1"})
				}
				// Check percentage ceiling using current or updated type
				dt := coupon.GetString("discount_type")
				if newDt, ok := req["discount_type"].(string); ok {
					dt = newDt
				}
				if dt == "percentage" && discountValue > 100 {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "percentage discount cannot exceed 100"})
				}
				coupon.Set("discount_value", discountValue)
			}
			if applicableType, ok := req["applicable_type"].(string); ok {
				coupon.Set("applicable_type", applicableType)
			}
			if applicableIDs, ok := req["applicable_ids"]; ok {
				coupon.Set("applicable_ids", applicableIDs)
			}
			if maxUses, ok := req["max_uses"].(float64); ok {
				coupon.Set("max_uses", int(maxUses))
			}
			if expiresAt, ok := req["expires_at"].(string); ok {
				coupon.Set("expires_at", expiresAt)
			}
			if active, ok := req["active"].(bool); ok {
				coupon.Set("active", active)
			}

			if err := app.Save(coupon); err != nil {
				app.Logger().Error("Failed to update coupon", "error", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update coupon"})
			}

			return e.JSON(http.StatusOK, couponToMap(coupon))
		})

		// DELETE /api/admin/coupons/{id} - Delete coupon
		se.Router.DELETE("/api/admin/coupons/{id}", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			couponID := e.Request.PathValue("id")
			coupon, err := app.FindRecordById("coupons", couponID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "coupon not found"})
			}

			if err := app.Delete(coupon); err != nil {
				app.Logger().Error("Failed to delete coupon", "error", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete coupon"})
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
		})

		// ──────────────────────────────────────────────────────────────────
		// PUBLIC ENDPOINTS
		// ──────────────────────────────────────────────────────────────────

		// POST /api/coupons/validate - Validate a coupon code
		se.Router.POST("/api/coupons/validate", RateLimitMiddleware(rl, "strict")(func(e *core.RequestEvent) error {
			var req struct {
				Code        string `json:"code"`
				ContentType string `json:"content_type"`
				ContentID   string `json:"content_id"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			if req.Code == "" {
				return e.JSON(http.StatusOK, map[string]interface{}{
					"valid":  false,
					"reason": "Code is required",
				})
			}

			reason := validateCoupon(app, strings.ToUpper(strings.TrimSpace(req.Code)), req.ContentType, req.ContentID)
			if reason != "" {
				return e.JSON(http.StatusOK, map[string]interface{}{
					"valid":  false,
					"reason": reason,
				})
			}

			// Coupon is valid - fetch it to return details
			couponRecords, _ := app.FindRecordsByFilter(
				"coupons",
				"code = {:code}",
				"",
				1,
				0,
				map[string]interface{}{"code": strings.ToUpper(strings.TrimSpace(req.Code))},
			)
			if len(couponRecords) == 0 {
				return e.JSON(http.StatusOK, map[string]interface{}{
					"valid":  false,
					"reason": "Coupon not found",
				})
			}

			coupon := couponRecords[0]
			return e.JSON(http.StatusOK, map[string]interface{}{
				"valid":          true,
				"discount_type":  coupon.GetString("discount_type"),
				"discount_value": coupon.GetFloat("discount_value"),
				"code":           coupon.GetString("code"),
			})
		}))

		return se.Next()
	})
}

// validateCoupon checks if a coupon code is valid for the given content.
// Returns empty string if valid, or a reason string if invalid.
func validateCoupon(app *pocketbase.PocketBase, code, contentType, contentID string) string {
	couponRecords, err := app.FindRecordsByFilter(
		"coupons",
		"code = {:code}",
		"",
		1,
		0,
		map[string]interface{}{"code": code},
	)
	if err != nil || len(couponRecords) == 0 {
		return "Coupon not found"
	}

	coupon := couponRecords[0]

	// Check active
	if !coupon.GetBool("active") {
		return "Coupon is not active"
	}

	// Check expiration
	expiresAt := coupon.GetDateTime("expires_at")
	if !expiresAt.IsZero() && expiresAt.Time().Before(time.Now()) {
		return "Coupon expired"
	}

	// Check max uses
	maxUses := coupon.GetInt("max_uses")
	currentUses := coupon.GetInt("current_uses")
	if maxUses > 0 && currentUses >= maxUses {
		return "Coupon usage limit reached"
	}

	// Check applicable type
	applicableType := coupon.GetString("applicable_type")
	if applicableType != "" && applicableType != "all" && contentType != "" {
		if applicableType != contentType {
			return "Coupon not applicable to this content type"
		}
	}

	// Check applicable IDs (if specified)
	if contentID != "" && applicableType != "all" {
		applicableIDs := coupon.Get("applicable_ids")
		if applicableIDs != nil {
			// Check if the array is non-empty and the content ID is in it
			if ids, ok := applicableIDs.([]interface{}); ok && len(ids) > 0 {
				found := false
				for _, id := range ids {
					if idStr, ok := id.(string); ok && idStr == contentID {
						found = true
						break
					}
				}
				if !found {
					return "Coupon not applicable to this content"
				}
			}
		}
	}

	return ""
}

// IncrementCouponUses increments the current_uses count for a coupon.
// Called from the webhook handler after successful purchase.
func IncrementCouponUses(app *pocketbase.PocketBase, couponCode string) {
	if couponCode == "" {
		return
	}

	couponRecords, err := app.FindRecordsByFilter(
		"coupons",
		"code = {:code}",
		"",
		1,
		0,
		map[string]interface{}{"code": strings.ToUpper(strings.TrimSpace(couponCode))},
	)
	if err != nil || len(couponRecords) == 0 {
		return
	}

	coupon := couponRecords[0]
	coupon.Set("current_uses", coupon.GetInt("current_uses")+1)
	if err := app.Save(coupon); err != nil {
		app.Logger().Error("Failed to increment coupon uses", "error", err, "code", couponCode)
	}
}

// CalculateDiscountedPrice applies a coupon discount to a price (in cents).
// For percentage discounts, discountValue is 1-100.
// For fixed discounts, discountValue is in dollars (e.g. 5.00 = $5 off).
// Returns the discounted price in cents.
func CalculateDiscountedPrice(originalPrice int, discountType string, discountValue float64) int {
	var discount float64
	if discountType == "percentage" {
		discount = float64(originalPrice) * (discountValue / 100)
	} else {
		// Fixed discount: value stored as dollars, convert to cents
		discount = discountValue * 100
	}
	result := originalPrice - int(math.Round(discount))
	if result < 0 {
		return 0
	}
	return result
}

// couponToMap converts a coupon record to a map for JSON response
func couponToMap(record *core.Record) map[string]interface{} {
	return map[string]interface{}{
		"id":              record.Id,
		"code":            record.GetString("code"),
		"discount_type":   record.GetString("discount_type"),
		"discount_value":  record.GetFloat("discount_value"),
		"applicable_type": record.GetString("applicable_type"),
		"applicable_ids":  record.Get("applicable_ids"),
		"max_uses":        record.GetInt("max_uses"),
		"current_uses":    record.GetInt("current_uses"),
		"expires_at":      record.GetDateTime("expires_at"),
		"active":          record.GetBool("active"),
		"created":         record.GetDateTime("created"),
	}
}
