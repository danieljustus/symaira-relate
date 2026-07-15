package security

import (
	"context"
	"errors"
	"testing"
)

func TestStaticKeyProvider_UnavailableWhenEmpty(t *testing.T) {
	_, _, err := StaticKeyProvider{}.Resolve(context.Background())
	if !errors.Is(err, ErrKeyUnavailable) {
		t.Fatalf("error = %v, want ErrKeyUnavailable", err)
	}
}

func TestEnvKeyProvider_UnavailableWhenUnset(t *testing.T) {
	t.Setenv("SYMRELATE_TEST_PASSPHRASE_UNSET", "")
	_, _, err := EnvKeyProvider{VarName: "SYMRELATE_TEST_PASSPHRASE_UNSET"}.Resolve(context.Background())
	if !errors.Is(err, ErrKeyUnavailable) {
		t.Fatalf("error = %v, want ErrKeyUnavailable", err)
	}
}

func TestEnvKeyProvider_ResolvesFromEnv(t *testing.T) {
	t.Setenv("SYMRELATE_TEST_PASSPHRASE", "hunter2")
	key, source, err := EnvKeyProvider{VarName: "SYMRELATE_TEST_PASSPHRASE"}.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if string(key) != "hunter2" {
		t.Errorf("key = %q, want hunter2", key)
	}
	if source == "" {
		t.Error("source is empty, want a diagnostic label")
	}
}

// TestSymVaultKeyProvider_UnavailableWhenNotInstalled documents the
// standalone fallback: when symvault is not on PATH (the case in this CI
// environment and in every standalone install), Resolve must return
// ErrKeyUnavailable immediately rather than blocking or erroring hard —
// SymVault is a convenience, never a requirement.
func TestSymVaultKeyProvider_UnavailableWhenNotInstalled(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // guarantee no "symvault" binary is found
	_, _, err := SymVaultKeyProvider{}.Resolve(context.Background())
	if !errors.Is(err, ErrKeyUnavailable) {
		t.Fatalf("error = %v, want ErrKeyUnavailable", err)
	}
}

func TestChain_FallsThroughToStandaloneProvider(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	chain := DefaultKeyProviders(nil) // no explicit passphrase either
	if _, _, err := chain.Resolve(context.Background()); !errors.Is(err, ErrKeyUnavailable) {
		t.Fatalf("error = %v, want ErrKeyUnavailable when no provider has a key", err)
	}

	chain = DefaultKeyProviders([]byte("explicit-passphrase"))
	key, source, err := chain.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if string(key) != "explicit-passphrase" || source != "explicit" {
		t.Errorf("key=%q source=%q, want explicit-passphrase/explicit", key, source)
	}
}

func TestChain_StopsOnNonUnavailableError(t *testing.T) {
	boom := errors.New("boom")
	chain := Chain{failingProvider{err: boom}, StaticKeyProvider{Passphrase: []byte("should-not-be-reached")}}
	_, _, err := chain.Resolve(context.Background())
	if !errors.Is(err, boom) {
		t.Fatalf("error = %v, want boom (chain must not swallow non-ErrKeyUnavailable errors)", err)
	}
}

type failingProvider struct{ err error }

func (f failingProvider) Resolve(context.Context) ([]byte, string, error) {
	return nil, "", f.err
}
