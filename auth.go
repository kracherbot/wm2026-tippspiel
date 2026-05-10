package main

import (
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		a.render(w, r, "login.html", map[string]interface{}{
			"Title": "Anmelden",
		})
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	log.Printf("LOGIN attempt: email=%q len(password)=%d", email, len(password))

	var user User
	err := a.db.QueryRow("SELECT id, email, password_hash, display_name, is_admin, is_verified FROM users WHERE email = ?", email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName, &user.IsAdmin, &user.IsVerified,
	)
	if err != nil {
		log.Printf("LOGIN failed: user not found for email=%q err=%v", email, err)
		a.render(w, r, "login.html", map[string]interface{}{
			"Title": "Anmelden",
			"Error":  "E-Mail oder Passwort falsch",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		log.Printf("LOGIN failed: password mismatch for email=%q err=%v", email, err)
		a.render(w, r, "login.html", map[string]interface{}{
			"Title": "Anmelden",
			"Error":  "E-Mail oder Passwort falsch",
		})
		return
	}

	log.Printf("LOGIN success: email=%q userID=%d isAdmin=%v", email, user.ID, user.IsAdmin)

	token := createJWT(user.ID, a.config.JWTSecret)
	// Determine if request is over HTTPS (NPM terminates SSL, so check X-Forwarded-Proto)
	isSecure := r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		MaxAge:   7 * 24 * 3600,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/ranking", http.StatusSeeOther)
}

func (a *App) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		a.render(w, r, "register.html", map[string]interface{}{
			"Title": "Registrieren",
		})
		return
	}

	email := r.FormValue("email")
	displayName := r.FormValue("display_name")
	password := r.FormValue("password")
	passwordConfirm := r.FormValue("password_confirm")

	if email == "" || displayName == "" || password == "" {
		a.render(w, r, "register.html", map[string]interface{}{
			"Title": "Registrieren",
			"Error": "Alle Felder sind Pflicht",
			"Email": email,
			"Name":  displayName,
		})
		return
	}

	if len(password) < 6 {
		a.render(w, r, "register.html", map[string]interface{}{
			"Title": "Registrieren",
			"Error": "Passwort muss mindestens 6 Zeichen haben",
			"Email": email,
			"Name":  displayName,
		})
		return
	}

	if password != passwordConfirm {
		a.render(w, r, "register.html", map[string]interface{}{
			"Title": "Registrieren",
			"Error": "Passwörter stimmen nicht überein",
			"Email": email,
			"Name":  displayName,
		})
		return
	}

	var existing int
	a.db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&existing)
	if existing > 0 {
		a.render(w, r, "register.html", map[string]interface{}{
			"Title": "Registrieren",
			"Error": "Diese E-Mail ist bereits registriert",
			"Email": email,
			"Name":  displayName,
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	token := generateToken()
	_, err = a.db.Exec(
		"INSERT INTO users (email, password_hash, display_name, is_admin, is_verified, verify_token) VALUES (?, ?, ?, 0, 0, ?)",
		email, string(hash), displayName, token,
	)
	if err != nil {
		a.render(w, r, "register.html", map[string]interface{}{
			"Title": "Registrieren",
			"Error": "Fehler bei der Registrierung",
			"Email": email,
			"Name":  displayName,
		})
		return
	}

	go func() {
		if err := a.mailer.SendVerification(email, token); err != nil {
			log.Printf("SMTP send error: %v", err)
		}
	}()

	a.render(w, r, "register.html", map[string]interface{}{
		"Title":   "Registrieren",
		"Success": "Registrierung erfolgreich! Bitte überprüfe dein Postfach und klicke auf den Bestätigungslink.",
	})
}

func (a *App) handleVerify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Ungültiger Link", http.StatusBadRequest)
		return
	}

	var userID int64
	err := a.db.QueryRow("SELECT id FROM users WHERE verify_token = ? AND is_verified = 0", token).Scan(&userID)
	if err != nil {
		a.render(w, r, "login.html", map[string]interface{}{
			"Title": "Anmelden",
			"Error": "Ungültiger oder abgelaufener Bestätigungslink",
		})
		return
	}

	a.db.Exec("UPDATE users SET is_verified = 1, verify_token = '' WHERE id = ?", userID)

	isSecure := r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil

	sessionToken := createJWT(userID, a.config.JWTSecret)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionToken,
		Path:     "/",
		MaxAge:   7 * 24 * 3600,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/ranking?msg=E-Mail+bestätigt!+Willkommen+beim+Tippspiel!&msgType=success", http.StatusSeeOther)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) handleHome(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		userID, jwtErr := parseJWT(cookie.Value, a.config.JWTSecret)
		if jwtErr == nil {
			var verified bool
			a.db.QueryRow("SELECT is_verified FROM users WHERE id = ?", userID).Scan(&verified)
			if verified {
				http.Redirect(w, r, "/ranking", http.StatusSeeOther)
				return
			}
		}
	}

	var userCount, matchCount int
	a.db.QueryRow("SELECT COUNT(*) FROM users WHERE is_verified = 1").Scan(&userCount)
	a.db.QueryRow("SELECT COUNT(*) FROM matches").Scan(&matchCount)

	a.render(w, r, "home.html", map[string]interface{}{
		"Title":      "WM 2026 Tippspiel",
		"UserCount":  userCount,
		"MatchCount": matchCount,
	})
}