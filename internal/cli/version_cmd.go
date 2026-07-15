package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"github.com/danieljustus/symaira-relate/internal/version"
)

func init() {
	Register(&Command{
		Name:  "version",
		Short: "Print version information",
		Run:   runVersion,
	})
}

func runVersion(ctx context.Context, iostreams IO, args []string) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(iostreams.Stderr)
	jsonOut := fs.Bool("json", false, "emit machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	info := version.Get()
	if *jsonOut {
		return json.NewEncoder(iostreams.Stdout).Encode(info)
	}
	fmt.Fprintf(iostreams.Stdout, "%s %s (schema %d)\n", info.Tool, info.Version, info.SchemaVersion)
	return nil
}
