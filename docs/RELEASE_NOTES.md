# Release Notes - v0.1.0-beta.1

`symrelate` is a local-first, standalone contact and relationship manager. It serves as the authoritative repository for raw contact points (names, emails, phones, addresses) and the relationships built on top of them, operating entirely offline without any external service dependency.

This release represents the completion of the core beta milestone.

## Resolved Issues

This release resolves the following issues:
- [#1](https://github.com/danieljustus/symaira-relate/issues/1): Establish the standalone local-first core and persistence foundation
- [#2](https://github.com/danieljustus/symaira-relate/issues/2): Implement the beta contact, organization, and contact-point model
- [#3](https://github.com/danieljustus/symaira-relate/issues/3): Add relationships, interaction history, and follow-up workflows
- [#4](https://github.com/danieljustus/symaira-relate/issues/4): Protect contact data and provide auditable deletion, backup, and export
- [#5](https://github.com/danieljustus/symaira-relate/issues/5): Import vCard and CSV through a reviewed deduplication workflow
- [#6](https://github.com/danieljustus/symaira-relate/issues/6): Define the stable CLI JSON and MCP contact-management contract
- [#7](https://github.com/danieljustus/symaira-relate/issues/7): Ship a local web console for everyday contact management
- [#8](https://github.com/danieljustus/symaira-relate/issues/8): Add an optional SymMemory context adapter with privacy-safe external links
- [#9](https://github.com/danieljustus/symaira-relate/issues/9): Import SymMeet artifacts as privacy-safe contact interactions
- [#10](https://github.com/danieljustus/symaira-relate/issues/10): Complete the beta quality gate, documentation, and release artifacts

## Major Features

- **Local-first SQLite Storage**: Zero network dependencies, pure Go CGO-free driver for fully static cross-platform binaries.
- **Privacy & Security**: Encrypted database backups (AES-256-GCM + Argon2id), audited hard-erasure (`ON DELETE CASCADE`), data-minimized exports (minimizing notes/metadata leak), and automatic terminal/log output redaction of contact-point values.
- **Reviewed Imports**: Interactive vCard and CSV deduplication workflow with dry-run support and idempotency checks.
- **Flexible Interfaces**: Subcommand CLI with stable JSON formats, stdio MCP server for agentic interaction, and local web console UI with token bootstrap.
- **Eco integrations**: Runtime-detected optional links to SymMemory and SymMeet meeting logs.

## Verified Environments

- **OS**: macOS, Linux, Windows
- **Go Version**: 1.22+
- **Database Schema Version**: 1 (automatically migrated on start)

## Intentional Exclusions & Known Limitations

- **SymMemory Context Lookup**: Sibling entity creation/relation linking requires upstream `symmemory` changes. Context resolution is currently ID-only, and name searches fallback to best-effort text match.
- **SymMeet Handshake**: Sibling binary discovery cannot be verified end-to-end due to the lack of a real `symmeet` executable on the testing path. Manifest file import works as specified.
- **Stable APIs**: All CLI output JSON contracts, database schemes, and MCP schemas are pre-1.0 and subject to modification.
