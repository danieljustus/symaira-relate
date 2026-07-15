package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/danieljustus/symaira-relate/internal/app"
	"github.com/danieljustus/symaira-relate/internal/domain/relationship"
)

func init() {
	Register(&Command{
		Name:  "relate",
		Short: "Manage directed relationships between contacts",
		Run:   runRelate,
	})
}

func runRelate(ctx context.Context, iostreams IO, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: symrelate relate <add|outgoing|incoming-person|incoming-org> ...")
	}
	verb, rest := args[0], args[1:]

	a, err := app.Open(ctx)
	if err != nil {
		return err
	}
	defer a.Close()

	switch verb {
	case "add":
		return relateAdd(ctx, iostreams, a, rest)
	case "outgoing":
		return relateOutgoing(ctx, iostreams, a, rest)
	case "incoming-person":
		return relateIncomingPerson(ctx, iostreams, a, rest)
	case "incoming-org":
		return relateIncomingOrg(ctx, iostreams, a, rest)
	default:
		return fmt.Errorf("symrelate relate: unknown subcommand %q", verb)
	}
}

func relateAdd(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	fs := flag.NewFlagSet("relate add", flag.ContinueOnError)
	fs.SetOutput(iostreams.Stderr)
	from := fs.String("from", "", "subject person id (required)")
	toPerson := fs.String("to-person", "", "target person id")
	toOrg := fs.String("to-org", "", "target organization id")
	relType := fs.String("type", "", "relationship type, e.g. colleague/friend/manager (required)")
	notes := fs.String("notes", "", "free-text notes")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rel, err := a.Relationships.AddRelationship(ctx, relationship.RelationshipInput{
		FromPersonID: *from, ToPersonID: *toPerson, ToOrganizationID: *toOrg, Type: *relType, Notes: *notes,
	})
	if err != nil {
		return err
	}
	return printJSON(iostreams, rel)
}

func relateOutgoing(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: symrelate relate outgoing <person-id>")
	}
	list, err := a.Relationships.ListOutgoingFromPerson(ctx, args[0])
	if err != nil {
		return err
	}
	return printJSON(iostreams, list)
}

func relateIncomingPerson(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: symrelate relate incoming-person <person-id>")
	}
	list, err := a.Relationships.ListIncomingToPerson(ctx, args[0])
	if err != nil {
		return err
	}
	return printJSON(iostreams, list)
}

func relateIncomingOrg(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: symrelate relate incoming-org <org-id>")
	}
	list, err := a.Relationships.ListIncomingToOrganization(ctx, args[0])
	if err != nil {
		return err
	}
	return printJSON(iostreams, list)
}
