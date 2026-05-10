package main

import (
	"database/sql"
	"fmt"
	"time"
)

type User struct {
	ID          int64
	Email       string
	PasswordHash string
	DisplayName string
	IsAdmin     bool
	IsVerified  bool
	VerifyToken string
	CreatedAt   time.Time
}

type Match struct {
	ID        int64
	Phase     string
	GroupName string
	HomeTeam  string
	AwayTeam  string
	MatchDate time.Time
	HomeGoals sql.NullInt64
	AwayGoals sql.NullInt64
	Finished  bool
	CreatedAt time.Time
}

type Tip struct {
	ID        int64
	UserID    int64
	MatchID   int64
	HomeGoals int
	AwayGoals int
	CreatedAt time.Time
	UpdatedAt time.Time
	// Joined
	HomeTeam       string
	AwayTeam       string
	MatchDate      time.Time
	MatchFinished  bool
	MatchHomeGoals sql.NullInt64
	MatchAwayGoals sql.NullInt64
}

type Comment struct {
	ID        int64
	UserID    int64
	MatchID   int64
	Text      string
	CreatedAt time.Time
	// Joined
	UserName string
}

type RankingEntry struct {
	Rank         int
	DisplayName  string
	TotalPoints  int
	ExactScores  int
	GoalDiffs    int
}

func (m *Match) CanTip() bool {
	return !m.Finished && time.Now().Before(m.MatchDate)
}

func (m *Match) HomeGoalsStr() string {
	if m.HomeGoals.Valid {
		return fmt.Sprintf("%d", m.HomeGoals.Int64)
	}
	return "-"
}

func (m *Match) AwayGoalsStr() string {
	if m.AwayGoals.Valid {
		return fmt.Sprintf("%d", m.AwayGoals.Int64)
	}
	return "-"
}

func (t *Tip) Points() (total, exact, diff int) {
	if !t.MatchHomeGoals.Valid || !t.MatchAwayGoals.Valid {
		return 0, 0, 0
	}
	hg, ag := int(t.MatchHomeGoals.Int64), int(t.MatchAwayGoals.Int64)
	if t.HomeGoals == hg && t.AwayGoals == ag {
		return 3, 1, 0
	}
	if (t.HomeGoals - t.AwayGoals) == (hg - ag) {
		return 1, 0, 1
	}
	return 0, 0, 0
}