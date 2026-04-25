package main

import (
	"net/http"
	"strconv"
)

func (a *App) handleMatchDetail(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	matchIDStr := path[len("/match/"):]
	matchID, err := strconv.ParseInt(matchIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Ungültige Match-ID", http.StatusBadRequest)
		return
	}

	var m Match
	err = a.db.QueryRow("SELECT id, phase, group_name, home_team, away_team, match_date, home_goals, away_goals, finished FROM matches WHERE id = ?", matchID).Scan(
		&m.ID, &m.Phase, &m.GroupName, &m.HomeTeam, &m.AwayTeam, &m.MatchDate, &m.HomeGoals, &m.AwayGoals, &m.Finished,
	)
	if err != nil {
		http.Error(w, "Spiel nicht gefunden", http.StatusNotFound)
		return
	}

	rows, err := a.db.Query(`
		SELECT c.id, c.user_id, c.match_id, c.text, c.created_at, u.display_name
		FROM comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.match_id = ?
		ORDER BY c.created_at ASC
	`, matchID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		rows.Scan(&c.ID, &c.UserID, &c.MatchID, &c.Text, &c.CreatedAt, &c.UserName)
		comments = append(comments, c)
	}

	a.render(w, r, "match_detail.html", map[string]interface{}{
		"Title":    m.HomeTeam + " vs " + m.AwayTeam,
		"Match":    m,
		"Comments": comments,
	})
}

func (a *App) handleCommentAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := a.getUserID(r)
	matchID := r.FormValue("match_id")
	text := r.FormValue("text")

	if matchID == "" || text == "" {
		a.redirectWithMsg(w, r, "/match/"+matchID, "Kommentar+darf+nicht+leer+sein", "error")
		return
	}

	a.db.Exec("INSERT INTO comments (user_id, match_id, text) VALUES (?, ?, ?)", userID, matchID, text)

	http.Redirect(w, r, "/match/"+matchID, http.StatusSeeOther)
}