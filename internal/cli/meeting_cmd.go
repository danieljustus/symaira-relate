package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/danieljustus/symaira-relate/internal/app"
	"github.com/danieljustus/symaira-relate/internal/integration/symmeet"
	relationshipsvc "github.com/danieljustus/symaira-relate/internal/service/relationship"
)

func init() {
	Register(&Command{
		Name:  "meeting",
		Short: "Import a reviewed SymMeet meeting as a contact interaction — see docs/integrations/SYMMEET.md",
		Run:   runMeeting,
	})
}

func runMeeting(ctx context.Context, iostreams IO, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: symrelate meeting <status|import> ...")
	}
	verb, rest := args[0], args[1:]

	switch verb {
	case "status":
		return meetingStatus(ctx, iostreams)
	case "import":
		return meetingImport(ctx, iostreams, rest)
	default:
		return fmt.Errorf("symrelate meeting: unknown subcommand %q", verb)
	}
}

// meetingStatus reports whether a compatible SymMeet installation was
// found — never an error, since SymMeet is always optional.
func meetingStatus(ctx context.Context, iostreams IO) error {
	caps, err := symmeet.Discover(ctx)
	if err != nil {
		return printJSON(iostreams, map[string]any{"available": false, "reason": err.Error()})
	}
	return printJSON(iostreams, map[string]any{"available": true, "tool": caps.Tool, "version": caps.Version, "schema_version": caps.SchemaVersion})
}

// meetingFixture is the provisional, assumed shape of a local SymMeet
// artifact file — see docs/integrations/SYMMEET.md. The real published
// contract (symaira-meet#19, open) may differ; this is a local input
// convenience, not a network or binary-artifact import, and every field
// it supplies can be overridden by an explicit flag.
type meetingFixture struct {
	MeetingID  string `json:"meeting_id"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	OccurredAt string `json:"occurred_at"`
}

func meetingImport(ctx context.Context, iostreams IO, args []string) error {
	fs := flag.NewFlagSet("meeting import", flag.ContinueOnError)
	fs.SetOutput(iostreams.Stderr)
	id := fs.String("id", "", "opaque SymMeet meeting id (required unless --fixture supplies one)")
	title := fs.String("title", "", "meeting title")
	summary := fs.String("summary", "", "short summary")
	occurred := fs.String("occurred", "", "RFC3339 timestamp; defaults to now")
	personsFlag := fs.String("person", "", "comma-separated person ids to attach this meeting to")
	orgsFlag := fs.String("org", "", "comma-separated organization ids to attach this meeting to")
	fixture := fs.String("fixture", "", "path to a local JSON file supplying meeting_id/title/summary/occurred_at (PROVISIONAL shape, see docs/integrations/SYMMEET.md); explicit flags above override it")
	if err := fs.Parse(args); err != nil {
		return err
	}

	in := relationshipsvc.MeetingImportInput{MeetingID: *id, Title: *title, Summary: *summary}
	if *fixture != "" {
		if err := applyMeetingFixture(*fixture, &in); err != nil {
			return err
		}
	}
	if in.MeetingID == "" {
		return fmt.Errorf("usage: symrelate meeting import --id <meeting-id> (--person <id>[,...] | --org <id>[,...]) [--fixture <path>]")
	}

	if *occurred != "" {
		t, err := time.Parse(time.RFC3339, *occurred)
		if err != nil {
			return fmt.Errorf("--occurred: %w", err)
		}
		in.OccurredAt = t
	} else if in.OccurredAt.IsZero() {
		in.OccurredAt = time.Now().UTC()
	}

	persons := splitNonEmpty(*personsFlag)
	orgs := splitNonEmpty(*orgsFlag)
	if len(persons) == 0 && len(orgs) == 0 {
		return fmt.Errorf("at least one of --person or --org is required — contact association is always manual and reviewed, never automatic")
	}

	a, err := app.Open(ctx)
	if err != nil {
		return err
	}
	defer a.Close()

	type outcome struct {
		PersonID       string `json:"person_id,omitempty"`
		OrganizationID string `json:"organization_id,omitempty"`
		InteractionID  string `json:"interaction_id"`
		Created        bool   `json:"created"`
	}
	var results []outcome
	for _, personID := range persons {
		interaction, created, err := a.Relationships.ImportPersonMeeting(ctx, personID, in)
		if err != nil {
			return err
		}
		results = append(results, outcome{PersonID: personID, InteractionID: interaction.ID, Created: created})
	}
	for _, orgID := range orgs {
		interaction, created, err := a.Relationships.ImportOrganizationMeeting(ctx, orgID, in)
		if err != nil {
			return err
		}
		results = append(results, outcome{OrganizationID: orgID, InteractionID: interaction.ID, Created: created})
	}

	return printJSON(iostreams, results)
}

func applyMeetingFixture(path string, in *relationshipsvc.MeetingImportInput) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read --fixture file: %w", err)
	}
	var fx meetingFixture
	if err := json.Unmarshal(data, &fx); err != nil {
		return fmt.Errorf("--fixture: invalid JSON: %w", err)
	}
	if in.MeetingID == "" {
		in.MeetingID = fx.MeetingID
	}
	if in.Title == "" {
		in.Title = fx.Title
	}
	if in.Summary == "" {
		in.Summary = fx.Summary
	}
	if fx.OccurredAt != "" && in.OccurredAt.IsZero() {
		t, err := time.Parse(time.RFC3339, fx.OccurredAt)
		if err != nil {
			return fmt.Errorf("--fixture: occurred_at: %w", err)
		}
		in.OccurredAt = t
	}
	return nil
}

func splitNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
