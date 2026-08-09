// OVAV cPanel v5.3 — Native auth analytics.
//
// Stores auth events in a local SQLite database embedded in the container.
// No external services required. Query via GET /api/v1/auth/analytics.
//
// Schema:
//   auth_events(id, timestamp, user_id, email, ip, country, action,
//               status, risk_score, detail, req_id, user_agent)

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite" // SQLite — pure Go, no C deps
)

// ── Auth analytics store ────────────────────────────────────────────────────────

var (
	analyticsDB     *sql.DB
	analyticsDBOnce sync.Once
	analyticsMu     sync.RWMutex
)

func getAnalyticsDB() *sql.DB {
	analyticsDBOnce.Do(func() {
		db, err := sql.Open("sqlite", ".ovav/security/auth_analytics.db?_journal=WAL&_busy_timeout=5000")
		if err != nil {
			fmt.Fprintf(os.Stderr, "[analytics] failed to open DB: %v\n", err)
			return
		}
		analyticsDB = db
		if err := initAnalyticsSchema(); err != nil {
			fmt.Fprintf(os.Stderr, "[analytics] schema init failed: %v\n", err)
		}
	})
	return analyticsDB
}

func initAnalyticsSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS auth_events (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp  TEXT    NOT NULL,
		user_id    TEXT    NOT NULL DEFAULT '',
		email      TEXT    NOT NULL DEFAULT '',
		ip         TEXT    NOT NULL DEFAULT '',
		country    TEXT    NOT NULL DEFAULT '',
		action     TEXT    NOT NULL DEFAULT '',
		status     TEXT    NOT NULL DEFAULT '',
		risk_score INTEGER NOT NULL DEFAULT 0,
		detail     TEXT    NOT NULL DEFAULT '',
		req_id     TEXT    NOT NULL DEFAULT '',
		user_agent TEXT    NOT NULL DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_auth_timestamp ON auth_events(timestamp);
	CREATE INDEX IF NOT EXISTS idx_auth_email    ON auth_events(email);
	CREATE INDEX IF NOT EXISTS idx_auth_action   ON auth_events(action);
	CREATE INDEX IF NOT EXISTS idx_auth_status   ON auth_events(status);
	`
	_, err := analyticsDB.Exec(schema)
	return err
}

// RecordAuthEvent persists an auth event to SQLite.
func RecordAuthEvent(event AuditEvent, riskScore int) {
	db := getAnalyticsDB()
	if db == nil {
		return
	}
	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := db.Exec(`
		INSERT INTO auth_events
		(timestamp, user_id, email, ip, country, action, status, risk_score, detail, req_id, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		event.Timestamp,
		event.UserID,
		event.Email,
		event.IP,
		event.Country,
		event.Action,
		event.Status,
		riskScore,
		event.Detail,
		event.ReqID,
		event.UserAgent,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[analytics] insert failed: %v\n", err)
	}
}

// ── Analytics handlers ──────────────────────────────────────────────────────────

// handleAuthAnalytics returns login statistics for the admin dashboard.
func handleAuthAnalytics(w http.ResponseWriter, r *http.Request) {
	db := getAnalyticsDB()
	if db == nil {
		sendError(w, "analytics DB not available", http.StatusServiceUnavailable)
		return
	}

	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		fmt.Sscanf(d, "%d", &days)
	}
	if days < 1 {
		days = 1
	}
	if days > 90 {
		days = 90
	}
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	type stat struct {
		Date    string `json:"date"`
		Count   int    `json:"count"`
		Failed  int    `json:"failed"`
		Allowed int    `json:"allowed"`
	}

	var stats []stat
	rows, err := db.Query(`
		SELECT DATE(timestamp) as date,
		       COUNT(*) as total,
		       SUM(CASE WHEN status IN ('denied','error') THEN 1 ELSE 0 END) as failed,
		       SUM(CASE WHEN status = 'ok' THEN 1 ELSE 0 END) as allowed
		FROM auth_events
		WHERE timestamp >= ?
		GROUP BY DATE(timestamp)
		ORDER BY date DESC
	`, cutoff)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s stat
			rows.Scan(&s.Date, &s.Count, &s.Failed, &s.Allowed)
			stats = append(stats, s)
		}
	}

	// Top risk events
	type riskEvent struct {
		Timestamp string `json:"timestamp"`
		Email     string `json:"email"`
		IP        string `json:"ip"`
		Country   string `json:"country"`
		Action    string `json:"action"`
		RiskScore int    `json:"risk_score"`
		Status    string `json:"status"`
		Detail    string `json:"detail"`
	}
	var topRisks []riskEvent
	rrows, err := db.Query(`
		SELECT timestamp, email, ip, country, action, risk_score, status, detail
		FROM auth_events
		WHERE risk_score > 0 AND timestamp >= ?
		ORDER BY risk_score DESC, timestamp DESC
		LIMIT 20
	`, cutoff)
	if err == nil {
		defer rrows.Close()
		for rrows.Next() {
			var re riskEvent
			rrows.Scan(&re.Timestamp, &re.Email, &re.IP, &re.Country, &re.Action, &re.RiskScore, &re.Status, &re.Detail)
			topRisks = append(topRisks, re)
		}
	}

	// Failed attempts by IP (potential brute force)
	type ipStat struct {
		IP     string `json:"ip"`
		Count  int    `json:"count"`
		Emails string `json:"emails"`
	}
	var ipStats []ipStat
	ipRows, err := db.Query(`
		SELECT ip, COUNT(*) as cnt, GROUP_CONCAT(DISTINCT email) as emails
		FROM auth_events
		WHERE status IN ('denied','error') AND timestamp >= ?
		GROUP BY ip
		HAVING cnt >= 3
		ORDER BY cnt DESC
		LIMIT 20
	`, cutoff)
	if err == nil {
		defer ipRows.Close()
		for ipRows.Next() {
			var s ipStat
			ipRows.Scan(&s.IP, &s.Count, &s.Emails)
			ipStats = append(ipStats, s)
		}
	}

	// Unique users + countries
	var totalUsers, totalCountries int
	db.QueryRow(`SELECT COUNT(DISTINCT email) FROM auth_events WHERE timestamp >= ? AND email != ''`, cutoff).Scan(&totalUsers)
	db.QueryRow(`SELECT COUNT(DISTINCT country) FROM auth_events WHERE timestamp >= ? AND country != ''`, cutoff).Scan(&totalCountries)

	sendOK(w, map[string]interface{}{
		"period_days":     days,
		"total_events":    len(stats),
		"unique_users":    totalUsers,
		"countries":       totalCountries,
		"daily_stats":     stats,
		"top_risk":        topRisks,
		"brute_force_ips": ipStats,
	})
}

