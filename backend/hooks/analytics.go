package hooks

import (
	"net/http"
	"strings"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// DailyViewCount is one day's view count in the analytics time series.
type DailyViewCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// fillDailyViewCounts returns a complete ascending daily series covering the
// last `days` calendar days ending today (UTC). `counts` holds the days that
// actually had views (keyed "YYYY-MM-DD"); every other day is emitted with
// count 0. Without this, `GROUP BY DATE(created)` silently omits zero-view days
// and the chart renders fewer bars than the selected period.
func fillDailyViewCounts(counts map[string]int, days int, now time.Time) []DailyViewCount {
	if days < 1 {
		return []DailyViewCount{}
	}
	series := make([]DailyViewCount, 0, days)
	today := now.UTC()
	for i := days - 1; i >= 0; i-- {
		d := today.AddDate(0, 0, -i).Format("2006-01-02")
		series = append(series, DailyViewCount{Date: d, Count: counts[d]})
	}
	return series
}

// RegisterAnalyticsHooks registers analytics event logging, the analytics API, and cleanup.
func RegisterAnalyticsHooks(app *pocketbase.PocketBase, planConfig *services.PlanConfig) {
	// Analytics API endpoint
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// GET /api/admin/analytics?period=7d|30d|90d
		se.Router.GET("/api/admin/analytics", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			if !planConfig.IsManaged() || planConfig.HasFeature("analytics") {
				// allowed
			} else {
				return respondForbidden(e, "Analytics is not available on your current plan")
			}

			hasAdvancedAnalytics := !planConfig.IsManaged() || planConfig.HasFeature("analytics")

			period := e.Request.URL.Query().Get("period")
			if period == "" {
				period = "30d"
			}

			days := 30
			switch period {
			case "7d":
				days = 7
			case "30d":
				days = 30
			case "90d":
				days = 90
			}

			cutoff := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02 15:04:05.000Z")

			// Views over time (daily counts). GROUP BY DATE omits zero-view days,
			// so collect the present days into a map and zero-fill the full window.
			dailyCounts := make(map[string]int)

			rows, err := app.DB().
				NewQuery("SELECT DATE(created) as date, COUNT(*) as count FROM access_logs WHERE created >= {:cutoff} GROUP BY DATE(created) ORDER BY date ASC").
				Bind(map[string]any{"cutoff": cutoff}).
				Rows()
			if err != nil {
				app.Logger().Warn("analytics: query failed", "query", "views_over_time", "error", err)
			} else {
				defer rows.Close()
				for rows.Next() {
					var date string
					var count int
					if err := rows.Scan(&date, &count); err == nil {
						dailyCounts[date] = count
					}
				}
			}
			viewsOverTime := fillDailyViewCounts(dailyCounts, days, time.Now())

			// Total views in period
			var totalViews int
			app.DB().
				NewQuery("SELECT COUNT(*) FROM access_logs WHERE created >= {:cutoff}").
				Bind(map[string]any{"cutoff": cutoff}).
				Row(&totalViews)

			// Unique visitors in period (distinct daily-salted IP hashes).
			// Note: ip_hash is re-salted each UTC day, so a visitor active on
			// multiple days counts once per day — this is a privacy-preserving
			// approximation, consistent with how access_logs stores IPs.
			var uniqueVisitors int
			app.DB().
				NewQuery("SELECT COUNT(DISTINCT ip_hash) FROM access_logs WHERE created >= {:cutoff} AND ip_hash != ''").
				Bind(map[string]any{"cutoff": cutoff}).
				Row(&uniqueVisitors)

			// Popular views
			type viewStat struct {
				ViewID   string `json:"view_id"`
				ViewSlug string `json:"view_slug"`
				Count    int    `json:"count"`
			}
			var popularViews []viewStat

			rows2, err := app.DB().
				NewQuery("SELECT view_id, view_slug, COUNT(*) as count FROM access_logs WHERE created >= {:cutoff} GROUP BY view_id ORDER BY count DESC LIMIT 10").
				Bind(map[string]any{"cutoff": cutoff}).
				Rows()
			if err != nil {
				app.Logger().Warn("analytics: query failed", "query", "popular_views", "error", err)
			} else {
				defer rows2.Close()
				for rows2.Next() {
					var vs viewStat
					if err := rows2.Scan(&vs.ViewID, &vs.ViewSlug, &vs.Count); err == nil {
						popularViews = append(popularViews, vs)
					}
				}
			}

			// Per-share-token view counts
			type tokenStat struct {
				ShareTokenID string `json:"share_token_id"`
				Count        int    `json:"count"`
			}
			var tokenStats []tokenStat

			rows5, err := app.DB().
				NewQuery("SELECT share_token_id, COUNT(*) as count FROM access_logs WHERE created >= {:cutoff} AND share_token_id != '' GROUP BY share_token_id ORDER BY count DESC").
				Bind(map[string]any{"cutoff": cutoff}).
				Rows()
			if err != nil {
				app.Logger().Warn("analytics: query failed", "query", "token_stats", "error", err)
			} else {
				defer rows5.Close()
				for rows5.Next() {
					var ts tokenStat
					if err := rows5.Scan(&ts.ShareTokenID, &ts.Count); err == nil {
						tokenStats = append(tokenStats, ts)
					}
				}
			}

			// Build response with basic analytics (available to all analytics tiers)
			result := map[string]any{
				"period":          period,
				"total_views":     totalViews,
				"unique_visitors": uniqueVisitors,
				"views_over_time": viewsOverTime,
				"popular_views":   popularViews,
				"token_stats":     tokenStats,
			}

			// Advanced analytics: referrers and countries (Pro+ only)
			if hasAdvancedAnalytics {
				// Top referrers — group by domain (strip protocol and path)
				type referrerStat struct {
					Referrer string `json:"referrer"`
					Count    int    `json:"count"`
				}
				var topReferrers []referrerStat

				rows3, err := app.DB().
					NewQuery(`SELECT
						CASE
							WHEN referrer LIKE 'https://%' THEN SUBSTR(referrer, 9, CASE WHEN INSTR(SUBSTR(referrer, 9), '/') > 0 THEN INSTR(SUBSTR(referrer, 9), '/') - 1 ELSE LENGTH(SUBSTR(referrer, 9)) END)
							WHEN referrer LIKE 'http://%' THEN SUBSTR(referrer, 8, CASE WHEN INSTR(SUBSTR(referrer, 8), '/') > 0 THEN INSTR(SUBSTR(referrer, 8), '/') - 1 ELSE LENGTH(SUBSTR(referrer, 8)) END)
							ELSE referrer
						END as domain,
						COUNT(*) as count
						FROM access_logs
						WHERE created >= {:cutoff} AND referrer != '' AND referrer NOT LIKE '%/admin%'
						GROUP BY domain ORDER BY count DESC LIMIT 10`).
					Bind(map[string]any{"cutoff": cutoff}).
					Rows()
				if err != nil {
					app.Logger().Warn("analytics: query failed", "query", "top_referrers", "error", err)
				} else {
					defer rows3.Close()
					for rows3.Next() {
						var rs referrerStat
						if err := rows3.Scan(&rs.Referrer, &rs.Count); err == nil {
							topReferrers = append(topReferrers, rs)
						}
					}
				}

				// Country breakdown
				type countryStat struct {
					CountryCode string `json:"country_code"`
					Count       int    `json:"count"`
				}
				var countries []countryStat

				rows4, err := app.DB().
					NewQuery("SELECT country_code, COUNT(*) as count FROM access_logs WHERE created >= {:cutoff} AND country_code != '' GROUP BY country_code ORDER BY count DESC LIMIT 20").
					Bind(map[string]any{"cutoff": cutoff}).
					Rows()
				if err != nil {
					app.Logger().Warn("analytics: query failed", "query", "countries", "error", err)
				} else {
					defer rows4.Close()
					for rows4.Next() {
						var cs countryStat
						if err := rows4.Scan(&cs.CountryCode, &cs.Count); err == nil {
							countries = append(countries, cs)
						}
					}
				}

				result["top_referrers"] = topReferrers
				result["countries"] = countries

				// UTM campaign breakdown
				type utmStat struct {
					Source   string `json:"source"`
					Medium   string `json:"medium"`
					Campaign string `json:"campaign"`
					Count    int    `json:"count"`
				}
				var utmStats []utmStat

				rows6, err := app.DB().
					NewQuery("SELECT COALESCE(NULLIF(utm_source,''),'(direct)') as source, COALESCE(NULLIF(utm_medium,''),'(none)') as medium, COALESCE(NULLIF(utm_campaign,''),'(none)') as campaign, COUNT(*) as count FROM access_logs WHERE created >= {:cutoff} AND (utm_source != '' OR utm_medium != '' OR utm_campaign != '') GROUP BY utm_source, utm_medium, utm_campaign ORDER BY count DESC LIMIT 20").
					Bind(map[string]any{"cutoff": cutoff}).
					Rows()
				if err != nil {
					app.Logger().Warn("analytics: query failed", "query", "utm_stats", "error", err)
				} else {
					defer rows6.Close()
					for rows6.Next() {
						var us utmStat
						if err := rows6.Scan(&us.Source, &us.Medium, &us.Campaign, &us.Count); err == nil {
							utmStats = append(utmStats, us)
						}
					}
				}
				result["utm_stats"] = utmStats
			}

			return e.JSON(http.StatusOK, result)
		})

		return se.Next()
	})

	// Cleanup: prune access_logs older than 90 days
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		done := make(chan struct{})

		app.OnTerminate().BindFunc(func(te *core.TerminateEvent) error {
			close(done)
			return te.Next()
		})

		go func() {
			defer func() {
				if r := recover(); r != nil {
					app.Logger().Error("analytics: pruning goroutine panicked", "error", r)
				}
			}()

			time.Sleep(60 * time.Second)
			pruneAccessLogs(app)

			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					pruneAccessLogs(app)
				case <-done:
					return
				}
			}
		}()

		return se.Next()
	})
}

