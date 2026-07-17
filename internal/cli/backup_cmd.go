package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/danieljustus/symaira-relate/internal/app"
	"github.com/danieljustus/symaira-relate/internal/domain/security"
)

func init() {
	Register(&Command{
		Name:  "backup",
		Short: "Create or restore an encrypted database backup",
		Run:   runBackup,
	})
}

func runBackup(ctx context.Context, iostreams IO, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: symrelate backup <create|restore> ...")
	}
	verb, rest := args[0], args[1:]

	switch verb {
	case "create":
		return backupCreate(ctx, iostreams, rest)
	case "restore":
		return backupRestore(ctx, iostreams, rest)
	default:
		return fmt.Errorf("symrelate backup: unknown subcommand %q", verb)
	}
}

// resolvePassphrase tries an explicit --passphrase flag, then the
// environment variable, then SymVault if installed — see
// security.DefaultKeyProviders and docs/PRIVACY.md for the documented
// standalone fallback.
func resolvePassphrase(ctx context.Context, explicit string) ([]byte, error) {
	key, _, err := security.DefaultKeyProviders([]byte(explicit)).Resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("no passphrase available: pass --passphrase, set %s, or install SymVault", security.DefaultPassphraseEnvVar)
	}
	return key, nil
}

func backupCreate(ctx context.Context, iostreams IO, args []string) error {
	fs := flag.NewFlagSet("backup create", flag.ContinueOnError)
	fs.SetOutput(iostreams.Stderr)
	out := fs.String("out", "", "output backup file path (required)")
	passphrase := fs.String("passphrase", "", "encryption passphrase (falls back to env/SymVault)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return fmt.Errorf("usage: symrelate backup create --out <file> [--passphrase ...]")
	}

	key, err := resolvePassphrase(ctx, *passphrase)
	if err != nil {
		return err
	}

	a, err := app.Open(ctx)
	if err != nil {
		return err
	}
	defer a.Close()

	f, err := createSecureFile(*out)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer f.Close()

	if err := a.Security.Backup(ctx, key, f); err != nil {
		os.Remove(*out)
		return err
	}
	fmt.Fprintf(iostreams.Stdout, "backup written to %s\n", *out)
	return nil
}

func backupRestore(ctx context.Context, iostreams IO, args []string) error {
	fs := flag.NewFlagSet("backup restore", flag.ContinueOnError)
	fs.SetOutput(iostreams.Stderr)
	in := fs.String("in", "", "backup file to restore (required)")
	target := fs.String("target", "", "path to write the restored database (required; must be a clean profile)")
	passphrase := fs.String("passphrase", "", "encryption passphrase (falls back to env/SymVault)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *in == "" || *target == "" {
		return fmt.Errorf("usage: symrelate backup restore --in <file> --target <db-path> [--passphrase ...]")
	}

	key, err := resolvePassphrase(ctx, *passphrase)
	if err != nil {
		return err
	}

	f, err := os.Open(*in)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer f.Close()

	if err := app.RestoreBackup(ctx, key, f, *target); err != nil {
		return err
	}
	fmt.Fprintf(iostreams.Stdout, "restored to %s\n", *target)
	return nil
}
