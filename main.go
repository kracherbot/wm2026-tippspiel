package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

//go:embed templates/* templates/admin/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

type contextKey string

const ctxUserID contextKey = "userID"

type AppConfig struct {
	Port       string
	DBPath     string
	SMTPHost   string
	SMTPPort   string
	SMTPUser   string
	SMTPPass   string
	AdminEmail string
	BaseURL    string
	JWTSecret  string
}

type App struct {
	config *AppConfig
	db     *sql.DB
	mailer *Mailer
	tmpl   *template.Template
}

func main() {
	config := &AppConfig{
		Port:       getEnv("PORT", "8080"),
		DBPath:     getEnv("DB_PATH", "/data/tippspiel.db"),
		SMTPHost:   getEnv("SMTP_HOST", "mail.walsi.org"),
		SMTPPort:   getEnv("SMTP_PORT", "587"),
		SMTPUser:   getEnv("SMTP_USER", ""),
		SMTPPass:   getEnv("SMTP_PASS", ""),
		AdminEmail: getEnv("ADMIN_EMAIL", "admin.wm2026@walsi.org"),
		BaseURL:    getEnv("BASE_URL", "https://wm2026.walsi.org"),
		JWTSecret:  getEnv("JWT_SECRET", "wm2026-tippspiel-secret-change-me"),
	}

	db, err := sql.Open("sqlite3", config.DBPath+"?_loc=UTC&_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := migrate(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	funcs := template.FuncMap{
		"formatTime":    formatTime,
		"formatTimeFull": formatTimeFull,
	}

	// Parse all templates
	tmpl, err := template.New("").Funcs(funcs).ParseFS(templateFS, "templates/*.html", "templates/admin/*.html")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	app := &App{
		config: config,
		db:     db,
		mailer: &Mailer{config: config},
		tmpl:   tmpl,
	}

	app.ensureAdmin()

	mux := http.NewServeMux()

	staticContent, _ := fs.Sub(staticFS, "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticContent))))

	mux.HandleFunc("/", app.handleHome)
	mux.HandleFunc("/login", app.handleLogin)
	mux.HandleFunc("/register", app.handleRegister)
	mux.HandleFunc("/verify", app.handleVerify)
	mux.HandleFunc("/logout", app.handleLogout)

	mux.HandleFunc("/dashboard", app.auth(app.handleDashboard))
	mux.HandleFunc("/tippen", app.auth(app.handleTippen))
	mux.HandleFunc("/tippen/save", app.auth(app.handleTippenSave))
	mux.HandleFunc("/ranking", app.auth(app.handleRanking))
	mux.HandleFunc("/regeln", app.auth(app.handleRegeln))
	mux.HandleFunc("/match/", app.auth(app.handleMatchDetail))
	mux.HandleFunc("/comment/add", app.auth(app.handleCommentAdd))
	mux.HandleFunc("/password", app.auth(app.handleChangePassword))

	mux.HandleFunc("/admin", app.auth(app.admin(app.handleAdminDashboard)))
	mux.HandleFunc("/admin/matches", app.auth(app.admin(app.handleAdminMatches)))
	mux.HandleFunc("/admin/matches/save", app.auth(app.admin(app.handleAdminMatchesSave)))
	mux.HandleFunc("/admin/results", app.auth(app.admin(app.handleAdminResults)))
	mux.HandleFunc("/admin/results/save", app.auth(app.admin(app.handleAdminResultsSave)))
	mux.HandleFunc("/admin/users", app.auth(app.admin(app.handleAdminUsers)))
	mux.HandleFunc("/admin/users/toggle-admin", app.auth(app.admin(app.handleAdminToggleAdmin)))
	mux.HandleFunc("/admin/users/delete", app.auth(app.admin(app.handleAdminDeleteUser)))

	log.Printf("⚽ WM 2026 Tippspiel starting on :%s", config.Port)
	server := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func (a *App) ensureAdmin() {
	var count int
	a.db.QueryRow("SELECT COUNT(*) FROM users WHERE is_admin = 1").Scan(&count)
	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin2026"), bcrypt.DefaultCost)
		token := generateToken()
		a.db.Exec(
			"INSERT INTO users (email, password_hash, display_name, is_admin, is_verified, verify_token) VALUES (?, ?, ?, 1, 1, ?)",
			a.config.AdminEmail, string(hash), "Admin", token,
		)
		log.Println("Created default admin user:", a.config.AdminEmail)
	}
}

func (a *App) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		userID, err := parseJWT(cookie.Value, a.config.JWTSecret)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		var verified bool
		a.db.QueryRow("SELECT is_verified FROM users WHERE id = ?", userID).Scan(&verified)
		if !verified {
			a.render(w, r, "verify_remind.html", map[string]interface{}{
				"Email": a.getUserEmail(userID),
			})
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, userID)
		next(w, r.WithContext(ctx))
	}
}

func (a *App) admin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(ctxUserID).(int64)
		var isAdmin bool
		a.db.QueryRow("SELECT is_admin FROM users WHERE id = ?", userID).Scan(&isAdmin)
		if !isAdmin {
			http.Redirect(w, r, "/ranking?msg=Nur+Admins+dürfen+das+Admin-Dashboard+aufrufen&msgType=error", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func (a *App) getUserID(r *http.Request) int64 {
	return r.Context().Value(ctxUserID).(int64)
}

func (a *App) getUserEmail(userID int64) string {
	var email string
	a.db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&email)
	return email
}

func (a *App) getUser(userID int64) *User {
	u := &User{}
	a.db.QueryRow("SELECT id, email, display_name, is_admin, is_verified FROM users WHERE id = ?", userID).Scan(&u.ID, &u.Email, &u.DisplayName, &u.IsAdmin, &u.IsVerified)
	return u
}

func (a *App) render(w http.ResponseWriter, r *http.Request, name string, data map[string]interface{}) {
	userID := r.Context().Value(ctxUserID)
	if userID != nil {
		data["User"] = a.getUser(userID.(int64))
	}
	data["CurrentPath"] = r.URL.Path

	if msg := r.URL.Query().Get("msg"); msg != "" {
		data["FlashMessage"] = msg
	}
	if msgType := r.URL.Query().Get("msgType"); msgType != "" {
		data["FlashType"] = msgType
	}

	if err := a.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Template error %s: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (a *App) redirectWithMsg(w http.ResponseWriter, r *http.Request, url, msg, msgType string) {
	http.Redirect(w, r, fmt.Sprintf("%s?msg=%s&msgType=%s", url, msg, msgType), http.StatusSeeOther)
}

func formatTime(t time.Time) string {
	loc, _ := time.LoadLocation("Europe/Zurich")
	return t.In(loc).Format("02.01. 15:04")
}

func formatTimeFull(t time.Time) string {
	loc, _ := time.LoadLocation("Europe/Zurich")
	return t.In(loc).Format("02.01.2006 15:04")
}