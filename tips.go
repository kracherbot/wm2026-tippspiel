package main

import (
	"net/http"
	"sort"
	"time"
)

func (a *App) handleDashboard(w http.ResponseWriter, r *http.Request) {
	userID := a.getUserID(r)
	user := a.getUser(userID)

	// Get upcoming matches without tips
	rows, err := a.db.Query(`
		SELECT m.id, m.phase, m.group_name, m.home_team, m.away_team, m.match_date, m.home_goals, m.away_goals, m.finished
		FROM matches m
		WHERE m.match_date > ? AND m.finished = 0
		AND m.id NOT IN (SELECT match_id FROM tips WHERE user_id = ?)
		ORDER BY m.match_date ASC
		LIMIT 10
	`, time.Now(), userID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var upcoming []Match
	for rows.Next() {
		var m Match
		rows.Scan(&m.ID, &m.Phase, &m.GroupName, &m.HomeTeam, &m.AwayTeam, &m.MatchDate, &m.HomeGoals, &m.AwayGoals, &m.Finished)
		upcoming = append(upcoming, m)
	}

	// Get user ranking
	var rank int
	a.db.QueryRow(`
		SELECT COUNT(*) + 1 FROM (
			SELECT u.id, COALESCE(SUM(CASE
				WHEN t.home_goals = m.home_goals AND t.away_goals = m.away_goals THEN 3
				WHEN ((t.home_goals > t.away_goals AND m.home_goals > m.away_goals) OR (t.home_goals < t.away_goals AND m.home_goals < m.away_goals) OR (t.home_goals = t.away_goals AND m.home_goals = m.away_goals)) THEN 1
				ELSE 0
			END), 0) as pts
			FROM users u
			LEFT JOIN tips t ON t.user_id = u.id
			LEFT JOIN matches m ON m.id = t.match_id AND m.finished = 1
			WHERE u.is_verified = 1
			GROUP BY u.id
		) sub WHERE pts > (
			SELECT COALESCE(SUM(CASE
				WHEN t.home_goals = m.home_goals AND t.away_goals = m.away_goals THEN 3
				WHEN ((t.home_goals > t.away_goals AND m.home_goals > m.away_goals) OR (t.home_goals < t.away_goals AND m.home_goals < m.away_goals) OR (t.home_goals = t.away_goals AND m.home_goals = m.away_goals)) THEN 1
				ELSE 0
			END), 0)
			FROM tips t
			LEFT JOIN matches m ON m.id = t.match_id AND m.finished = 1
			WHERE t.user_id = ?
		)
	`, userID).Scan(&rank)

	a.render(w, r, "dashboard.html", map[string]interface{}{
		"Title":     "Dashboard",
		"User":      user,
		"Upcoming":  upcoming,
		"Rank":      rank,
	})
}

func (a *App) handleTippen(w http.ResponseWriter, r *http.Request) {
	userID := a.getUserID(r)

	// Get all matches with user's tips AND comment count
	rows, err := a.db.Query(`
		SELECT m.id, m.phase, m.group_name, m.home_team, m.away_team, m.match_date, m.home_goals, m.away_goals, m.finished,
			COALESCE(t.home_goals, -1), COALESCE(t.away_goals, -1),
			COALESCE(c.cnt, 0)
		FROM matches m
		LEFT JOIN tips t ON t.match_id = m.id AND t.user_id = ?
		LEFT JOIN (SELECT match_id, COUNT(*) as cnt FROM comments GROUP BY match_id) c ON c.match_id = m.id
		ORDER BY m.match_date ASC
	`, userID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type MatchWithTip struct {
		Match        Match
		TipHome      int
		TipAway      int
		HasTip       bool
		CanTip       bool
		CommentCount int
	}

	var all []MatchWithTip
	byPhase := make(map[string][]MatchWithTip)
	var phaseOrder []string

	for rows.Next() {
		var m Match
		var tipH, tipA, commentCount int
		rows.Scan(&m.ID, &m.Phase, &m.GroupName, &m.HomeTeam, &m.AwayTeam, &m.MatchDate, &m.HomeGoals, &m.AwayGoals, &m.Finished, &tipH, &tipA, &commentCount)

		hasTip := tipH >= 0
		canTip := !m.Finished && time.Now().Before(m.MatchDate)

		mwt := MatchWithTip{Match: m, TipHome: tipH, TipAway: tipA, HasTip: hasTip, CanTip: canTip, CommentCount: commentCount}
		all = append(all, mwt)

		key := m.Phase
		if m.GroupName != "" {
			key = "Gruppe " + m.GroupName
		}
		if _, exists := byPhase[key]; !exists {
			phaseOrder = append(phaseOrder, key)
		}
		byPhase[key] = append(byPhase[key], mwt)
	}

	a.render(w, r, "tippen.html", map[string]interface{}{
		"Title":      "Tippen",
		"PhaseOrder": phaseOrder,
		"ByPhase":     byPhase,
		"All":         all,
	})
}

func (a *App) handleTippenSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := a.getUserID(r)
	matchID := r.FormValue("match_id")
	homeGoals := r.FormValue("home_goals")
	awayGoals := r.FormValue("away_goals")

	if matchID == "" || homeGoals == "" || awayGoals == "" {
		a.redirectWithMsg(w, r, "/tippen", "Ungültige+Eingabe", "error")
		return
	}

	// Check if match can still be tipped
	var matchDate time.Time
	var finished bool
	a.db.QueryRow("SELECT match_date, finished FROM matches WHERE id = ?", matchID).Scan(&matchDate, &finished)

	if finished || time.Now().After(matchDate) {
		a.redirectWithMsg(w, r, "/tippen", "Tipp-Frist+abgelaufen", "error")
		return
	}

	// Upsert tip
	a.db.Exec(`
		INSERT INTO tips (user_id, match_id, home_goals, away_goals, updated_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, match_id) DO UPDATE SET home_goals = excluded.home_goals, away_goals = excluded.away_goals, updated_at = CURRENT_TIMESTAMP
	`, userID, matchID, homeGoals, awayGoals)

	a.redirectWithMsg(w, r, "/tippen", "Tipp+gespeichert!", "success")
}

func (a *App) handleRanking(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(`
		SELECT u.display_name,
			SUM(CASE WHEN t.home_goals = m.home_goals AND t.away_goals = m.away_goals THEN 1 ELSE 0 END) as exact,
			SUM(CASE WHEN ((t.home_goals > t.away_goals AND m.home_goals > m.away_goals) OR (t.home_goals < t.away_goals AND m.home_goals < m.away_goals) OR (t.home_goals = t.away_goals AND m.home_goals = m.away_goals))
				AND NOT (t.home_goals = m.home_goals AND t.away_goals = m.away_goals) THEN 1 ELSE 0 END) as diff
		FROM users u
		LEFT JOIN tips t ON t.user_id = u.id
		LEFT JOIN matches m ON m.id = t.match_id AND m.finished = 1
		WHERE u.is_verified = 1
		GROUP BY u.id
		ORDER BY (exact * 3 + diff) DESC, u.display_name ASC
	`)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var rankings []RankingEntry
	rank := 1
	for rows.Next() {
		var re RankingEntry
		rows.Scan(&re.DisplayName, &re.ExactScores, &re.GoalDiffs)
		re.Rank = rank
		re.TotalPoints = re.ExactScores*3 + re.GoalDiffs
		rankings = append(rankings, re)
		rank++
	}

	// Sort by total points desc
	sort.Slice(rankings, func(i, j int) bool {
		if rankings[i].TotalPoints != rankings[j].TotalPoints {
			return rankings[i].TotalPoints > rankings[j].TotalPoints
		}
		return rankings[i].DisplayName < rankings[j].DisplayName
	})

	// Re-assign ranks
	for i := range rankings {
		if i > 0 && rankings[i].TotalPoints == rankings[i-1].TotalPoints {
			rankings[i].Rank = rankings[i-1].Rank
		} else {
			rankings[i].Rank = i + 1
		}
	}

	a.render(w, r, "ranking.html", map[string]interface{}{
		"Title":    "Tabelle",
		"Rankings": rankings,
	})
}
func (a *App) handleRegeln(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "regeln.html", map[string]interface{}{
		"Title": "Regeln",
	})
}
