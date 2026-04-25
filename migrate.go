package main

import (
	"database/sql"
	"log"
)

func migrate(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			display_name TEXT NOT NULL,
			is_admin INTEGER DEFAULT 0,
			is_verified INTEGER DEFAULT 0,
			verify_token TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS matches (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			phase TEXT NOT NULL,
			group_name TEXT DEFAULT '',
			home_team TEXT NOT NULL,
			away_team TEXT NOT NULL,
			match_date DATETIME NOT NULL,
			home_goals INTEGER,
			away_goals INTEGER,
			finished INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS tips (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			match_id INTEGER NOT NULL,
			home_goals INTEGER NOT NULL,
			away_goals INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, match_id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (match_id) REFERENCES matches(id)
		)`,
		`CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			match_id INTEGER NOT NULL,
			text TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (match_id) REFERENCES matches(id)
		)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return err
		}
	}

	// Seed matches if empty
	var count int
	db.QueryRow("SELECT COUNT(*) FROM matches").Scan(&count)
	if count == 0 {
		log.Println("Seeding WM 2026 matches...")
		seedMatches(db)
	}

	return nil
}