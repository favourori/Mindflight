package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type dashboardResponse struct {
	Title          string        `json:"title"`
	Subtitle       string        `json:"subtitle"`
	GeneratedAt    time.Time     `json:"generatedAt"`
	Coverage       coverageStats `json:"coverage"`
	Metrics        []metricCard  `json:"metrics"`
	Trends         []trendPoint  `json:"trends"`
	Stressors      []barItem     `json:"stressors"`
	Resources      []barItem     `json:"resources"`
	SupportRoutes  []barItem     `json:"supportRoutes"`
	ModuleAdoption []barItem     `json:"moduleAdoption"`
	Alerts         []alertItem   `json:"alerts"`
	Actions        []actionItem  `json:"actions"`
}

type coverageStats struct {
	TotalAirmen    int     `json:"totalAirmen"`
	CheckedInToday int     `json:"checkedInToday"`
	CompletionRate float64 `json:"completionRate"`
}

type metricCard struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Delta string `json:"delta"`
	Tone  string `json:"tone"`
}

type trendPoint struct {
	Day        string  `json:"day"`
	Readiness  float64 `json:"readiness"`
	Completion float64 `json:"completion"`
	Stress     float64 `json:"stress"`
	Sleep      float64 `json:"sleep"`
	CheckedIn  int     `json:"checkedIn"`
}

type barItem struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

type alertItem struct {
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Severity string `json:"severity"`
}

type actionItem struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Owner  string `json:"owner"`
	Status string `json:"status"`
}

type authUser struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

type airmanDashboardResponse struct {
	Title        string             `json:"title"`
	Subtitle     string             `json:"subtitle"`
	GeneratedAt  time.Time          `json:"generatedAt"`
	LatestMood   float64            `json:"latestMood"`
	LatestStress float64            `json:"latestStress"`
	LatestSleep  float64            `json:"latestSleep"`
	Trend        []trendPoint       `json:"trend"`
	Resources    []barItem          `json:"resources"`
	Tips         []actionItem       `json:"tips"`
	Modules      []trainingModule   `json:"modules"`
	Coach        coachInsight       `json:"coach"`
	Wearable     wearableStatus     `json:"wearable"`
	PeerSupport  peerSupportPayload `json:"peerSupport"`
	Crisis       crisisPayload      `json:"crisis"`
	Privacy      privacySettings    `json:"privacy"`
}

type trainingModule struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Category    string `json:"category"`
	Duration    int    `json:"duration"`
	Description string `json:"description"`
	Progress    int    `json:"progress"`
	Status      string `json:"status"`
}

type coachInsight struct {
	Headline        string   `json:"headline"`
	Message         string   `json:"message"`
	Recommendations []string `json:"recommendations"`
}

type wearableStatus struct {
	Connected  bool    `json:"connected"`
	Source     string  `json:"source"`
	LastSync   string  `json:"lastSync"`
	SleepHours float64 `json:"sleepHours"`
	Strain     float64 `json:"strain"`
	Note       string  `json:"note"`
}

type peerSupportPayload struct {
	Channels []supportChannel        `json:"channels"`
	Requests []supportRequestSummary `json:"requests"`
}

