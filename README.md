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
symrelate version --json  # {"tool":"symrelate","version":"...","schema_version":N}
```

## Data location

Paths follow the XDG Base Directory convention:

| Purpose | Default | Override |
|---|---|---|
| Config | `$XDG_CONFIG_HOME/symrelate` (`~/.config/symrelate`) | `SYMRELATE_CONFIG_HOME` |
| Data (the SQLite database) | `$XDG_DATA_HOME/symrelate` (`~/.local/share/symrelate`) | `SYMRELATE_DATA_HOME` |
| Cache | `$XDG_CACHE_HOME/symrelate` (`~/.cache/symrelate`) | `SYMRELATE_CACHE_HOME` |

## Development

```sh
go build ./...
go vet ./...
go test ./... -race
```
