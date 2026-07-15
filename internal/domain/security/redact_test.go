package security

import (
	"strings"
	"testing"
)

func TestRedact_MasksEmailAndPhone(t *testing.T) {
	in := "conflict on ada@example.com and +1 (555) 123-4567 for person p1"
	out := Redact(in)

	if strings.Contains(out, "ada@example.com") {
		t.Errorf("Redact() did not mask email: %q", out)
	}
	if strings.Contains(out, "555") {
		t.Errorf("Redact() did not mask phone: %q", out)
	}
	if !strings.Contains(out, "[redacted-email]") || !strings.Contains(out, "[redacted-phone]") {
		t.Errorf("Redact() = %q, want redaction markers", out)
	}
	if !strings.Contains(out, "person p1") {
		t.Errorf("Redact() over-redacted non-sensitive text: %q", out)
	}
}

func TestRedact_LeavesPlainTextUntouched(t *testing.T) {
	in := "person not found"
	if out := Redact(in); out != in {
		t.Errorf("Redact(%q) = %q, want unchanged", in, out)
	}
}
