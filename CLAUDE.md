# CLAUDE.md

Dieses Dokument enthält die verbindlichen Projektregeln und den Kontext für die Arbeit an diesem Repository.

## Projektziel

Entwicklung eines Plugins für **Hashpoint**, das die Abwesenheits-/Dayoff-Tage aus **Personio** ermittelt.

- Es wird **nicht** die offizielle Personio-API verwendet.
- Stattdessen wird die **interne API des Personio User-Frontends** genutzt (Reverse-Engineering der Frontend-Endpunkte, inkl. Login-Flow, Session-Handling, CSRF/Tokens, etc.).
- Das Plugin integriert sich in das Hashpoint-Plugin-System über das offizielle SDK.

## Technische Rahmenbedingungen

- **Sprache:** Go (native).
- **Kein CGO.** `CGO_ENABLED=0` ist verbindlich für alle Builds. Es dürfen keine Abhängigkeiten eingeführt werden, die CGO benötigen.
- Reine Go-Module, statisch kompilierbar, cross-platform fähig.
- Go-Version: aktuelle stabile Release (siehe `go.mod`).

## SDK-Referenzen

- SDK-Quellcode: https://github.com/DustHoff/hashpoint/tree/main/plugin/sdk
- SDK-Dokumentation: https://github.com/DustHoff/hashpoint/blob/main/docs/plugins/README.md

### Pflicht vor jedem neuen Feature

Vor Implementierungsbeginn eines neuen Features **immer** prüfen:

1. Hat sich das SDK seit dem letzten Stand geändert? (Commits / Releases im Hashpoint-Repo prüfen.)
2. Gibt es relevante Änderungen an Interfaces, Contracts, Auth- oder Lifecycle-Hooks?
3. Falls ja: Plugin-Code an die neue SDK-Version anpassen, bevor das eigentliche Feature umgesetzt wird.

Diese Prüfung ist **nicht optional** und gehört explizit in jeden Feature-Workflow.

## Branching & Pull Requests

- Das Repository wird **öffentlich auf GitHub** geführt.
- **Jedes Feature** wird in einem **eigenen Feature-Branch** entwickelt.
- Branch-Naming: `feature/<kurzbeschreibung>`, `fix/<kurzbeschreibung>`, `chore/<kurzbeschreibung>`.
- Für jedes Feature wird ein **eigener Pull Request** gegen `main` erstellt.
- Kein direktes Pushen auf `main`.
- Commits folgen idealerweise Conventional Commits (`feat:`, `fix:`, `chore:`, `docs:`, `refactor:` …), damit automatisches Versioning sauber funktioniert.

## Release-Pipeline & Versionierung

- **CI/CD via GitHub Actions.**
- Auf **jedem Push auf `main`** wird automatisch ein **neues Release** erzeugt.
- Versionierung folgt **Semantic Versioning** (`MAJOR.MINOR.PATCH`).
  - `fix:` → PATCH
  - `feat:` → MINOR
  - Breaking Change (`!` oder `BREAKING CHANGE:` im Footer) → MAJOR
- Das Release enthält gebaute Artefakte (statische Binaries, mind. linux/amd64; weitere Plattformen nach Bedarf) und einen automatisch generierten Changelog.

## Qualitätsregeln

- `go vet`, `go build`, `go test ./...` müssen vor jedem Push grün sein.
- Empfohlen: `golangci-lint` als Linter in der Pipeline.
- Keine Secrets, Tokens oder Personio-Credentials ins Repo. Konfiguration via Env-Variablen / Plugin-Config laut SDK.

## Arbeitsweise für Claude

- Vor neuen Features: SDK-Stand prüfen (siehe oben).
- Änderungen immer in einem Feature-Branch + PR vorbereiten, nicht direkt auf `main`.
- Keine CGO-Abhängigkeiten einführen.
- Keine Nutzung der offiziellen Personio-API als „Abkürzung“ – das widerspricht dem Projektziel.
