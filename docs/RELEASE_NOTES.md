# Release Notes - v0.2.0-beta.1

`symrelate` is a local-first, standalone contact and relationship manager. It serves as the authoritative repository for raw contact points (names, emails, phones, addresses) and the relationships built on top of them, operating entirely offline without any external service dependency.

This release replaces the hand-written JSON-RPC transport with the shared `corekit/mcpserver` stack and adds the versionkit handshake payload, aligning symrelate with the rest of the Symaira ecosystem.

## Resolved Issues

This release resolves the following issues:
- [#62](https://github.com/danieljustus/symaira-relate/issues/62): Replace the hand-written JSON-RPC stack with corekit/mcpserver and wire up the base packages
- [#61](https://github.com/danieljustus/symaira-relate/issues/61): Add `version --check` update detection (shipped in v0.1.2-beta.1)

## Major Features

- **MCP transport via corekit/mcpserver**: The own JSON-RPC 2.0 stack in `internal/mcp/` (framing, validation, error codes, initialize handshake, zero-stdio-pollution rule) is replaced by the shared `corekit/mcpserver` transport — the same stack the sibling Symaira tools use. The 8-tool catalog and its PII-redaction boundary are unchanged. Requires corekit v0.8.0.
- **Versionkit handshake**: `symrelate version --json` now emits the ecosystem-standard `{tool, version, schema_version}` payload (plus the existing `api_version` contract field), so GUI clients using SymairaToolKit can recognize this tool.
- **Update detection (v0.1.2-beta.1)**: `symrelate version --check` checks GitHub for newer releases via corekit/updatecheck.

## Verified Environments

- **OS**: macOS, Linux, Windows
- **Go Version**: 1.22+
- **Database Schema Version**: 7 (automatically migrated on start)

## Intentional Exclusions & Known Limitations

- **SymMemory Context Lookup**: Sibling entity creation/relation linking requires upstream `symmemory` changes. Context resolution is currently ID-only, and name searches fallback to best-effort text match.
- **Stable APIs**: All CLI output JSON contracts, database schemes, and MCP schemas are pre-1.0 and subject to modification.
