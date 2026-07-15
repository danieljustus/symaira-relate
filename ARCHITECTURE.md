# Architecture

## Standalone-first

`symrelate` is a public, general-purpose contact manager. It must build,
run and pass its full test suite with no other Symaira repository present
on the machine or importable at compile time.

- No package in this repository imports a package from `symmemory` or any
  other Symaira repository.
- Optional integration with a sibling tool (for example resolving a backup
  key through SymVault, see the privacy design below) is always detected
  at **runtime** — by checking for a binary on `PATH` or a socket file —
  and degrades to a documented local fallback when the sibling is absent.
  A missing sibling tool is never a startup failure.

## Data ownership

`symrelate` is the authoritative store for direct contact identifiers:
names, email addresses, phone numbers, postal addresses, and the
relationships and interaction history built on top of them. This is a
deliberately different privacy boundary from Symaira's semantic memory
(`symmemory`), which stores redacted facts and explicitly does not persist
raw contact points. Nothing in this repository sends contact data to
another Symaira tool, a memory store, or any external service — see
[`docs/PRIVACY.md`](docs/PRIVACY.md) for the full data-handling policy
introduced alongside the protection and export features.

## Package layout

```
cmd/symrelate/         CLI entry point (main package only — no logic)
internal/cli/          command dispatch; one file per subcommand,
                        self-registering via init()
internal/mcp/           the MCP server (stdio JSON-RPC 2.0) — see docs/MCP.md
internal/console/       the local web console (localhost HTTP) — see docs/CONSOLE.md
internal/app/          the service boundary: wires storage + domain
                        services for CLI, MCP and console callers
internal/domain/        pure domain types and business rules, no SQL
internal/service/       use-case orchestration on top of domain + storage
internal/integration/   optional runtime-detected sibling-tool adapters
                        (SymMemory, SymMeet)
internal/storage/sqlite/ the only package allowed to open the database;
                        owns the embedded migrations
internal/errs/          structured error vocabulary shared by every layer
internal/xdg/           XDG Base Directory path resolution
internal/version/       build-time version metadata
```

`internal/app` is the single boundary the CLI, MCP server and web console
are allowed to depend on for behavior. None of the three front ends import
`internal/storage` or a service package directly; they call through
`app.App`, so there is exactly one implementation of every business rule
across all three surfaces.

## Persistence

- SQLite via `modernc.org/sqlite`, a pure-Go (CGO-free) driver — required
  so `symrelate` cross-compiles and ships as a single static binary.
  `go build`/`go test` in this repository always run with `CGO_ENABLED=0`
  in CI.
- WAL journal mode, `synchronous=NORMAL`, `foreign_keys=ON`, a single
  writer connection (`SetMaxOpenConns(1)`) — SQLite has no real connection
  pool, and a single writer avoids `SQLITE_BUSY` churn under WAL.
  `busy_timeout` is set for the rare cross-process contention case (a
  concurrent `doctor` run, for example).
- Migrations are embedded SQL files (`internal/storage/sqlite/migrations`,
  `%04d_name.sql`), tracked in a `schema_migrations` table, and applied
  in ascending version order inside one transaction per file on every
  startup. Migrations are additive only — once released, a migration file
  is never edited, only superseded by a new one.

## Diagnostics

Diagnostic and progress output goes to **stderr**. **Stdout** is reserved
for the actual result of a command (`version --json`'s JSON payload,
`doctor`'s final `ok`/error line) so it stays parseable by scripts and
safe to use as the `symrelate mcp` JSON-RPC transport (see
[`docs/MCP.md`](docs/MCP.md)) without stdout being polluted by
human-readable log lines.