// handleAuthAuditLog returns paginated auth events with filters.
func handleAuthAuditLog(w http.ResponseWriter, r *http.Request) {
	db := getAnalyticsDB()
	if db == nil {
		sendError(w, "analytics DB not available", http.StatusServiceUnavailable)
		return
	}

	email := r.URL.Query().Get("email")
	ip := r.URL.Query().Get("ip")
	action := r.URL.Query().Get("action")
	status := r.URL.Query().Get("status")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if limit < 1 || limit > 500 {
		limit = 50
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	where := "WHERE 1=1"
	var args []interface{}

	if email != "" {
		where += " AND email LIKE ?"
		args = append(args, "%"+email+"%")
	}
	if ip != "" {
		where += " AND ip LIKE ?"
		args = append(args, "%"+ip+"%")
	}
	if action != "" {
		where += " AND action = ?"
		args = append(args, action)
	}
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}

	args = append(args, limit, offset)
	rows, err := db.Query(fmt.Sprintf(`
		SELECT id, timestamp, user_id, email, ip, country, action, status, risk_score, detail, req_id, user_agent
		FROM auth_events %s
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`, where), args...)
	if err != nil {
		sendError(w, "query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type auditRow struct {
		ID        int    `json:"id"`
		Timestamp string `json:"timestamp"`
		UserID    string `json:"user_id"`
		Email     string `json:"email"`
		IP        string `json:"ip"`
		Country   string `json:"country"`
		Action    string `json:"action"`
		Status    string `json:"status"`
		RiskScore int    `json:"risk_score"`
		Detail    string `json:"detail"`
		ReqID     string `json:"req_id"`
		UserAgent string `json:"user_agent"`
	}

	var events []auditRow
	for rows.Next() {
		var e auditRow
		rows.Scan(&e.ID, &e.Timestamp, &e.UserID, &e.Email, &e.IP, &e.Country,
			&e.Action, &e.Status, &e.RiskScore, &e.Detail, &e.ReqID, &e.UserAgent)
		events = append(events, e)
	}

	var total int
	countWhere := strings.Replace(where, "LIMIT ? OFFSET ?", "", 1)
	db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM auth_events %s", countWhere), args[:len(args)-2]...).Scan(&total)

	sendOK(w, map[string]interface{}{
		"events":  events,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
		"filters": map[string]string{"email": email, "ip": ip, "action": action, "status": status},
	})
}

// recordRiskScore extracts risk score from AuditEvent detail.
// In production this would be stored as a proper field; this is a heuristic parse.
func extractRiskScore(event AuditEvent) int {
	if event.Detail == "" {
		return 0
	}
	var score int
	if _, err := fmt.Sscanf(event.Detail, "risk=%d", &score); err == nil {
		return score
	}
	return 0
}

// handlePortalEvent receives login events from the ovav.dev login portal
// (separate frontend) and stores them in the cPanel analytics DB.
// Called by the login portal frontend after each login attempt.
// Security: requires X-OVAV-Portal-Key header matching PORTAL_EVENT_KEY env var.
func handlePortalEvent(w http.ResponseWriter, r *http.Request) {
	portalKey := os.Getenv("PORTAL_EVENT_KEY")
	if portalKey == "" {
		http.Error(w, "portal event intake disabled", http.StatusServiceUnavailable)
		return
	}
	if r.Header.Get("X-OVAV-Portal-Key") != portalKey {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload struct {
		Email     string `json:"email"`
		IP        string `json:"ip"`
		Country   string `json:"country"`
		Action    string `json:"action"`
		Status    string `json:"status"`
		Detail    string `json:"detail"`
		RiskScore int    `json:"risk_score"`
		UserAgent string `json:"user_agent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	event := AuditEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		UserID:    "",
		Email:     payload.Email,
		IP:        payload.IP,
		Country:   payload.Country,
		Action:    payload.Action,
		Status:    payload.Status,
		Detail:    payload.Detail,
		UserAgent: payload.UserAgent,
	}
	RecordAuthEvent(event, payload.RiskScore)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"recorded":true}`)
}
