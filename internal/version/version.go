// Package version carries the build-time version metadata surfaced by
// `symrelate version --json` and the doctor command.
package version

// SchemaVersion is the highest embedded migration version this build knows
// about. It is independent of the tool release version.
const SchemaVersion = 5

// APIVersion is the declared version of the CLI `--json` output and MCP
// tool contract (see docs/CLI_CONTRACT.md). It is independent of both
// Version (the release) and SchemaVersion (the on-disk database): the JSON
// contract may stay stable across schema migrations, and changes to it are
// additive within a major version — a field is never removed or repurposed
// without bumping APIVersion.
const APIVersion = "v1"

// Version is overridden at build time via
// -ldflags "-X github.com/danieljustus/symaira-relate/internal/version.Version=v0.1.0".
var Version = "dev"

// Tool is the stable machine-readable tool identifier.
const Tool = "symrelate"

// Info is the JSON-serializable payload for `version --json`.
type Info struct {
	Tool          string `json:"tool"`
	Version       string `json:"version"`
	SchemaVersion int    `json:"schema_version"`
	APIVersion    string `json:"api_version"`
}

func Get() Info {
	return Info{Tool: Tool, Version: Version, SchemaVersion: SchemaVersion, APIVersion: APIVersion}
}
