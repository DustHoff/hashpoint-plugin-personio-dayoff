# hashpoint-plugin-personio-dayoff

Hashpoint-Plugin, das Dayoff-/Abwesenheitstage aus **Personio** ueber die
**Frontend-API** (nicht die offizielle Public-API) abruft und sie als
`off_hours_provider` an Hashpoint liefert.

## Aufbau

- Implementiert das Hashpoint-Plugin-SDK
  (`github.com/dusthoff/hashpoint/plugin/sdk`).
- Capability: `off_hours_provider`.
- Session/Cookies/CSRF werden vom Host via `HostAPI.RequestPersonioSession`
  bereitgestellt - das Plugin verwaltet **keine** Credentials.

## Build

Native Go, **kein CGO**. Windows-Build als GUI-Subsystem (keine Konsole):

```powershell
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -trimpath -ldflags="-s -w -H=windowsgui" -o personio-dayoff.exe .\
```

Lokal testen (`go test ./...`, `go vet ./...`, `golangci-lint run`).

## Installation

Nach dem Build die folgenden Dateien nach
`%APPDATA%\TimeTracker\plugins\personio-dayoff\` kopieren:

- `personio-dayoff.exe`
- `manifest.toml`

## Entwicklung

- Branches: `feature/...`, `fix/...`, `chore/...` - **kein** direktes Push auf `main`.
- Commits: [Conventional Commits](https://www.conventionalcommits.org/) -
  steuern die automatische SemVer-Bumpung der Release-Pipeline.
- Vor jedem neuen Feature: SDK-Stand pruefen
  (<https://github.com/DustHoff/hashpoint/tree/main/plugin/sdk>).

## Release

GitHub Actions erzeugt auf jedem Push auf `main` automatisch ein Release:

1. `mathieudutour/github-tag-action` bestimmt den naechsten SemVer-Tag aus den
   Conventional-Commits-Messages.
2. GoReleaser baut das Windows-Binary (GUI-Subsystem, statisch, ohne CGO) und
   haengt das Archiv inkl. `manifest.toml` an den GitHub-Release.

Wenn seit dem letzten Release keine release-relevanten Commits (`feat:`,
`fix:`, breaking) vorhanden sind, wird **kein** neues Release erzeugt.