// LogViewAccess writes an access_logs entry asynchronously.
// Called from view.go when a public view is accessed.
// Filters out bots and captures user agent + hashed IP.
// shareTokenID is optional — set when access came via a share token.
func LogViewAccess(app *pocketbase.PocketBase, e *core.RequestEvent, viewID, viewSlug, shareTokenID string) {
	// Extract data synchronously from the request
	ua := GetUserAgent(e)
	if services.IsBot(ua) {
		return
	}

	referrer := e.Request.Header.Get("Referer")
	if len(referrer) > 2048 {
		referrer = referrer[:2048]
	}
	// Strip query params from referrer for privacy
	if idx := strings.Index(referrer, "?"); idx > 0 {
		referrer = referrer[:idx]
	}
	// Ignore admin page referrers — they're internal navigation, not real traffic sources
	if strings.Contains(referrer, "/admin") {
		referrer = ""
	}

	// Capture UTM parameters from the request URL
	utmSource := e.Request.URL.Query().Get("utm_source")
	utmMedium := e.Request.URL.Query().Get("utm_medium")
	utmCampaign := e.Request.URL.Query().Get("utm_campaign")

	path := e.Request.URL.Path
	// GDPR: daily-salted hash rotates daily and cannot be tracked across days.
	// Shared helper keeps this identical to the view dedup cache key.
	ipHash := services.HashIP(GetIP(e))

	// Write asynchronously
	go func() {
		defer func() {
			if r := recover(); r != nil {
				app.Logger().Error("analytics: access log goroutine panicked", "error", r)
			}
		}()

		collection, err := app.FindCollectionByNameOrId("access_logs")
		if err != nil {
			app.Logger().Warn("analytics: access_logs collection not found", "error", err)
			return
		}

		record := core.NewRecord(collection)
		record.Set("view_id", viewID)
		record.Set("view_slug", viewSlug)
		record.Set("referrer", referrer)
		record.Set("path", path)
		record.Set("user_agent", ua)
		record.Set("ip_hash", ipHash)
		if shareTokenID != "" {
			record.Set("share_token_id", shareTokenID)
		}
		if utmSource != "" {
			record.Set("utm_source", utmSource)
		}
		if utmMedium != "" {
			record.Set("utm_medium", utmMedium)
		}
		if utmCampaign != "" {
			record.Set("utm_campaign", utmCampaign)
		}
		// country_code left empty for now — can be populated later with GeoIP

		if err := app.Save(record); err != nil {
			app.Logger().Warn("Failed to write access log", "error", err)
		}
	}()
}

// pruneAccessLogs deletes access_logs entries older than 90 days using bulk SQL DELETE.
func pruneAccessLogs(app *pocketbase.PocketBase) {
	cutoff := time.Now().UTC().AddDate(0, 0, -90).Format("2006-01-02 15:04:05.000Z")
	res, err := app.DB().NewQuery(
		"DELETE FROM access_logs WHERE created < {:cutoff}",
	).Bind(map[string]any{"cutoff": cutoff}).Execute()
	if err != nil {
		app.Logger().Warn("analytics: failed to prune access logs", "error", err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		app.Logger().Info("Pruned access logs", "count", n)
	}
}
