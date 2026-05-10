# WM 2026 Tippspiel

Fussball-WM 2026 Tippspiel für bis zu 20 Spieler.

## Features

- E-Mail Registrierung mit Verifizierung
- Tippen auf alle WM-Spiele
- Live-Tabelle mit Punkteberechnung
- Kommentarfunktion pro Spiel
- Admin-Panel zur Verwaltung

## Punkte

- 3 Punkte: Exaktes Ergebnis getippt
- 1 Punkt: Richtige Tordifferenz
- 0 Punkte: Sonst

## Deployment

```bash
docker compose up -d --build
```

## Admin

Standard-Admin: `admin@example.com` / `admin` (Passwort via ADMIN_PASSWORD env konfigurierbar)

Passwort nach erstem Login ändern!

## Konfiguration

Kopiere `.env.example` nach `.env` und trage die produktiven Werte lokal ein. `.env` wird nicht versioniert, damit keine Zugangsdaten oder persönlichen URLs/E-Mail-Adressen im Repository landen.
