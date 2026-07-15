package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func setTestProfileDirs(t *testing.T) {
	t.Helper()
	t.Setenv("SYMRELATE_CONFIG_HOME", t.TempDir())
	t.Setenv("SYMRELATE_DATA_HOME", t.TempDir())
	t.Setenv("SYMRELATE_CACHE_HOME", t.TempDir())
}

func runCLI(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errBuf bytes.Buffer
	code = Run(context.Background(), IO{Stdout: &out, Stderr: &errBuf}, args)
	return out.String(), errBuf.String(), code
}

// TestContactLifecycle_EndToEnd drives the full contact/org/membership CLI
// surface, including flag-then-positional-id commands (add-point, update)
// — flags must come before the trailing id because Go's flag package stops
// parsing at the first non-flag argument.
func TestContactLifecycle_EndToEnd(t *testing.T) {
	setTestProfileDirs(t)

	out, stderr, code := runCLI(t, "contact", "add", "--name", "Ada Lovelace", "--email", "ada@example.com")
	if code != 0 {
		t.Fatalf("contact add: code=%d stderr=%s", code, stderr)
	}
	var person map[string]any
	if err := json.Unmarshal([]byte(out), &person); err != nil {
		t.Fatalf("contact add: invalid JSON: %v (%s)", err, out)
	}
	personID, _ := person["ID"].(string)
	if personID == "" {
		t.Fatalf("contact add: missing ID in %s", out)
	}

	if _, stderr, code = runCLI(t, "contact", "add-point", "--kind", "phone", "--value", "+1 555 000 1111", personID); code != 0 {
		t.Fatalf("contact add-point: code=%d stderr=%s", code, stderr)
	}

	if _, stderr, code = runCLI(t, "contact", "update", "--name", "Ada King", personID); code != 0 {
		t.Fatalf("contact update: code=%d stderr=%s", code, stderr)
	}

	out, stderr, code = runCLI(t, "contact", "show", personID)
	if code != 0 {
		t.Fatalf("contact show: code=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(out, "Ada King") {
		t.Errorf("contact show output missing updated name: %s", out)
	}
	if !strings.Contains(out, "+1 555 000 1111") {
		t.Errorf("contact show output missing added phone: %s", out)
	}

	if _, stderr, code = runCLI(t, "contact", "tag", personID, "vip"); code != 0 {
		t.Fatalf("contact tag: code=%d stderr=%s", code, stderr)
	}
	if _, stderr, code = runCLI(t, "contact", "classify", personID, "business"); code != 0 {
		t.Fatalf("contact classify: code=%d stderr=%s", code, stderr)
	}

	out, stderr, code = runCLI(t, "org", "add", "--name", "Acme Analytical Engines")
	if code != 0 {
		t.Fatalf("org add: code=%d stderr=%s", code, stderr)
	}
	var org map[string]any
	if err := json.Unmarshal([]byte(out), &org); err != nil {
		t.Fatalf("org add: invalid JSON: %v (%s)", err, out)
	}
	orgID, _ := org["ID"].(string)

	if _, stderr, code = runCLI(t, "membership", "add", "--person", personID, "--org", orgID, "--role", "employee"); code != 0 {
		t.Fatalf("membership add: code=%d stderr=%s", code, stderr)
	}

	out, stderr, code = runCLI(t, "membership", "list-person", personID)
	if code != 0 {
		t.Fatalf("membership list-person: code=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(out, orgID) {
		t.Errorf("membership list-person missing org id: %s", out)
	}

	// Duplicate contact point on the same person must be rejected.
	_, stderr, code = runCLI(t, "contact", "add-point", "--kind", "email", "--value", "ADA@EXAMPLE.COM", personID)
	if code == 0 {
		t.Fatalf("expected non-zero exit for duplicate contact point, stderr=%s", stderr)
	}

	if _, stderr, code = runCLI(t, "contact", "delete", personID); code != 0 {
		t.Fatalf("contact delete: code=%d stderr=%s", code, stderr)
	}
	if _, _, code = runCLI(t, "contact", "show", personID); code == 0 {
		t.Fatal("contact show after delete: expected non-zero exit")
	}
}
