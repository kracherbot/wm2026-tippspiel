package main

import (
	"net/http"
	"strconv"
)

func (a *App) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	var userCount, matchCount, tipCount, finishedCount int
	a.db.QueryRow("SELECT COUNT(*) FROM users WHERE is_verified = 1").Scan(&userCount)
	a.db.QueryRow("SELECT COUNT(*) FROM matches").Scan(&matchCount)
	a.db.QueryRow("SELECT COUNT(DISTINCT user_id || match_id) FROM tips").Scan(&tipCount)
	a.db.QueryRow("SELECT COUNT(*) FROM matches WHERE finished = 1").Scan(&finishedCount)

	a.render(w, r, "admin_dashboard.html", map[string]interface{}{
		"Title":         "Admin Dashboard",
		"UserCount":     userCount,
		"MatchCount":    matchCount,
		"TipCount":      tipCount,
		"FinishedCount": finishedCount,
	})
}

func (a *App) handleAdminMatches(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		rows, err := a.db.Query("SELECT id, phase, group_name, home_team, away_team, match_date, home_goals, away_goals, finished FROM matches ORDER BY match_date ASC")
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type MatchRow struct {
			ID        int64
			Phase     string
			GroupName string
			HomeTeam  string
			AwayTeam  string
			MatchDate string
			Finished  bool
		}

		var matches []MatchRow
		for rows.Next() {
			var mr MatchRow
			var m Match
			rows.Scan(&m.ID, &m.Phase, &m.GroupName, &m.HomeTeam, &m.AwayTeam, &m.MatchDate, &m.HomeGoals, &m.AwayGoals, &m.Finished)
			mr.ID = m.ID
			mr.Phase = m.Phase
			mr.GroupName = m.GroupName
			mr.HomeTeam = m.HomeTeam
			mr.AwayTeam = m.AwayTeam
			mr.MatchDate = formatTimeFull(m.MatchDate)
			mr.Finished = m.Finished
			matches = append(matches, mr)
		}

		a.render(w, r, "admin_matches.html", map[string]interface{}{
			"Title":   "Spiele verwalten",
			"Matches": matches,
		})
		return
	}
}

func (a *App) handleAdminMatchesSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.FormValue("id")
	homeTeam := r.FormValue("home_team")
	awayTeam := r.FormValue("away_team")

	if id == "" {
		phase := r.FormValue("phase")
		groupName := r.FormValue("group_name")
		matchDate := r.FormValue("match_date")
		a.db.Exec("INSERT INTO matches (phase, group_name, home_team, away_team, match_date) VALUES (?, ?, ?, ?, ?)",
			phase, groupName, homeTeam, awayTeam, matchDate)
	} else {
		a.db.Exec("UPDATE matches SET home_team = ?, away_team = ? WHERE id = ?", homeTeam, awayTeam, id)
	}

	a.redirectWithMsg(w, r, "/admin/matches", "Spiel+gespeichert", "success")
}

func (a *App) handleAdminResults(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		rows, err := a.db.Query("SELECT id, phase, group_name, home_team, away_team, match_date, home_goals, away_goals, finished FROM matches WHERE finished = 0 ORDER BY match_date ASC")
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type MatchRow struct {
			ID        int64
			Phase     string
			GroupName string
			HomeTeam  string
			AwayTeam  string
			MatchDate string
			Finished  bool
		}

		phaseOrder := make([]string, 0)
		byPhase := make(map[string][]MatchRow)
		phaseSeen := make(map[string]bool)

		for rows.Next() {
			var m Match
			rows.Scan(&m.ID, &m.Phase, &m.GroupName, &m.HomeTeam, &m.AwayTeam, &m.MatchDate, &m.HomeGoals, &m.AwayGoals, &m.Finished)

			phaseKey := m.Phase
			if m.GroupName != "" && m.Phase == "Gruppenphase" {
				phaseKey = "Gruppe " + m.GroupName
			}

			mr := MatchRow{
				ID: m.ID, Phase: m.Phase, GroupName: m.GroupName,
				HomeTeam: m.HomeTeam, AwayTeam: m.AwayTeam,
				MatchDate: formatTimeFull(m.MatchDate), Finished: m.Finished,
			}

			if !phaseSeen[phaseKey] {
				phaseSeen[phaseKey] = true
				phaseOrder = append(phaseOrder, phaseKey)
			}
			byPhase[phaseKey] = append(byPhase[phaseKey], mr)
		}

		a.render(w, r, "admin_results.html", map[string]interface{}{
			"Title":      "Ergebnisse eintragen",
			"PhaseOrder": phaseOrder,
			"ByPhase":    byPhase,
		})
		return
	}
}
func (a *App) handleAdminResultsSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	matchID := r.FormValue("match_id")
	homeGoals := r.FormValue("home_goals")
	awayGoals := r.FormValue("away_goals")

	hg, _ := strconv.Atoi(homeGoals)
	ag, _ := strconv.Atoi(awayGoals)

	a.db.Exec("UPDATE matches SET home_goals = ?, away_goals = ?, finished = 1 WHERE id = ?", hg, ag, matchID)

	a.redirectWithMsg(w, r, "/admin/results", "Ergebnis+gespeichert!+Punkte+wurden+aktualisiert.", "success")
}

func (a *App) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query("SELECT id, email, display_name, is_admin, is_verified, created_at FROM users ORDER BY id ASC")
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type UserRow struct {
		ID          int64
		Email       string
		DisplayName string
		IsAdmin     bool
		IsVerified  bool
		CreatedAt   string
	}

	var users []UserRow
	for rows.Next() {
		var u UserRow
		rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.IsAdmin, &u.IsVerified, &u.CreatedAt)
		u.CreatedAt = u.CreatedAt[:19]
		users = append(users, u)
	}

	a.render(w, r, "admin_users.html", map[string]interface{}{
		"Title": "Benutzer verwalten",
		"Users": users,
	})
}

func (a *App) handleAdminToggleAdmin(w http.ResponseWriter, r *http.Request) {
	userID := r.FormValue("user_id")
	a.db.Exec("UPDATE users SET is_admin = NOT is_admin WHERE id = ?", userID)
	a.redirectWithMsg(w, r, "/admin/users", "Admin-Status+geändert", "success")
}

func (a *App) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := r.FormValue("user_id")
	currentID := a.getUserID(r)
	if userID == strconv.FormatInt(currentID, 10) {
		a.redirectWithMsg(w, r, "/admin/users", "Du+kannst+dich+nicht+selbst+löschen", "error")
		return
	}
	a.db.Exec("DELETE FROM tips WHERE user_id = ?", userID)
	a.db.Exec("DELETE FROM comments WHERE user_id = ?", userID)
	a.db.Exec("DELETE FROM users WHERE id = ?", userID)
	a.redirectWithMsg(w, r, "/admin/users", "Benutzer+gelöscht", "success")
}