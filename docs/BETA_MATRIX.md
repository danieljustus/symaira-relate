# Beta manual QA matrix

A manual verification checklist for `v0.1.0-beta.1`, covering the
surfaces automated tests don't fully exercise end-to-end (a real
terminal, a real browser, a machine without SymMemory/SymMeet
installed). Run through this before tagging a release; record the
result (date, build, pass/fail, notes) for each row rather than just
checking a box, so a failure has enough context to act on without
re-running the whole matrix.

## Install and quick start

| # | Check | How |
|---|---|---|
| 1 | Clean-machine build | `go install github.com/danieljustus/symaira-relate/cmd/symrelate@latest` on a machine with no prior `symrelate` state |
| 2 | `doctor` succeeds on a fresh profile | `symrelate doctor` — expect `ok` on stdout, no error |
| 3 | `version --json` reports all three versions | Confirm `tool`, `version`, `schema_version`, `api_version` are all present and non-empty |

## Core contact / CLI

| # | Check | How |
|---|---|---|
| 4 | Create, show, update, list, delete a person | `contact add/show/update/list/delete` |
| 5 | Create, show, update, list, delete an organization | `org add/show/update/list/delete` |
| 6 | Membership linking survives a role change | `membership add`, then re-`add` with a different role; both appear in `list-person` |
| 7 | Relationship + interaction + follow-up timeline merges correctly | `relate add`, `interaction add`, `followup add`, then `timeline` shows both, most-recent-first |
| 8 | `--human` output is readable, `--json` (default) stays parseable | Same command with and without `--human`; pipe the default output through `jq` |

## Import / dedupe

| # | Check | How |
|---|---|---|
| 9 | vCard dry-run writes nothing | `import vcard --dry-run testdata/import/sample.vcf`, then `contact list` shows no new rows |
| 10 | vCard apply creates the expected count | `import vcard testdata/import/sample.vcf`, confirm `Created` matches the fixture's contact count |
| 11 | Re-import of the same file is idempotent | Re-run the same `import vcard` command; `Created` is 0 the second time |
| 12 | A duplicate candidate requires `--resolve` | Import a file with a contact matching an existing email; confirm the row is excluded until resolved |
| 13 | CSV auto-detected column mapping | `import csv` against a CSV with recognizable headers (`name`, `email`, ...), no `--map` |

## Privacy / recovery

| # | Check | How |
|---|---|---|
| 14 | Backup round-trips | `backup create --out b.enc --passphrase x`, then `backup restore --in b.enc --target /tmp/restored.db --passphrase x`, confirm the restored DB opens |
| 15 | Wrong passphrase is rejected before any write | `backup restore` with an incorrect passphrase; confirm no file is created at `--target` |
| 16 | Erase removes cascaded data and leaves an audit trail | `contact erase <id>`, then `audit list` shows one `erase` event with counts, no contact data |
| 17 | No contact-point value appears in any error text | Trigger a duplicate contact-point error and a not-found error; grep the output for the value you used |

## MCP

| # | Check | How |
|---|---|---|
| 18 | `symrelate mcp` stdout is protocol-clean | Pipe one JSON-RPC request in, confirm every stdout line parses as JSON, diagnostics are on stderr |
| 19 | An MCP client (Claude Desktop, or any MCP-compatible client) can list and call tools | Configure `symrelate mcp` as a stdio server, call `contact_search` and `contact_create` |

## Console

| # | Check | How |
|---|---|---|
| 20 | Console binds to loopback only | `symrelate console --port 4287`, confirm `lsof -i :4287` shows `127.0.0.1`, not `*` or `0.0.0.0` |
| 21 | Token bootstrap works, token disappears from the URL | Open the printed URL, confirm the browser's address bar no longer shows `?token=` after the first load |
| 22 | An unauthenticated request is rejected | `curl http://127.0.0.1:4287/api/v1/contacts` (no token) returns 401 |
| 23 | Create → detail → erase flow works in a real browser | Add a contact through the UI, open its detail view, erase it, confirm it's gone from the list |
| 24 | Import review flow works in a real browser | Preview and apply `testdata/import/sample.vcf` through the Import tab |
| 25 | Keyboard-only navigation reaches every control | Tab through the whole UI with a mouse disconnected; every button/input/select is reachable and operable |

## Absent optional tools (standalone-first)

| # | Check | How |
|---|---|---|
| 26 | Full build/test with no other Symaira tool present | On a machine (or container) with no `symmemory`/`symmeet`/`symvault` binary anywhere on `PATH`, run `go build ./...` and `go test ./...` — both must succeed |
| 27 | `memory status` / `meeting status` degrade gracefully | On that same machine, both report `"available": false` with a reason, and no other command fails because of it |
| 28 | Backup falls back to env-var passphrase without SymVault | `SYMRELATE_BACKUP_PASSPHRASE=x symrelate backup create --out b.enc` succeeds with no `--passphrase` flag and no SymVault installed |
| 29 | `meeting import --fixture` reads the real published SymMeet manifest | `symrelate meeting import --fixture internal/cli/testdata/symmeet/meeting-complete --person <id>` creates the interaction, and the source/consent/retention status line appears on Stderr |

## Sign-off

Once every row has a recorded result for the target build/commit, the
result set — not just "all green" — belongs in the release notes'
"verified against" section, alongside any rows that were skipped and why
(e.g. Windows console accessibility not covered by this pass).