type supportChannel struct {
	Key          string `json:"key"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Availability string `json:"availability"`
}

type supportRequestSummary struct {
	Channel   string    `json:"channel"`
	Urgency   string    `json:"urgency"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type crisisPayload struct {
	Options []supportChannel `json:"options"`
	Notice  string           `json:"notice"`
}

type privacySettings struct {
	ShareTrends             bool `json:"shareTrends"`
	AllowPeerSupport        bool `json:"allowPeerSupport"`
	AllowLeadershipOutreach bool `json:"allowLeadershipOutreach"`
	AllowWearableSync       bool `json:"allowWearableSync"`
}

type supportRequest struct {
	Channel string `json:"channel"`
	Urgency string `json:"urgency"`
	Note    string `json:"note"`
}

type coachRequest struct {
	Prompt string `json:"prompt"`
}

type moduleUpdateRequest struct {
	ID       string `json:"id"`
	Progress int    `json:"progress"`
}

type checkinSubmission struct {
	Mood   float64 `json:"mood"`
	Stress float64 `json:"stress"`
	Sleep  float64 `json:"sleep"`
	Note   string  `json:"note"`
}

type checkinRow struct {
	Day         string
	TotalAirmen int
	CheckedIn   int
	Completion  float64
	AvgMood     float64
	AvgStress   float64
	AvgSleep    float64
}

type valueRow struct {
	Name  string
	Count int
}

func main() {
	db, err := openDatabase("mindflight.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := initializeDatabase(db); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		var request loginRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		request.Username = strings.TrimSpace(strings.ToLower(request.Username))
		if request.Username == "" || request.Password == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password are required"})
			return
		}

		user, err := authenticateUser(db, request.Username, request.Password)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}

		sessionID, err := createSession(db, user)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "mindflight_session",
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().UTC().Add(12 * time.Hour),
		})

		writeJSON(w, http.StatusOK, loginResponse{Username: user.Username, Role: user.Role})
	})
	mux.HandleFunc("/api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		user, err := currentUser(r, db)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
			return
		}

		writeJSON(w, http.StatusOK, loginResponse{Username: user.Username, Role: user.Role})
	})
	mux.HandleFunc("/api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		if cookie, err := r.Cookie("mindflight_session"); err == nil {
			_, _ = db.Exec(`DELETE FROM sessions WHERE id = ?`, cookie.Value)
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "mindflight_session",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
		})
		writeJSON(w, http.StatusOK, map[string]string{"status": "signed_out"})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/leadership", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		user, err := currentUser(r, db)
		if err != nil || user.Role != "leadership" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authorized"})
			return
		}

		payload, err := buildDashboard(db)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, payload)
	})
	mux.HandleFunc("/api/airman", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		user, err := currentUser(r, db)
		if err != nil || user.Role != "airman" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authorized"})
			return
		}

		payload, err := buildAirmanDashboard(db, user.Username)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, payload)
	})
	mux.HandleFunc("/api/checkins", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		user, err := currentUser(r, db)
		if err != nil || user.Role != "airman" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authorized"})
			return
		}

		var submission checkinSubmission
		if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		if submission.Mood < 1 || submission.Mood > 5 || submission.Stress < 1 || submission.Stress > 5 || submission.Sleep < 0 || submission.Sleep > 12 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "check-in values out of range"})
			return
		}

		note := submission.Note
		if len(note) > 240 {
			note = note[:240]
		}

		createdAt := time.Now().UTC()
		if _, err := db.Exec(
			`INSERT INTO checkin_submissions (created_at, username, mood, stress, sleep, note) VALUES (?, ?, ?, ?, ?, ?)`,
			createdAt.Format(time.RFC3339),
			user.Username,
			submission.Mood,
			submission.Stress,
			submission.Sleep,
			note,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"status":    "saved",
			"createdAt": createdAt,
			"mood":      submission.Mood,
			"stress":    submission.Stress,
			"sleep":     submission.Sleep,
			"note":      note,
		})
	})
	mux.HandleFunc("/api/airman/privacy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		user, err := currentUser(r, db)
		if err != nil || user.Role != "airman" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authorized"})
			return
		}

		var settings privacySettings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		if err := savePrivacySettings(db, user.Username, settings); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, settings)
	})
	mux.HandleFunc("/api/airman/support-request", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		user, err := currentUser(r, db)
		if err != nil || user.Role != "airman" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authorized"})
			return
		}

		var request supportRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		request.Channel = strings.TrimSpace(request.Channel)
		request.Urgency = strings.TrimSpace(request.Urgency)
		request.Note = strings.TrimSpace(request.Note)
		if request.Channel == "" || request.Urgency == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "channel and urgency are required"})
			return
		}

		if len(request.Note) > 320 {
			request.Note = request.Note[:320]
		}

		createdAt, err := createSupportRequest(db, user.Username, request)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{"status": "queued", "createdAt": createdAt})
	})
	mux.HandleFunc("/api/airman/modules/complete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		user, err := currentUser(r, db)
		if err != nil || user.Role != "airman" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authorized"})
			return
		}

		var request moduleUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		if request.ID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "module id is required"})
			return
		}
		request.Progress = int(clamp(float64(request.Progress), 0, 100))

		if err := updateModuleProgress(db, user.Username, request.ID, request.Progress); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "progress": request.Progress})
	})
	mux.HandleFunc("/api/airman/coach", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		user, err := currentUser(r, db)
		if err != nil || user.Role != "airman" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authorized"})
			return
		}

		var request coachRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		insight, err := buildCoachInsight(db, user.Username, request.Prompt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, insight)
	})

	staticRoot := filepath.Join(".", "frontend", "dist")
	staticFS := http.FileServer(http.Dir(staticRoot))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/healthz" {
			http.NotFound(w, r)
			return
		}

		if r.URL.Path != "/" {
			if _, err := os.Stat(filepath.Join(staticRoot, strings.TrimPrefix(r.URL.Path, "/"))); err == nil {
				staticFS.ServeHTTP(w, r)
				return
			}
		}

		http.ServeFile(w, r, filepath.Join(staticRoot, "index.html"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("MindFlight API listening on http://127.0.0.1:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, cors(mux)))
}

func authenticateUser(db *sql.DB, username, password string) (authUser, error) {
	var role string
	if err := db.QueryRow(`SELECT role FROM users WHERE username = ? AND password = ?`, username, password).Scan(&role); err != nil {
		return authUser{}, err
	}

	return authUser{Username: username, Role: role}, nil
}

func createSession(db *sql.DB, user authUser) (string, error) {
	sessionID, err := randomToken(32)
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().UTC().Add(12 * time.Hour)
	if _, err := db.Exec(`INSERT INTO sessions (id, username, role, expires_at) VALUES (?, ?, ?, ?)`, sessionID, user.Username, user.Role, expiresAt.Format(time.RFC3339)); err != nil {
		return "", err
	}

	return sessionID, nil
}

func currentUser(r *http.Request, db *sql.DB) (authUser, error) {
	cookie, err := r.Cookie("mindflight_session")
	if err != nil {
		return authUser{}, err
	}

	var user authUser
	var expiresAt string
	if err := db.QueryRow(`SELECT username, role, expires_at FROM sessions WHERE id = ?`, cookie.Value).Scan(&user.Username, &user.Role, &expiresAt); err != nil {
		return authUser{}, err
	}

	parsedExpires, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return authUser{}, err
	}
	if time.Now().UTC().After(parsedExpires) {
		_, _ = db.Exec(`DELETE FROM sessions WHERE id = ?`, cookie.Value)
		return authUser{}, errors.New("session expired")
	}

	return user, nil
}

func randomToken(length int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buffer := make([]byte, length)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	for index, value := range buffer {
		buffer[index] = alphabet[int(value)%len(alphabet)]
	}

	return string(buffer), nil
}

func openDatabase(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", path))
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func initializeDatabase(db *sql.DB) error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS checkins (
			day TEXT PRIMARY KEY,
			total_airmen INTEGER NOT NULL,
			checked_in INTEGER NOT NULL,
			completion REAL NOT NULL,
			avg_mood REAL NOT NULL,
			avg_stress REAL NOT NULL,
			avg_sleep REAL NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS stressors (
			name TEXT PRIMARY KEY,
			count INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS resources (
			name TEXT PRIMARY KEY,
			count INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS alerts (
			title TEXT PRIMARY KEY,
			detail TEXT NOT NULL,
			severity TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS actions (
			title TEXT PRIMARY KEY,
			detail TEXT NOT NULL,
			owner TEXT NOT NULL,
			status TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY,
			password TEXT NOT NULL,
			role TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			role TEXT NOT NULL,
			expires_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS checkin_submissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at TEXT NOT NULL,
			username TEXT NOT NULL,
			mood REAL NOT NULL,
			stress REAL NOT NULL,
			sleep REAL NOT NULL,
			note TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS privacy_settings (
			username TEXT PRIMARY KEY,
			share_trends INTEGER NOT NULL,
			allow_peer_support INTEGER NOT NULL,
			allow_leadership_outreach INTEGER NOT NULL,
			allow_wearable_sync INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS module_progress (
			username TEXT NOT NULL,
			module_id TEXT NOT NULL,
			title TEXT NOT NULL,
			category TEXT NOT NULL,
			duration_minutes INTEGER NOT NULL,
			description TEXT NOT NULL,
			progress INTEGER NOT NULL,
			PRIMARY KEY (username, module_id)
		);`,
		`CREATE TABLE IF NOT EXISTS support_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			channel TEXT NOT NULL,
			urgency TEXT NOT NULL,
			note TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS wearable_status (
			username TEXT PRIMARY KEY,
			source TEXT NOT NULL,
			connected INTEGER NOT NULL,
			last_sync TEXT NOT NULL,
			sleep_hours REAL NOT NULL,
			strain REAL NOT NULL,
			note TEXT NOT NULL
		);`,
	}

	for _, statement := range schema {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}

	if err := ensureCheckinSubmissionUsernameColumn(db); err != nil {
		return err
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM checkins`).Scan(&count); err != nil {
		return err
	}
	if err := seedAuthUsers(db); err != nil {
		return err
	}
	if count > 0 {
		return seedPrototypeData(db)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UTC().Truncate(24 * time.Hour)
	for i := 13; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		completion := 82 + float64((13-i)%4)*2.1 + float64(i%3)*0.8
		avgMood := 3.0 + float64((13-i)%5)*0.12 + float64(i%2)*0.05
		avgStress := 4.9 - float64((13-i)%4)*0.08 + float64(i%3)*0.06
		avgSleep := 6.5 + float64((13-i)%4)*0.14 - float64(i%2)*0.05
		checkedIn := int(math.Round(146 * completion / 100))

		if _, err := tx.Exec(
			`INSERT INTO checkins (day, total_airmen, checked_in, completion, avg_mood, avg_stress, avg_sleep)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			day.Format("2006-01-02"),
			146,
			checkedIn,
			completion,
			avgMood,
			avgStress,
			avgSleep,
		); err != nil {
			return err
		}
	}

	for _, row := range []valueRow{
		{Name: "Shift load", Count: 42},
		{Name: "Sleep debt", Count: 36},
		{Name: "Family strain", Count: 28},
		{Name: "Heat / hydration", Count: 19},
		{Name: "Financial pressure", Count: 15},
	} {
		if _, err := tx.Exec(`INSERT INTO stressors (name, count) VALUES (?, ?)`, row.Name, row.Count); err != nil {
			return err
		}
	}

	for _, row := range []valueRow{
		{Name: "Self-guided resilience", Count: 56},
		{Name: "Sleep tools", Count: 44},
		{Name: "Mindfulness sessions", Count: 31},
		{Name: "Chaplain referrals", Count: 18},
		{Name: "Peer support prompts", Count: 13},
	} {
		if _, err := tx.Exec(`INSERT INTO resources (name, count) VALUES (?, ?)`, row.Name, row.Count); err != nil {
			return err
		}
	}

	alerts := []alertItem{
		{Title: "Two-week completion dip", Detail: "The maintenance flightline and night shift teams are falling below the target check-in rate.", Severity: "medium"},
		{Title: "Stress trend is rising", Detail: "Average stress moved up across the last four days after the ops tempo increase.", Severity: "high"},
		{Title: "Sleep recovery window is thin", Detail: "The unit is averaging less than 7 hours of sleep for a third of the force.", Severity: "medium"},
	}

	for _, alert := range alerts {
		if _, err := tx.Exec(`INSERT INTO alerts (title, detail, severity) VALUES (?, ?, ?)`, alert.Title, alert.Detail, alert.Severity); err != nil {
			return err
		}
	}

	actions := []actionItem{
		{Title: "Open the week with a normalized check-in reminder", Detail: "Push a short, non-punitive message at shift change and after PT.", Owner: "Sqdn leadership", Status: "Planned"},
		{Title: "Run a 10-minute sleep hygiene micro-brief", Detail: "Use the next staff meeting to cover recovery, hydration, and screen discipline.", Owner: "Flight chief", Status: "In progress"},
		{Title: "Route at-risk trends to trusted support points", Detail: "Keep the path discreet and voluntary so people can self-navigate early.", Owner: "Resilience team", Status: "Queued"},
	}

	for _, action := range actions {
		if _, err := tx.Exec(`INSERT INTO actions (title, detail, owner, status) VALUES (?, ?, ?, ?)`, action.Title, action.Detail, action.Owner, action.Status); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := seedPrototypeData(db); err != nil {
		return err
	}

	return seedAuthUsers(db)
}

func ensureCheckinSubmissionUsernameColumn(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(checkin_submissions)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var defaultValue sql.NullString
		var primaryKey int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if name == "username" {
			return nil
		}
	}

	_, err = db.Exec(`ALTER TABLE checkin_submissions ADD COLUMN username TEXT NOT NULL DEFAULT ''`)
	return err
}

func seedAuthUsers(db *sql.DB) error {
	users := []authUser{
		{Username: "leadership", Role: "leadership"},
		{Username: "airman", Role: "airman"},
	}
	passwords := map[string]string{
		"leadership": "flight2026",
		"airman":     "wingman2026",
	}

	for _, user := range users {
		if _, err := db.Exec(
			`INSERT INTO users (username, password, role) VALUES (?, ?, ?)
			 ON CONFLICT(username) DO UPDATE SET password = excluded.password, role = excluded.role`,
			user.Username,
			passwords[user.Username],
			user.Role,
		); err != nil {
			return err
		}
	}

	return nil
}

func seedPrototypeData(db *sql.DB) error {
	defaults := privacySettings{
		ShareTrends:             true,
		AllowPeerSupport:        true,
		AllowLeadershipOutreach: false,
		AllowWearableSync:       true,
	}
	if err := savePrivacySettings(db, "airman", defaults); err != nil {
		return err
	}

	modules := []trainingModule{
		{ID: "reset-drill", Title: "Reset Drill", Category: "Performance", Duration: 6, Description: "A short decompression routine for shift changes and pre-brief resets.", Progress: 100, Status: "Complete"},
		{ID: "sleep-recovery", Title: "Sleep Recovery Basics", Category: "Recovery", Duration: 8, Description: "Practical recovery habits for irregular schedules and split sleep.", Progress: 60, Status: "In progress"},
		{ID: "thought-reframe", Title: "Thought Reframe", Category: "CBT", Duration: 10, Description: "Catch spiraling thoughts and convert them into action-ready reframes.", Progress: 35, Status: "In progress"},
		{ID: "crew-check", Title: "Crew Check Conversation", Category: "Connection", Duration: 5, Description: "Practice a stigma-free way to check in on a teammate before strain compounds.", Progress: 0, Status: "Queued"},
	}
	for _, module := range modules {
		if _, err := db.Exec(
			`INSERT INTO module_progress (username, module_id, title, category, duration_minutes, description, progress)
			 VALUES (?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT(username, module_id) DO UPDATE SET
			 title = excluded.title,
			 category = excluded.category,
			 duration_minutes = excluded.duration_minutes,
			 description = excluded.description,
			 progress = CASE WHEN module_progress.progress > 0 THEN module_progress.progress ELSE excluded.progress END`,
			"airman", module.ID, module.Title, module.Category, module.Duration, module.Description, module.Progress,
		); err != nil {
			return err
		}
	}

	if _, err := db.Exec(
		`INSERT INTO wearable_status (username, source, connected, last_sync, sleep_hours, strain, note)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(username) DO UPDATE SET
		 source = excluded.source,
		 connected = excluded.connected,
		 last_sync = excluded.last_sync,
		 sleep_hours = excluded.sleep_hours,
		 strain = excluded.strain,
		 note = excluded.note`,
		"airman", "Oura demo sync", 1, time.Now().UTC().Add(-27*time.Minute).Format(time.RFC3339), 6.4, 7.1, "Sleep debt is improving, but recovery is still thin after late shift turnover.",
	); err != nil {
		return err
	}

	var supportCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM support_requests`).Scan(&supportCount); err != nil {
		return err
	}
	if supportCount == 0 {
		samples := []supportRequest{
			{Channel: "peer", Urgency: "medium", Note: "Requested a peer check-in after a rough night rotation."},
			{Channel: "chaplain", Urgency: "low", Note: "Quiet conversation requested for family stress."},
			{Channel: "mental-health", Urgency: "high", Note: "Requested same-day callback after stress spike."},
		}
		for _, request := range samples {
			if _, err := createSupportRequest(db, "airman", request); err != nil {
				return err
			}
		}
	}

	return nil
}

func savePrivacySettings(db *sql.DB, username string, settings privacySettings) error {
	_, err := db.Exec(
		`INSERT INTO privacy_settings (username, share_trends, allow_peer_support, allow_leadership_outreach, allow_wearable_sync)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(username) DO UPDATE SET
		 share_trends = excluded.share_trends,
		 allow_peer_support = excluded.allow_peer_support,
		 allow_leadership_outreach = excluded.allow_leadership_outreach,
		 allow_wearable_sync = excluded.allow_wearable_sync`,
		username,
		boolToInt(settings.ShareTrends),
		boolToInt(settings.AllowPeerSupport),
		boolToInt(settings.AllowLeadershipOutreach),
		boolToInt(settings.AllowWearableSync),
	)
	return err
}

func loadPrivacySettings(db *sql.DB, username string) (privacySettings, error) {
	var settings privacySettings
	var shareTrends, allowPeerSupport, allowLeadershipOutreach, allowWearableSync int
	if err := db.QueryRow(
		`SELECT share_trends, allow_peer_support, allow_leadership_outreach, allow_wearable_sync FROM privacy_settings WHERE username = ?`,
		username,
	).Scan(&shareTrends, &allowPeerSupport, &allowLeadershipOutreach, &allowWearableSync); err != nil {
		return settings, err
	}
	settings.ShareTrends = shareTrends == 1
	settings.AllowPeerSupport = allowPeerSupport == 1
	settings.AllowLeadershipOutreach = allowLeadershipOutreach == 1
	settings.AllowWearableSync = allowWearableSync == 1
	return settings, nil
}

func createSupportRequest(db *sql.DB, username string, request supportRequest) (time.Time, error) {
	createdAt := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO support_requests (username, channel, urgency, note, status, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		username,
		request.Channel,
		request.Urgency,
		request.Note,
		"Queued",
		createdAt.Format(time.RFC3339),
	)
	return createdAt, err
}

func loadModules(db *sql.DB, username string) ([]trainingModule, error) {
	rows, err := db.Query(`SELECT module_id, title, category, duration_minutes, description, progress FROM module_progress WHERE username = ? ORDER BY progress DESC, title`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []trainingModule
	for rows.Next() {
		var module trainingModule
		if err := rows.Scan(&module.ID, &module.Title, &module.Category, &module.Duration, &module.Description, &module.Progress); err != nil {
			return nil, err
		}
		switch {
		case module.Progress >= 100:
			module.Status = "Complete"
		case module.Progress > 0:
			module.Status = "In progress"
		default:
			module.Status = "Queued"
		}
		modules = append(modules, module)
	}
	return modules, rows.Err()
}

func updateModuleProgress(db *sql.DB, username, moduleID string, progress int) error {
	_, err := db.Exec(`UPDATE module_progress SET progress = ? WHERE username = ? AND module_id = ?`, progress, username, moduleID)
	return err
}

func loadWearableStatus(db *sql.DB, username string) (wearableStatus, error) {
	var status wearableStatus
	var connected int
	if err := db.QueryRow(
		`SELECT source, connected, last_sync, sleep_hours, strain, note FROM wearable_status WHERE username = ?`,
		username,
	).Scan(&status.Source, &connected, &status.LastSync, &status.SleepHours, &status.Strain, &status.Note); err != nil {
		return status, err
	}
	status.Connected = connected == 1
	return status, nil
}

func loadSupportRequests(db *sql.DB, username string) ([]supportRequestSummary, error) {
	rows, err := db.Query(`SELECT channel, urgency, status, created_at FROM support_requests WHERE username = ? ORDER BY created_at DESC LIMIT 3`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []supportRequestSummary
	for rows.Next() {
		var item supportRequestSummary
		var createdAt string
		if err := rows.Scan(&item.Channel, &item.Urgency, &item.Status, &createdAt); err != nil {
			return nil, err
		}
		parsed, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, err
		}
		item.CreatedAt = parsed
		requests = append(requests, item)
	}
	return requests, rows.Err()
}

func loadAggregateCounts(db *sql.DB, query string) ([]barItem, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []barItem
	for rows.Next() {
		var item barItem
		if err := rows.Scan(&item.Label, &item.Value); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func buildCoachInsight(db *sql.DB, username, prompt string) (coachInsight, error) {
	prompt = strings.TrimSpace(strings.ToLower(prompt))
	latestMood, latestStress, latestSleep, err := loadLatestCheckin(db, username)
	if err != nil {
		return coachInsight{}, err
	}

	headline := "Reset the next hour"
	message := "Your recent pattern suggests strain is outrunning recovery. Pick one small action you can finish before your next obligation."
	recommendations := []string{
		"Take a 2-minute breathing reset before the next task switch.",
		"Protect one uninterrupted hydration and snack window this shift.",
		"Use the sleep recovery module before end of day if you are still running hot.",
	}

	if strings.Contains(prompt, "sleep") || latestSleep < 6.5 {
		headline = "Protect recovery tonight"
		message = "Sleep is the thinnest part of your current readiness picture. Lower stimulation earlier and shorten the recovery ramp, not just bedtime."
		recommendations = []string{
			"Set a hard screen-off checkpoint 30 minutes before sleep.",
			"Keep caffeine out of the last third of your shift.",
			"Run the Sleep Recovery Basics module for one concrete habit change.",
		}
	} else if strings.Contains(prompt, "stress") || latestStress >= 4 || latestMood <= 2.5 {
		headline = "Downshift stress before it compounds"
		message = "Your stress score is elevated. Aim for one decompression action now and one support action before the end of the day."
		recommendations = []string{
			"Do a 90-second exhale-focused breathing cycle.",
			"Message a peer wingman instead of carrying the whole load alone.",
			"If the pressure is staying high, queue a discreet support request.",
		}
	}

	return coachInsight{Headline: headline, Message: message, Recommendations: recommendations}, nil
}

func loadLatestCheckin(db *sql.DB, username string) (float64, float64, float64, error) {
	var mood, stress, sleep float64
	err := db.QueryRow(
		`SELECT mood, stress, sleep FROM checkin_submissions WHERE username = ? ORDER BY created_at DESC LIMIT 1`,
		username,
	).Scan(&mood, &stress, &sleep)
	if err == sql.ErrNoRows {
		return 3, 3, 7, nil
	}
	return mood, stress, sleep, err
}

func defaultPeerSupport() peerSupportPayload {
	return peerSupportPayload{
		Channels: []supportChannel{
			{Key: "peer", Title: "Peer wingman", Description: "Quiet, non-clinical check-in with a trained peer supporter.", Availability: "Usually within 30 minutes"},
			{Key: "chaplain", Title: "Chaplain", Description: "Confidential conversation for moral injury, family strain, or grief.", Availability: "On-call today"},
			{Key: "mental-health", Title: "Mental health team", Description: "Escalate when the strain needs licensed follow-up or same-day support.", Availability: "Same-day callback"},
		},
	}
}

func defaultCrisisPayload() crisisPayload {
	return crisisPayload{
		Options: []supportChannel{
			{Key: "urgent-peer", Title: "Immediate peer reachback", Description: "Request a live peer response without filing a formal report.", Availability: "15 minutes"},
			{Key: "chaplain-now", Title: "Chaplain now", Description: "Contact a chaplain for immediate confidential support.", Availability: "Available now"},
			{Key: "emergency", Title: "Emergency handoff", Description: "Use emergency channels immediately if you or a teammate may be in danger.", Availability: "24/7"},
		},
		Notice: "MindFlight keeps this path discreet, but emergencies still need direct real-world response channels.",
	}
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func buildDashboard(db *sql.DB) (dashboardResponse, error) {
	rows, err := db.Query(`SELECT day, total_airmen, checked_in, completion, avg_mood, avg_stress, avg_sleep FROM checkins ORDER BY day`)
	if err != nil {
		return dashboardResponse{}, err
	}
	defer rows.Close()

	var checkins []checkinRow
	for rows.Next() {
		var row checkinRow
		if err := rows.Scan(&row.Day, &row.TotalAirmen, &row.CheckedIn, &row.Completion, &row.AvgMood, &row.AvgStress, &row.AvgSleep); err != nil {
			return dashboardResponse{}, err
		}
		checkins = append(checkins, row)
	}
	if err := rows.Err(); err != nil {
		return dashboardResponse{}, err
	}

	if len(checkins) == 0 {
		return dashboardResponse{}, errors.New("no dashboard data available")
	}

	todaySubmissions, err := countTodaySubmissions(db)
	if err != nil {
		return dashboardResponse{}, err
	}

	stressors, err := loadBarItems(db, `SELECT name, count FROM stressors ORDER BY count DESC`)
	if err != nil {
		return dashboardResponse{}, err
	}
	resources, err := loadBarItems(db, `SELECT name, count FROM resources ORDER BY count DESC`)
	if err != nil {
		return dashboardResponse{}, err
	}
	supportRoutes, err := loadAggregateCounts(db, `SELECT channel, COUNT(*) FROM support_requests GROUP BY channel ORDER BY COUNT(*) DESC`)
	if err != nil {
		return dashboardResponse{}, err
	}
	moduleAdoption, err := loadAggregateCounts(db, `SELECT category, COUNT(*) FROM module_progress WHERE progress > 0 GROUP BY category ORDER BY COUNT(*) DESC`)
	if err != nil {
		return dashboardResponse{}, err
	}
	alerts, err := loadAlerts(db)
	if err != nil {
		return dashboardResponse{}, err
	}
	actions, err := loadActions(db)
	if err != nil {
		return dashboardResponse{}, err
	}

	today := checkins[len(checkins)-1]
	today.CheckedIn += todaySubmissions
	if today.CheckedIn > today.TotalAirmen {
		today.CheckedIn = today.TotalAirmen
	}
	today.Completion = float64(today.CheckedIn) * 100 / float64(today.TotalAirmen)
	window := checkins
	if len(window) > 7 {
		window = window[len(window)-7:]
	}

	metrics := buildMetrics(checkins, window, todaySubmissions)
	trends := make([]trendPoint, 0, len(checkins))
	for _, row := range checkins {
		trends = append(trends, trendPoint{
			Day:        shortDay(row.Day),
			Readiness:  readinessScore(row.Completion, row.AvgMood, row.AvgStress, row.AvgSleep),
			Completion: row.Completion,
			Stress:     row.AvgStress,
			Sleep:      row.AvgSleep,
			CheckedIn:  row.CheckedIn,
		})
	}

	return dashboardResponse{
		Title:       "MindFlight leadership dashboard",
		Subtitle:    "Anonymous, aggregate-only readiness signals across the wing. Track trends, catch friction early, and direct support where the pattern says it matters.",
		GeneratedAt: time.Now().UTC(),
		Coverage: coverageStats{
			TotalAirmen:    today.TotalAirmen,
			CheckedInToday: today.CheckedIn,
			CompletionRate: today.Completion,
		},
		Metrics:        metrics,
		Trends:         trends,
		Stressors:      stressors,
		Resources:      resources,
		SupportRoutes:  supportRoutes,
		ModuleAdoption: moduleAdoption,
		Alerts:         alerts,
		Actions:        actions,
	}, nil
}

func buildAirmanDashboard(db *sql.DB, username string) (airmanDashboardResponse, error) {
	rows, err := db.Query(`SELECT created_at, mood, stress, sleep FROM checkin_submissions WHERE username = ? ORDER BY created_at DESC LIMIT 7`, username)
	if err != nil {
		return airmanDashboardResponse{}, err
	}
	defer rows.Close()

	var trend []trendPoint
	var latestMood, latestStress, latestSleep float64
	for rows.Next() {
		var createdAt string
		var mood, stress, sleep float64
		if err := rows.Scan(&createdAt, &mood, &stress, &sleep); err != nil {
			return airmanDashboardResponse{}, err
		}

		parsed, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return airmanDashboardResponse{}, err
		}

		trend = append(trend, trendPoint{
			Day:       parsed.Format("Jan 2"),
			Readiness: readinessScore(0, mood, stress, sleep),
			Stress:    stress,
			Sleep:     sleep,
			CheckedIn: 1,
		})
		if latestMood == 0 && latestStress == 0 && latestSleep == 0 {
			latestMood = mood
			latestStress = stress
			latestSleep = sleep
		}
	}
	if err := rows.Err(); err != nil {
		return airmanDashboardResponse{}, err
	}

	resources, err := loadBarItems(db, `SELECT name, count FROM resources ORDER BY count DESC`)
	if err != nil {
		return airmanDashboardResponse{}, err
	}
	modules, err := loadModules(db, username)
	if err != nil {
		return airmanDashboardResponse{}, err
	}
	wearable, err := loadWearableStatus(db, username)
	if err != nil {
		return airmanDashboardResponse{}, err
	}
	privacy, err := loadPrivacySettings(db, username)
	if err != nil {
		return airmanDashboardResponse{}, err
	}
	requests, err := loadSupportRequests(db, username)
	if err != nil {
		return airmanDashboardResponse{}, err
	}

	if len(trend) == 0 {
		trend = []trendPoint{{Day: "Today", Readiness: 0, Stress: 0, Sleep: 0}}
	}

	tips := []actionItem{
		{Title: "Take a reset break", Detail: "Two minutes to breathe, hydrate, and step away from the noise.", Owner: "MindFlight", Status: "Ready"},
		{Title: "Open sleep guidance", Detail: "Use the sleep tools when your recovery is slipping.", Owner: "MindFlight", Status: "Ready"},
		{Title: "Send a note to support", Detail: "If you want a human follow-up, use the discreet support path.", Owner: "MindFlight", Status: "Ready"},
	}
	coach, err := buildCoachInsight(db, username, "")
	if err != nil {
		return airmanDashboardResponse{}, err
	}
	peerSupport := defaultPeerSupport()
	peerSupport.Requests = requests

	return airmanDashboardResponse{
		Title:        "MindFlight check-in",
		Subtitle:     "Personal wellness entry point with your own history and tools.",
		GeneratedAt:  time.Now().UTC(),
		LatestMood:   latestMood,
		LatestStress: latestStress,
		LatestSleep:  latestSleep,
		Trend:        trend,
		Resources:    resources,
		Tips:         tips,
		Modules:      modules,
		Coach:        coach,
		Wearable:     wearable,
		PeerSupport:  peerSupport,
		Crisis:       defaultCrisisPayload(),
		Privacy:      privacy,
	}, nil
}

func buildMetrics(all, window []checkinRow, todaySubmissions int) []metricCard {
	avg := func(selector func(checkinRow) float64, rows []checkinRow) float64 {
		if len(rows) == 0 {
			return 0
		}
		var total float64
		for _, row := range rows {
			total += selector(row)
		}
		return total / float64(len(rows))
	}

	current := window[len(window)-1]
	previous := all
	if len(previous) > len(window) {
		previous = previous[:len(previous)-len(window)]
	}
	if len(previous) == 0 {
		previous = window
	}

	currentReadiness := readinessScore(current.Completion, current.AvgMood, current.AvgStress, current.AvgSleep)
	priorReadiness := avg(func(row checkinRow) float64 {
		return readinessScore(row.Completion, row.AvgMood, row.AvgStress, row.AvgSleep)
	}, previous)

	stressTrend := current.AvgStress - avg(func(row checkinRow) float64 { return row.AvgStress }, previous)
	completionTrend := current.Completion - avg(func(row checkinRow) float64 { return row.Completion }, previous)
	sleepTrend := current.AvgSleep - avg(func(row checkinRow) float64 { return row.AvgSleep }, previous)

	return []metricCard{
		{Label: "Readiness index", Value: fmt.Sprintf("%.0f", currentReadiness), Delta: signedDelta(currentReadiness-priorReadiness, "points vs prior window"), Tone: toneFor(currentReadiness, 70, 60)},
		{Label: "Check-in completion", Value: fmt.Sprintf("%.0f%%", current.Completion), Delta: signedDelta(completionTrend, "vs prior window"), Tone: toneFor(current.Completion, 88, 80)},
		{Label: "Average stress", Value: fmt.Sprintf("%.1f", current.AvgStress), Delta: signedDelta(stressTrend, "vs prior window"), Tone: toneForInverse(current.AvgStress, 4.3, 5.0)},
		{Label: "Sleep recovery", Value: fmt.Sprintf("%.1f h", current.AvgSleep), Delta: signedDelta(sleepTrend, "vs prior window"), Tone: toneFor(current.AvgSleep, 7.0, 6.5)},
		{Label: "New submissions", Value: fmt.Sprintf("%d", todaySubmissions), Delta: "anonymous check-ins today", Tone: toneFor(float64(todaySubmissions), 3, 1)},
	}
}

func countTodaySubmissions(db *sql.DB) (int, error) {
	var count int
	today := time.Now().UTC().Format("2006-01-02")
	if err := db.QueryRow(`SELECT COUNT(*) FROM checkin_submissions WHERE substr(created_at, 1, 10) = ?`, today).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func readinessScore(completion, mood, stress, sleep float64) float64 {
	score := 26 + (completion * 0.36) + (mood * 10) + (sleep * 4.5) - (stress * 8.25)
	return math.Round(clamp(score, 0, 100))
}

func loadBarItems(db *sql.DB, query string) ([]barItem, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []barItem
	for rows.Next() {
		var row valueRow
		if err := rows.Scan(&row.Name, &row.Count); err != nil {
			return nil, err
		}
		items = append(items, barItem{Label: row.Name, Value: row.Count})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].Value > items[j].Value })
	return items, nil
}

func loadAlerts(db *sql.DB) ([]alertItem, error) {
	rows, err := db.Query(`SELECT title, detail, severity FROM alerts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []alertItem
	for rows.Next() {
		var item alertItem
		if err := rows.Scan(&item.Title, &item.Detail, &item.Severity); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func loadActions(db *sql.DB) ([]actionItem, error) {
	rows, err := db.Query(`SELECT title, detail, owner, status FROM actions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []actionItem
	for rows.Next() {
		var item actionItem
		if err := rows.Scan(&item.Title, &item.Detail, &item.Owner, &item.Status); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func signedDelta(delta float64, suffix string) string {
	return fmt.Sprintf("%+0.1f %s", delta, suffix)
}

func toneFor(value, goodThreshold, warnThreshold float64) string {
	if value >= goodThreshold {
		return "good"
	}
	if value >= warnThreshold {
		return "warn"
	}
	return "alert"
}

func toneForInverse(value, goodThreshold, warnThreshold float64) string {
	if value <= goodThreshold {
		return "good"
	}
	if value <= warnThreshold {
		return "warn"
	}
	return "alert"
}

func shortDay(day string) string {
	parsed, err := time.Parse("2006-01-02", day)
	if err != nil {
		return day
	}
	return parsed.Format("Jan 2")
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
