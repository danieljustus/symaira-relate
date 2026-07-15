// Package security defines the key-provider abstraction, redaction rules
// and audit/backup/erase domain types for the beta privacy boundary. See
// docs/PRIVACY.md for the full data-handling policy.
package security

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
)

// ErrKeyUnavailable is returned by a KeyProvider that has no key material
// to offer right now (SymVault not installed, no passphrase configured,
// ...). It is not fatal — callers try the next provider in a Chain.
var ErrKeyUnavailable = errors.New("security: key unavailable")

// KeyProvider resolves the passphrase used to derive a backup/export
// encryption key. Implementations must be safe to call in tests with no
// real secret store present — see StaticKeyProvider.
type KeyProvider interface {
	// Resolve returns the passphrase and a short, non-sensitive label
	// identifying where it came from (for diagnostics — never the
	// passphrase itself).
	Resolve(ctx context.Context) (passphrase []byte, source string, err error)
}

// Chain tries each provider in order and returns the first success. If
// every provider reports ErrKeyUnavailable, Chain reports
// ErrKeyUnavailable too; any other error aborts immediately.
type Chain []KeyProvider

func (c Chain) Resolve(ctx context.Context) ([]byte, string, error) {
	for _, p := range c {
		key, source, err := p.Resolve(ctx)
		if err == nil {
			return key, source, nil
		}
		if !errors.Is(err, ErrKeyUnavailable) {
			return nil, "", err
		}
	}
	return nil, "", ErrKeyUnavailable
}

// StaticKeyProvider wraps an explicitly supplied passphrase (a CLI flag,
// a test fixture). This is the standalone fallback that always works with
// no external tool installed.
type StaticKeyProvider struct {
	Passphrase []byte
}

func (p StaticKeyProvider) Resolve(context.Context) ([]byte, string, error) {
	if len(p.Passphrase) == 0 {
		return nil, "", ErrKeyUnavailable
	}
	return p.Passphrase, "explicit", nil
}

// DefaultPassphraseEnvVar is the environment variable EnvKeyProvider reads
// by default.
const DefaultPassphraseEnvVar = "SYMRELATE_BACKUP_PASSPHRASE"

// EnvKeyProvider reads a passphrase from an environment variable.
type EnvKeyProvider struct {
	VarName string // defaults to DefaultPassphraseEnvVar when empty
}

func (p EnvKeyProvider) Resolve(context.Context) ([]byte, string, error) {
	name := p.VarName
	if name == "" {
		name = DefaultPassphraseEnvVar
	}
	v := os.Getenv(name)
	if v == "" {
		return nil, "", ErrKeyUnavailable
	}
	return []byte(v), "env:" + name, nil
}

// DefaultSymVaultKeyName is the key name symrelate asks SymVault for.
const DefaultSymVaultKeyName = "symrelate-backup"

// SymVaultKeyProvider resolves a passphrase from a local SymVault
// installation, detected at runtime via a PATH lookup only — never a
// compile-time dependency (see ARCHITECTURE.md's standalone-first rule).
// When symvault is not installed, Resolve returns ErrKeyUnavailable
// immediately so callers fall back to the next provider: SymVault is
// always a convenience, never a requirement.
type SymVaultKeyProvider struct {
	KeyName string // defaults to DefaultSymVaultKeyName when empty
}

func (p SymVaultKeyProvider) Resolve(ctx context.Context) ([]byte, string, error) {
	binPath, err := exec.LookPath("symvault")
	if err != nil {
		return nil, "", ErrKeyUnavailable
	}
	keyName := p.KeyName
	if keyName == "" {
		keyName = DefaultSymVaultKeyName
	}
	out, err := exec.CommandContext(ctx, binPath, "get", keyName).Output()
	if err != nil {
		return nil, "", ErrKeyUnavailable
	}
	passphrase := bytes.TrimSpace(out)
	if len(passphrase) == 0 {
		return nil, "", ErrKeyUnavailable
	}
	return passphrase, "symvault", nil
}

// DefaultKeyProviders returns the standard resolution order: an explicit
// passphrase (if any), then the environment variable, then SymVault if
// installed. Every step before SymVault works with zero external
// dependencies, satisfying the documented standalone fallback.
func DefaultKeyProviders(explicit []byte) Chain {
	return Chain{
		StaticKeyProvider{Passphrase: explicit},
		EnvKeyProvider{},
		SymVaultKeyProvider{},
	}
}
