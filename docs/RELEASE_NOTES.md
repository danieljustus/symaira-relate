# Release Notes - v0.1.1-beta.1

`symrelate` is a local-first, standalone contact and relationship manager. It serves as the authoritative repository for raw contact points (names, emails, phones, addresses) and the relationships built on top of them, operating entirely offline without any external service dependency.

This release introduces a privacy-safe reference contract for external integrations (such as meeting notes) to link to contact identities without exposing raw PII.

## Resolved Issues

This release resolves the following issues:
- [#57](https://github.com/danieljustus/symaira-relate/issues/57): Publish a reference-only contact-reference JSON contract for external consumers

## Major Features

- **Reference-only Contact Contract**: Added `docs/integrations/CONTACT_REF.md` defining the consumer contract for `Ref` object (provider, schema_version, id, kind, display_name) which is additive-only.
- **CLI Contact Reference Lookup**: Added `symrelate contact ref <id> [--human]` command allowing external consumers to resolve reference details safely (using id/name columns only, never leaking email/phone/address/notes).

## Verified Environments

- **OS**: macOS, Linux, Windows
- **Go Version**: 1.22+
- **Database Schema Version**: 7 (automatically migrated on start)

## Intentional Exclusions & Known Limitations

- **SymMemory Context Lookup**: Sibling entity creation/relation linking requires upstream `symmemory` changes. Context resolution is currently ID-only, and name searches fallback to best-effort text match.
- **Stable APIs**: All CLI output JSON contracts, database schemes, and MCP schemas are pre-1.0 and subject to modification.
