# SymMeet integration

`symrelate` can turn a reviewed SymMeet meeting into a contact interaction
— an opaque meeting reference on a person or organization's timeline,
never a transcript, recording, or automatically-matched participant list.
This document is the status report for that integration.

See [ARCHITECTURE.md](../../ARCHITECTURE.md)'s standalone-first rule: no
package in this repository imports a symaira-meet package. Discovery
talks to a `symmeet` binary on `PATH` via subprocess calls, the same
pattern [SYMMEMORY.md](SYMMEMORY.md) documents for SymMemory.

## What's implemented

- **Runtime discovery** (`internal/integration/symmeet.Discover`): a
  bounded `symmeet capabilities --json` handshake. Missing binary,
  timeout, non-zero exit, malformed JSON, or a `tool` field that isn't
  `"symmeet"` all degrade to `ErrUnavailable` — `symrelate meeting status`
  reports availability without ever failing the command, and every other
  `symrelate` command is entirely unaffected by SymMeet's presence or
  absence.
- **Idempotent meeting interactions**
  (`internal/service/relationship.ImportPersonMeeting` /
  `ImportOrganizationMeeting`, migration
  `0007_meeting_idempotency.sql`): reuses the existing `meeting`
  `InteractionKind` and `ExternalRef` field already in
  `internal/domain/relationship` — no new domain concept was needed here,
  only the idempotency guarantee. Re-importing the same meeting id for
  the same person/organization returns the existing interaction (verified
  by `TestImportPersonMeeting_ReImport_IsIdempotent`) instead of creating
  a duplicate, enforced at the database level with a unique partial index
  (not just a service-layer check) so a concurrent double-import can't
  race past it either.
- **CLI surface**: `symrelate meeting status` and `symrelate meeting
  import --id <meeting-id> [--title ...] [--summary ...] [--occurred
  RFC3339] (--person <id>[,...] | --org <id>[,...]) [--fixture <path>]`.
  Contact association is always an explicit `--person`/`--org` list —
  nothing here guesses who attended.

## What is provisional, and why

Unlike SymMemory (verified against a real local installation — see
[SYMMEMORY.md](SYMMEMORY.md)), **no `symmeet` binary was available while
building this integration**, and symaira-meet's own v1 ecosystem
integration contract
([symaira-meet#19](https://github.com/danieljustus/symaira-meet/issues/19))
is itself still open, depending on four other open issues in that repo
(#2 artifact contracts, #4 version handshake, #11 exports, #18 MCP). So
two things here are best-effort assumptions rather than confirmed
behavior:

- The `symmeet capabilities --json` command name and its `tool`/
  `version`/`schema_version` fields, taken directly from #19's own
  evidence section. `Capabilities` intentionally leaves every other
  documented field (engine capabilities, supported formats, artifact
  contract versions) unparsed rather than guessing their shape.
- **There is no real SymMeet artifact import.** The acceptance criterion
  "Relate can import a published synthetic SymMeet fixture only after
  user confirmation" needs that fixture format, which doesn't exist yet.
  `meeting import --fixture <path>` reads a local JSON file with an
  assumed `{meeting_id, title, summary, occurred_at}` shape as a stopgap
  — explicitly documented as provisional in the CLI's own `-h` output —
  so the command is usable and testable today, with every field
  overridable by an explicit flag. Once #19 publishes the real fixture
  format, `applyMeetingFixture` in `internal/cli/meeting_cmd.go` is the
  only place that needs to change.

## What is still blocked upstream

- Real fixture import (needs symaira-meet#19 and its four dependencies).
- Any richer SymMeet capability data (recording/transcript status,
  consent/retention display) — the parent issue's "source/consent/
  retention status display" criterion needs a published capabilities
  schema to read those fields from; today `Capabilities` only reads
  enough to answer "is SymMeet here and compatible."

The interaction/idempotency layer above is real, tested, and does not
need to change once #19 lands — only the fixture-parsing and capability-
reading code does.
