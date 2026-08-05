# symaira-relate — Agent Instructions

Contact & relationship manager. `symrelate` = CLI + localhost web console + stdio MCP server over a single encrypted-at-rest-aware SQLite vault. Beta maturity (`v0.2.1-beta.1`): API/schema may change.

## Commands

No Makefile, no GoReleaser, no cobra — plain `go` only, `CGO_ENABLED=0` everywhere:

```bash
CGO_ENABLED=0 go build ./...                   # build
CGO_ENABLED=0 go test ./... -count=1           # test (caching disabled, per CI)
CGO_ENABLED=0 go vet ./...                     # lint
CGO_ENABLED=0 go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Entry point: `cmd/symrelate/main.go` (17 lines, no logic). Release workflow is hand-rolled (`.github/workflows/release.yml`): cross-platform build matrix + SHA256SUMS + `gh release create` on `v*` tags. Version injected via `-ldflags -X .../internal/version.Version`.

## Structure

```
internal/
  app/         # service container (app.go + wire.go) — THE single boundary; CLI/MCP/console all go through app.App
  cli/         # 18 command files; custom dispatcher, commands self-register via init() → cli.Register()
  mcp/         # narrow 8-tool catalog over the corekit/mcpserver stdio transport (protocol 2024-11-05)
  console/     # localhost HTTP console, token auth; static/ = vanilla HTML/CSS/JS SPA via //go:embed (no npm, no build step)
  domain/      # pure types: contact, relationship, security, importer, page, memorylink — zero I/O
  service/     # contact / relationship / security / importer / memorylink services
  storage/sqlite/  # modernc.org/sqlite, WAL, embedded migrations
  integration/ # symmemory + symmeet runtime adapters (discovery only, graceful fallback)
  errs/        # structured error vocabulary (KindNotFound/KindConflict/KindInvalid/...)
  xdg/         # SYMRELATE_*_HOME overrides, dirs 0700
```

## Conventions (repo-specific)

- **Storage**: `modernc.org/sqlite` only. WAL + `synchronous=NORMAL` + `foreign_keys=ON` + `SetMaxOpenConns(1)` (single writer, avoids SQLITE_BUSY) + `busy_timeout=5000`.
- **Migrations**: `internal/storage/sqlite/migrations/%04d_name.sql`, embedded, ascending, **pure-additive — never edit a released migration**. Current schema version: 7 (`internal/version`).
- **MCP scope is deliberately narrow**: 8 tools (`contact_search/get/create/update`, `organization_search/get`, `followup_list`, `timeline_get`). Delete/erase, import, backup/restore, export stay **CLI-only** — do not widen MCP surface without a privacy review.
- **Unknown JSON fields rejected** (`DisallowUnknownFields`); pagination clamped server-side (`MaxLimit=200`, default 50).
- **Tests**: in-memory SQLite via `OpenMemory()` with migrations applied; test files alongside source; CLI tested integration-style.
- **JSON contract**: versioned (`APIVersion=v1`), snake_case; changes follow `docs/CLI_CONTRACT.md`.

## PII Boundary (hard rules)

- `symrelate` owns raw contact points (emails, phones, addresses). Treat all output paths as PII-suspect.
- Every error message passes `security.Redact()` (masks email/phone patterns) before reaching clients — never bypass.
- Backups: AES-256-GCM + Argon2id. Erase: hard delete + `ON DELETE CASCADE` + audit trail (counts only, never values).
- Passphrase resolution chain: flag → env → SymVault → tty prompt (`domain/security/keyprovider.go`).
- `doctor` output must stay PII-free (paths + connectivity only).
- Nothing sends data to external services. Full policy: `docs/PRIVACY.md`.

## Anti-Patterns (this repo)

- Do NOT add cobra/CLI frameworks — custom dispatcher is intentional.
- Do NOT add GoReleaser — hand-rolled release workflow is intentional.
- Do NOT add a config file format — env vars + flags only (as of beta).
- Do NOT print diagnostics to stdout in MCP or CLI JSON mode (zero-pollution rule).
- Do NOT import sibling Symaira repos at compile time — runtime discovery via `exec.LookPath` with graceful fallback (`internal/integration/`). Sole exception: the MCP transport depends on the versioned `github.com/danieljustus/symaira-corekit` module (`mcpserver`), pinned in `go.mod` without a `replace` directive; everything else stays runtime-discovered.

## Docs

`README.md`, `docs/ARCHITECTURE.md`, `docs/MCP.md`, `docs/CONSOLE.md`, `docs/CLI_CONTRACT.md`, `docs/PRIVACY.md`, `docs/BETA_MATRIX.md` (29-check manual QA), `docs/integrations/{SYMMEMORY,SYMMEET}.md`.
