package cli

import (
	"context"
	"flag"
	"fmt"
	"os/user"

	"github.com/danieljustus/symaira-relate/internal/app"
	"github.com/danieljustus/symaira-relate/internal/domain/memorylink"
	"github.com/danieljustus/symaira-relate/internal/integration/symmemory"
)

// localUser identifies who requested a link for the memory_links audit
// column — the local OS username, never contact data, mirroring
// internal/service/security's localActor for the same reason (see
// docs/PRIVACY.md).
func localUser() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return "unknown"
}

func init() {
	Register(&Command{
		Name:  "memory",
		Short: "Optional SymMemory context links (status, link, unlink, show) — see docs/integrations/SYMMEMORY.md",
		Run:   runMemory,
	})
}

func runMemory(ctx context.Context, iostreams IO, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: symrelate memory <status|link|unlink|show> ...")
	}
	verb, rest := args[0], args[1:]

	switch verb {
	case "status":
		return memoryStatus(ctx, iostreams)
	case "link":
		return memoryLink(ctx, iostreams, rest)
	case "unlink":
		return memoryUnlink(ctx, iostreams, rest)
	case "show":
		return memoryShow(ctx, iostreams, rest)
	default:
		return fmt.Errorf("symrelate memory: unknown subcommand %q", verb)
	}
}

// memoryStatus reports whether a compatible SymMemory installation was
// found — never an error, since SymMemory is always optional (see
// ARCHITECTURE.md's standalone-first rule).
func memoryStatus(ctx context.Context, iostreams IO) error {
	info, err := symmemory.Discover(ctx)
	if err != nil {
		return printJSON(iostreams, map[string]any{"available": false, "reason": err.Error()})
	}
	return printJSON(iostreams, map[string]any{"available": true, "tool": info.Tool, "version": info.Version, "schema_version": info.SchemaVersion})
}

func memoryLink(ctx context.Context, iostreams IO, args []string) error {
	fs := flag.NewFlagSet("memory link", flag.ContinueOnError)
	fs.SetOutput(iostreams.Stderr)
	person := fs.String("person", "", "person id")
	org := fs.String("org", "", "organization id")
	entity := fs.String("entity", "", "SymMemory entity id, found by looking it up yourself in SymMemory first (required)")
	entityType := fs.String("entity-type", "", "SymMemory entity type, e.g. person (optional)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if (*person == "") == (*org == "") {
		return fmt.Errorf("exactly one of --person or --org is required")
	}
	if *entity == "" {
		return fmt.Errorf("--entity is required — linking is a reviewed action: look up the SymMemory entity id yourself before linking")
	}

	a, err := app.Open(ctx)
	if err != nil {
		return err
	}
	defer a.Close()

	in := memorylink.LinkInput{MemoryEntityID: *entity, MemoryEntityType: *entityType, LinkedBy: localUser()}
	var link *memorylink.Link
	if *person != "" {
		link, err = a.MemoryLinks.LinkPerson(ctx, *person, in)
	} else {
		link, err = a.MemoryLinks.LinkOrganization(ctx, *org, in)
	}
	if err != nil {
		return err
	}
	return printJSON(iostreams, link)
}

func memoryUnlink(ctx context.Context, iostreams IO, args []string) error {
	fs := flag.NewFlagSet("memory unlink", flag.ContinueOnError)
	fs.SetOutput(iostreams.Stderr)
	person := fs.String("person", "", "person id")
	org := fs.String("org", "", "organization id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if (*person == "") == (*org == "") {
		return fmt.Errorf("exactly one of --person or --org is required")
	}

	a, err := app.Open(ctx)
	if err != nil {
		return err
	}
	defer a.Close()

	if *person != "" {
		err = a.MemoryLinks.UnlinkPerson(ctx, *person)
	} else {
		err = a.MemoryLinks.UnlinkOrganization(ctx, *org)
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(iostreams.Stdout, "unlinked")
	return nil
}

// memoryShow prints the local link (if any) and, when SymMemory is
// available, its opaque context JSON. Contact-point data is never sent to
// SymMemory to answer this — only the already-linked entity id.
func memoryShow(ctx context.Context, iostreams IO, args []string) error {
	fs := flag.NewFlagSet("memory show", flag.ContinueOnError)
	fs.SetOutput(iostreams.Stderr)
	person := fs.String("person", "", "person id")
	org := fs.String("org", "", "organization id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if (*person == "") == (*org == "") {
		return fmt.Errorf("exactly one of --person or --org is required")
	}

	a, err := app.Open(ctx)
	if err != nil {
		return err
	}
	defer a.Close()

	var link *memorylink.Link
	if *person != "" {
		link, err = a.MemoryLinks.GetPersonLink(ctx, *person)
	} else {
		link, err = a.MemoryLinks.GetOrganizationLink(ctx, *org)
	}
	if err != nil {
		return err
	}
	if link == nil {
		return printJSON(iostreams, map[string]any{"linked": false})
	}

	contextJSON, ctxErr := symmemory.Context(ctx, link.MemoryEntityID)
	result := map[string]any{"linked": true, "link": link}
	if ctxErr != nil {
		result["context_available"] = false
	} else {
		result["context_available"] = true
		result["context"] = rawJSON(contextJSON)
	}
	return printJSON(iostreams, result)
}

// rawJSON marks a pre-formatted JSON string so printJSON embeds it as a
// JSON value rather than a quoted string.
type rawJSON string

func (r rawJSON) MarshalJSON() ([]byte, error) { return []byte(r), nil }
