# SymMemory integration

`symrelate` can optionally link a person or organization to a SymMemory
entity so a user can pull up related context without SymMemory ever
becoming a requirement, and without any contact-point data ever being
sent to it. This document is the status report for that integration —
what is implemented, what was verified against a real installation, and
what is still blocked on upstream work.

See [ARCHITECTURE.md](../../ARCHITECTURE.md)'s standalone-first rule: no
package in this repository imports a SymMemory package. Everything below
talks to a `symmemory` binary on `PATH` via subprocess calls, exactly like
`internal/domain/security.SymVaultKeyProvider` does for SymVault.

## What's implemented

- **Runtime discovery** (`internal/integration/symmemory.Discover`): a
  bounded (3s default, configurable), `PATH`-based handshake via
  `symmemory version --json`. Missing binary, timeout, non-zero exit,
  malformed JSON, or a `tool` field that isn't `"symmemory"` all degrade
  to `ErrUnavailable` — `symrelate` never fails to build, run, or pass its
  tests without SymMemory installed.
- **Reviewed external links** (`internal/domain/memorylink`,
  `internal/service/memorylink`, migration `0006_memory_links.sql`): a
  Relate-owned `memory_links` table mapping a person/organization to a
  SymMemory entity id. Linking is always explicit — `symrelate memory
  link --person <id> --entity <symmemory-entity-id>` — the user must have
  already found the entity id themselves; nothing here searches or
  matches automatically. At most one link per contact; relinking requires
  an explicit unlink first, so an existing link is never silently
  overwritten.
- **Context lookup** (`internal/integration/symmemory.Context`): given an
  already-linked entity id, shells out to `symmemory entity show <id>
  --output json` and returns the entity's own JSON representation for
  display — `symrelate` does not parse, store, or interpret its fields.
- **CLI surface**: `symrelate memory status|link|unlink|show` — see each
  command's `-h` output.

## Verified against a real installation

While building this, a real `symmemory v0.12.0` was available locally,
which let two assumptions get checked against actual behavior instead of
staying guesses:

1. `symmemory version --json` returns exactly `{"tool":"symmemory",
   "version":"...","schema_version":N}` — confirmed correct.
2. `symmemory entity show <arg> --output json` prints the entity as one
   JSON object **followed by non-JSON "Linked memories" text on the same
   stdout stream**. `Context()` was originally written to validate the
   *entire* stdout as JSON, which would have rejected every real,
   successful response — a decode-one-value streaming parser (rather than
   whole-output validation) is required, and that is what's implemented
   and tested (`TestContext_TrailingNonJSONText_IsIgnored`).

`internal/integration/symmemory/discover_test.go` also has a light
integration check (`TestContext_AgainstRealBinary`) that only runs when a
real `symmemory` is on `PATH` (skipped in CI, per the standalone-first
rule — no other Symaira tool is present there) so a developer with
SymMemory installed gets an early signal if the real CLI's output shape
drifts from what this package assumes.

## Known limitation, confirmed against that real installation

`entity show` resolves its argument **by name, not by id**. `memory_links`
deliberately stores the entity's stable `id` (never a mutable name), so a
link survives the linked entity being renamed in SymMemory — but that
means `symrelate memory show` calls `Context()` with an id that the real
`entity show` command cannot resolve today, and gets the safe, expected
`ErrUnavailable`. This is not a bug in this package; it is the same gap
[symaira-memory#343](https://github.com/danieljustus/symaira-memory/issues/343)
exists to close (deterministic, ID-aware candidate resolution).

## What is still blocked upstream

Two acceptance criteria from the parent issue genuinely cannot be
implemented against today's SymMemory:

- **Reviewed candidate selection with match reasoning** (exact/ambiguous/
  no-match) needs `ResolveEntityCandidates`, which
  [symaira-memory#343](https://github.com/danieljustus/symaira-memory/issues/343)
  (open) adds. Until then, linking requires the user to already know the
  target entity id — there is no in-`symrelate` search/suggest step.
- **ID-based, provenance-aware relation writes** need the extended
  `entity_relate` API from
  [symaira-memory#344](https://github.com/danieljustus/symaira-memory/issues/344)
  (open). This package does not write relations into SymMemory at all —
  only reviewed links and read-only context lookups are implemented.

Once #343 and #344 land, the two follow-ups are: add a candidate-search
step ahead of `memory link` (replacing "the user already knows the id"
with a reviewed suggestion list), and add an optional relation-write
command using the new ID/provenance API. Both are additive to the current
design, not a rework of it.
