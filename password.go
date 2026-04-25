package main

import (
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func (a *App) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		a.render(w, r, "password.html", map[string]interface{}{
			"Title": "Passwort ändern",
		})
		return
	}

	userID := a.getUserID(r)
	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	newPasswordConfirm := r.FormValue("new_password_confirm")

	if currentPassword == "" || newPassword == "" || newPasswordConfirm == "" {
		a.render(w, r, "password.html", map[string]interface{}{
			"Title": "Passwort ändern",
			"Error": "Alle Felder sind Pflicht",
		})
		return
	}

	if len(newPassword) < 6 {
		a.render(w, r, "password.html", map[string]interface{}{
			"Title": "Passwort ändern",
			"Error": "Neues Passwort muss mindestens 6 Zeichen haben",
		})
		return
	}

	if newPassword != newPasswordConfirm {
		a.render(w, r, "password.html", map[string]interface{}{
			"Title": "Passwort ändern",
			"Error": "Neue Passwörter stimmen nicht überein",
		})
		return
	}

	var currentHash string
	a.db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&currentHash)

	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(currentPassword)); err != nil {
		a.render(w, r, "password.html", map[string]interface{}{
			"Title": "Passwort ändern",
			"Error": "Aktuelles Passwort ist falsch",
		})
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("bcrypt error: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	_, err = a.db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", string(newHash), userID)
	if err != nil {
		log.Printf("DB error updating password: %v", err)
		a.render(w, r, "password.html", map[string]interface{}{
			"Title": "Passwort ändern",
			"Error": "Fehler beim Speichern",
		})
		return
	}

	a.render(w, r, "password.html", map[string]interface{}{
		"Title":   "Passwort ändern",
		"Success": "Passwort erfolgreich geändert!",
	})
}