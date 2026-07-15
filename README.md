# Symaira Relate

`symrelate` is a standalone, local-first contact and relationship manager.
It stores people, organizations, relationships, interaction history and
follow-ups in a single SQLite database on your machine — no account, no
cloud service, no network calls.

## Status

Beta (`v0.1.0-beta.1`). The CLI surface and on-disk schema are pre-1.0 and
may still change between beta releases.

## Install

```sh
go install github.com/danieljustus/symaira-relate/cmd/symrelate@latest
```

Requires Go 1.22+. `symrelate` builds with `CGO_ENABLED=0` and has no
compile-time dependency on any other Symaira tool — it runs standalone with
no other Symaira binary installed. See [ARCHITECTURE.md](ARCHITECTURE.md)
for the standalone-first and data-ownership boundaries this implies.

## Quick start

```sh
symrelate doctor          # verify paths and database health
symrelate version --json  # {"tool":"symrelate","version":"...","schema_version":N,"api_version":"v1"}
```

Every data command prints stable, versioned JSON by default; add
`--human` for a short readable summary instead (e.g. `symrelate contact
show --human <id>`). See [docs/CLI_CONTRACT.md](docs/CLI_CONTRACT.md) for
the full compatibility contract.

Agents can talk to `symrelate` directly via `symrelate mcp`, a narrow MCP
server over stdio — see [docs/MCP.md](docs/MCP.md).

For everyday contact management without the CLI, run `symrelate console`
for a localhost-only, authenticated web UI — see
[docs/CONSOLE.md](docs/CONSOLE.md).

## Data location

Paths follow the XDG Base Directory convention:

| Purpose | Default | Override |
|---|---|---|
| Config | `$XDG_CONFIG_HOME/symrelate` (`~/.config/symrelate`) | `SYMRELATE_CONFIG_HOME` |
| Data (the SQLite database) | `$XDG_DATA_HOME/symrelate` (`~/.local/share/symrelate`) | `SYMRELATE_DATA_HOME` |
| Cache | `$XDG_CACHE_HOME/symrelate` (`~/.cache/symrelate`) | `SYMRELATE_CACHE_HOME` |

## Importing existing contacts

```sh
symrelate import vcard --dry-run testdata/import/sample.vcf   # preview, writes nothing
symrelate import vcard testdata/import/sample.vcf             # apply
symrelate import csv --map name=Full Name,email=Email sample.csv --dry-run
symrelate import runs                                         # history of past imports
```

vCard (3.0/4.0) and CSV import share one workflow: a dry run never writes,
duplicate candidates (matched on normalized contact points, names, or a
prior import of the same source) require an explicit `--resolve
ROW=create|skip|merge:PERSONID`, and re-importing an unchanged source is
idempotent — nothing is duplicated on a second run.

## Data protection

Sensitive contact-point values never appear in logs, errors or `doctor`
output; backups are AES-256-GCM encrypted; erasing a contact is a hard
delete with an audit trail. See [docs/PRIVACY.md](docs/PRIVACY.md) for the
full policy.

## Development

```sh
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go vet ./...
CGO_ENABLED=0 go test ./...
```
