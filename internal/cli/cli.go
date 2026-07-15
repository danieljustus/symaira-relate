// Package cli implements the symrelate command dispatcher. Each subcommand
// lives in its own file and registers itself via an init() call to
// Register, so adding a command never requires touching this file.
package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
)

// IO bundles the streams a command reads and writes. Diagnostics belong on
// Stderr so Stdout can stay protocol-clean for future machine consumers
// (JSON output flags, and later an MCP server).
type IO struct {
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
}

// Command is one top-level subcommand.
type Command struct {
	Name  string
	Short string
	Run   func(ctx context.Context, io IO, args []string) error
}

var registry = map[string]*Command{}
var order []string

// Register adds cmd to the root dispatcher. Panics on duplicate names,
// which can only happen from a programming error at init time.
func Register(cmd *Command) {
	if _, exists := registry[cmd.Name]; exists {
		panic("cli: command already registered: " + cmd.Name)
	}
	registry[cmd.Name] = cmd
	order = append(order, cmd.Name)
}

// Run dispatches args[0] to its Command and returns a process exit code.
func Run(ctx context.Context, iostreams IO, args []string) int {
	if len(args) == 0 {
		printUsage(iostreams.Stderr)
		return 2
	}

	name := args[0]
	if name == "-h" || name == "--help" || name == "help" {
		printUsage(iostreams.Stdout)
		return 0
	}

	cmd, ok := registry[name]
	if !ok {
		fmt.Fprintf(iostreams.Stderr, "symrelate: unknown command %q\n", name)
		printUsage(iostreams.Stderr)
		return 2
	}

	if err := cmd.Run(ctx, iostreams, args[1:]); err != nil {
		fmt.Fprintf(iostreams.Stderr, "symrelate: %v\n", err)
		return 1
	}
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: symrelate <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	names := append([]string(nil), order...)
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintf(w, "  %-12s %s\n", n, registry[n].Short)
	}
}
