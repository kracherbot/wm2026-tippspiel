package main

import (
	"database/sql"
	"log"
	"time"
)

func seedMatches(db *sql.DB) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	// WM 2026: Official FIFA match schedule from WCup_2026_4.2.5_de.xlsx
	// Teams per group (draw positions 1-4)
	groups := map[string][]string{
		"A": {"Mexiko", "Südafrika", "Südkorea", "Tschechien"},
		"B": {"Kanada", "Bosnien-Herzegowina", "Katar", "Schweiz"},
		"C": {"Brasilien", "Marokko", "Haiti", "Schottland"},
		"D": {"USA", "Paraguay", "Australien", "Türkei"},
		"E": {"Deutschland", "Curaçao", "Elfenbeinküste", "Ecuador"},
		"F": {"Niederlande", "Japan", "Schweden", "Tunesien"},
		"G": {"Belgien", "Ägypten", "Iran", "Neuseeland"},
		"H": {"Spanien", "Kap Verde", "Saudi-Arabien", "Uruguay"},
		"I": {"Frankreich", "Senegal", "Irak", "Norwegen"},
		"J": {"Argentinien", "Algerien", "Österreich", "Jordanien"},
		"K": {"Portugal", "DR Kongo", "Usbekistan", "Kolumbien"},
		"L": {"England", "Kroatien", "Ghana", "Panama"},
	}

	stmt, err := tx.Prepare("INSERT INTO matches (phase, group_name, home_team, away_team, match_date) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	loc, _ := time.LoadLocation("Europe/Zurich")

	// Group stage: exact FIFA schedule (Matches 1-72)
	// {matchNo, group, homeDrawPos, awayDrawPos, "MEZ date"}
	type gm struct {
		g    string
		hp   int
		ap   int
		dmez string
	}

	groupMatches := []gm{
		// --- Spieltag 1 (11.-18. Juni) ---
		{"A", 1, 2, "2026-06-11 21:00"}, // M1:  Mexiko - Südafrika
		{"A", 3, 4, "2026-06-12 04:00"}, // M2:  Südkorea - Tschechien
		{"B", 1, 2, "2026-06-12 21:00"}, // M3:  Kanada - Bosnien-Herzegowina
		{"D", 1, 2, "2026-06-13 03:00"}, // M4:  USA - Paraguay
		{"C", 3, 4, "2026-06-14 03:00"}, // M5:  Haiti - Schottland
		{"D", 3, 4, "2026-06-14 06:00"}, // M6:  Australien - Türkei
		{"C", 1, 2, "2026-06-14 00:00"}, // M7:  Brasilien - Marokko
		{"B", 3, 4, "2026-06-13 21:00"}, // M8:  Katar - Schweiz
		{"E", 3, 4, "2026-06-15 01:00"}, // M9:  Elfenbeinküste - Ecuador
		{"E", 1, 2, "2026-06-14 19:00"}, // M10: Deutschland - Curaçao
		{"F", 1, 2, "2026-06-14 22:00"}, // M11: Niederlande - Japan
		{"F", 3, 4, "2026-06-15 04:00"}, // M12: Schweden - Tunesien
		{"H", 3, 4, "2026-06-16 00:00"}, // M13: Saudi-Arabien - Uruguay
		{"H", 1, 2, "2026-06-15 18:00"}, // M14: Spanien - Kap Verde
		{"G", 3, 4, "2026-06-16 03:00"}, // M15: Iran - Neuseeland
		{"G", 1, 2, "2026-06-15 21:00"}, // M16: Belgien - Ägypten
		{"I", 1, 2, "2026-06-16 21:00"}, // M17: Frankreich - Senegal
		{"I", 3, 4, "2026-06-17 00:00"}, // M18: Irak - Norwegen
		{"J", 1, 2, "2026-06-17 03:00"}, // M19: Argentinien - Algerien
		{"J", 3, 4, "2026-06-17 06:00"}, // M20: Österreich - Jordanien
		{"L", 3, 4, "2026-06-18 01:00"}, // M21: Ghana - Panama
		{"L", 1, 2, "2026-06-17 22:00"}, // M22: England - Kroatien
		{"K", 1, 2, "2026-06-17 19:00"}, // M23: Portugal - DR Kongo
		{"K", 3, 4, "2026-06-18 04:00"}, // M24: Usbekistan - Kolumbien
		// --- Spieltag 2 (18.-23. Juni) ---
		{"A", 4, 2, "2026-06-18 18:00"}, // M25: Tschechien - Südafrika
		{"B", 4, 2, "2026-06-18 21:00"}, // M26: Schweiz - Bosnien-Herzegowina
		{"B", 1, 3, "2026-06-19 00:00"}, // M27: Kanada - Katar
		{"A", 1, 3, "2026-06-19 03:00"}, // M28: Mexiko - Südkorea
		{"C", 1, 3, "2026-06-20 02:30"}, // M29: Brasilien - Haiti
		{"C", 4, 2, "2026-06-20 00:00"}, // M30: Schottland - Marokko
		{"D", 4, 2, "2026-06-20 05:00"}, // M31: Türkei - Paraguay
		{"D", 1, 3, "2026-06-19 21:00"}, // M32: USA - Australien
		{"E", 1, 3, "2026-06-20 22:00"}, // M33: Deutschland - Elfenbeinküste
		{"E", 4, 2, "2026-06-21 02:00"}, // M34: Ecuador - Curaçao
		{"F", 1, 3, "2026-06-20 19:00"}, // M35: Niederlande - Schweden
		{"F", 4, 2, "2026-06-21 06:00"}, // M36: Tunesien - Japan
		{"H", 4, 2, "2026-06-22 00:00"}, // M37: Uruguay - Kap Verde
		{"H", 1, 3, "2026-06-21 18:00"}, // M38: Spanien - Saudi-Arabien
		{"G", 1, 3, "2026-06-21 21:00"}, // M39: Belgien - Iran
		{"G", 4, 2, "2026-06-22 03:00"}, // M40: Neuseeland - Ägypten
		{"I", 4, 2, "2026-06-23 02:00"}, // M41: Norwegen - Senegal
		{"I", 1, 3, "2026-06-22 23:00"}, // M42: Frankreich - Irak
		{"J", 1, 3, "2026-06-22 19:00"}, // M43: Argentinien - Österreich
		{"J", 4, 2, "2026-06-23 05:00"}, // M44: Jordanien - Algerien
		{"L", 1, 3, "2026-06-23 22:00"}, // M45: England - Ghana
		{"L", 4, 2, "2026-06-24 01:00"}, // M46: Panama - Kroatien
		{"K", 1, 3, "2026-06-23 19:00"}, // M47: Portugal - Usbekistan
		{"K", 4, 2, "2026-06-24 04:00"}, // M48: Kolumbien - DR Kongo
		// --- Spieltag 3 (24.-28. Juni) ---
		{"C", 4, 1, "2026-06-25 00:00"}, // M49: Schottland - Brasilien
		{"C", 2, 3, "2026-06-25 00:00"}, // M50: Marokko - Haiti
		{"B", 4, 1, "2026-06-24 21:00"}, // M51: Schweiz - Kanada
		{"B", 2, 3, "2026-06-24 21:00"}, // M52: Bosnien-Herzegowina - Katar
		{"A", 4, 1, "2026-06-25 03:00"}, // M53: Tschechien - Mexiko
		{"A", 2, 3, "2026-06-25 03:00"}, // M54: Südafrika - Südkorea
		{"E", 2, 3, "2026-06-25 22:00"}, // M55: Curaçao - Elfenbeinküste
		{"E", 4, 1, "2026-06-25 22:00"}, // M56: Ecuador - Deutschland
		{"F", 2, 3, "2026-06-26 01:00"}, // M57: Japan - Schweden
		{"F", 4, 1, "2026-06-26 01:00"}, // M58: Tunesien - Niederlande
		{"D", 4, 1, "2026-06-26 04:00"}, // M59: Türkei - USA
		{"D", 2, 3, "2026-06-26 04:00"}, // M60: Paraguay - Australien
		{"I", 4, 1, "2026-06-26 21:00"}, // M61: Norwegen - Frankreich
		{"I", 2, 3, "2026-06-26 21:00"}, // M62: Senegal - Irak
		{"G", 2, 3, "2026-06-27 05:00"}, // M63: Ägypten - Iran
		{"G", 4, 1, "2026-06-27 05:00"}, // M64: Neuseeland - Belgien
		{"H", 2, 3, "2026-06-27 02:00"}, // M65: Kap Verde - Saudi-Arabien
		{"H", 4, 1, "2026-06-27 02:00"}, // M66: Uruguay - Spanien
		{"L", 4, 1, "2026-06-27 23:00"}, // M67: Panama - England
		{"L", 2, 3, "2026-06-27 23:00"}, // M68: Kroatien - Ghana
		{"J", 2, 3, "2026-06-28 04:00"}, // M69: Algerien - Österreich
		{"J", 4, 1, "2026-06-28 04:00"}, // M70: Jordanien - Argentinien
		{"K", 4, 1, "2026-06-28 01:30"}, // M71: Kolumbien - Portugal
		{"K", 2, 3, "2026-06-28 01:30"}, // M72: DR Kongo - Usbekistan
	}

	totalSeeded := 0
	for _, m := range groupMatches {
		teams := groups[m.g]
		home := teams[m.hp-1]
		away := teams[m.ap-1]
		t, err := time.ParseInLocation("2006-01-02 15:04", m.dmez, loc)
		if err != nil {
			log.Fatalf("Bad date %s: %v", m.dmez, err)
		}
		stmt.Exec("Gruppenphase", m.g, home, away, t.UTC())
		totalSeeded++
	}

	// Knockout stages - exact FIFA schedule from Excel
	type ko struct {
		phase string
		home  string
		away  string
		dmez  string
	}

	knockoutMatches := []ko{
		// Runde der 32 (Sechzehntelfinale) — Matches 73-88
		{"Runde der 32", "2. Gruppe A", "2. Gruppe B", "2026-06-28 21:00"},
		{"Runde der 32", "1. Gruppe E", "Bester 3. ABCDF", "2026-06-29 22:30"},
		{"Runde der 32", "1. Gruppe F", "2. Gruppe C", "2026-06-30 03:00"},
		{"Runde der 32", "1. Gruppe C", "2. Gruppe F", "2026-06-29 19:00"},
		{"Runde der 32", "1. Gruppe I", "Bester 3. CDFGH", "2026-06-30 23:00"},
		{"Runde der 32", "2. Gruppe E", "2. Gruppe I", "2026-06-30 19:00"},
		{"Runde der 32", "1. Gruppe A", "Bester 3. CEFHI", "2026-07-01 03:00"},
		{"Runde der 32", "1. Gruppe L", "Bester 3. EHIJK", "2026-07-01 18:00"},
		{"Runde der 32", "1. Gruppe D", "Bester 3. BEFIJ", "2026-07-02 02:00"},
		{"Runde der 32", "1. Gruppe G", "Bester 3. AEHIJ", "2026-07-01 22:00"},
		{"Runde der 32", "2. Gruppe K", "2. Gruppe L", "2026-07-03 01:00"},
		{"Runde der 32", "1. Gruppe H", "2. Gruppe J", "2026-07-02 21:00"},
		{"Runde der 32", "1. Gruppe B", "Bester 3. EFGIJ", "2026-07-03 05:00"},
		{"Runde der 32", "1. Gruppe J", "2. Gruppe H", "2026-07-04 00:00"},
		{"Runde der 32", "1. Gruppe K", "Bester 3. DEIJL", "2026-07-04 03:30"},
		{"Runde der 32", "2. Gruppe D", "2. Gruppe G", "2026-07-03 20:00"},
		// Achtelfinale — Matches 89-96
		{"Achtelfinale", "Sieger M74", "Sieger M77", "2026-07-04 23:00"},
		{"Achtelfinale", "Sieger M73", "Sieger M75", "2026-07-04 19:00"},
		{"Achtelfinale", "Sieger M76", "Sieger M78", "2026-07-05 22:00"},
		{"Achtelfinale", "Sieger M79", "Sieger M80", "2026-07-06 02:00"},
		{"Achtelfinale", "Sieger M83", "Sieger M84", "2026-07-06 21:00"},
		{"Achtelfinale", "Sieger M81", "Sieger M82", "2026-07-07 02:00"},
		{"Achtelfinale", "Sieger M86", "Sieger M88", "2026-07-07 18:00"},
		{"Achtelfinale", "Sieger M85", "Sieger M87", "2026-07-07 22:00"},
		// Viertelfinale — Matches 97-100
		{"Viertelfinale", "Sieger M89", "Sieger M90", "2026-07-09 22:00"},
		{"Viertelfinale", "Sieger M93", "Sieger M94", "2026-07-10 21:00"},
		{"Viertelfinale", "Sieger M91", "Sieger M92", "2026-07-11 23:00"},
		{"Viertelfinale", "Sieger M95", "Sieger M96", "2026-07-12 03:00"},
		// Halbfinale — Matches 101-102
		{"Halbfinale", "Sieger M97", "Sieger M98", "2026-07-14 21:00"},
		{"Halbfinale", "Sieger M99", "Sieger M100", "2026-07-15 21:00"},
		// Drittplatz — Match 103
		{"Drittplatz", "Verlierer HF1", "Verlierer HF2", "2026-07-18 23:00"},
		// Finale — Match 104
		{"Finale", "Sieger HF1", "Sieger HF2", "2026-07-19 21:00"},
	}

	for _, km := range knockoutMatches {
		t, err := time.ParseInLocation("2006-01-02 15:04", km.dmez, loc)
		if err != nil {
			log.Fatalf("Bad date %s: %v", km.dmez, err)
		}
		stmt.Exec(km.phase, "", km.home, km.away, t.UTC())
		totalSeeded++
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
	log.Printf("Seeded %d matches", totalSeeded)
}