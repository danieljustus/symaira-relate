package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/danieljustus/symaira-relate/internal/app"
	"github.com/danieljustus/symaira-relate/internal/domain/contact"
	"github.com/danieljustus/symaira-relate/internal/domain/page"
)

func init() {
	Register(&Command{
		Name:  "org",
		Short: "Manage organizations (add, show, list, update, delete)",
		Run:   runOrg,
	})
}

func runOrg(ctx context.Context, iostreams IO, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: symrelate org <add|show|list|update|delete|erase|tag|classify|add-point> ...")
	}
	verb, rest := args[0], args[1:]

	a, err := app.Open(ctx)
	if err != nil {
		return err
	}
	defer a.Close()

	switch verb {
	case "add":
		return orgAdd(ctx, iostreams, a, rest)
	case "show":
		return orgShow(ctx, iostreams, a, rest)
	case "list":
		return orgList(ctx, iostreams, a, rest)
	case "update":
		return orgUpdate(ctx, iostreams, a, rest)
	case "delete":
		return orgDelete(ctx, iostreams, a, rest)
	case "erase":
		return orgErase(ctx, iostreams, a, rest)
	case "tag":
		return orgTag(ctx, iostreams, a, rest)
	case "classify":
		return orgClassify(ctx, iostreams, a, rest)
	case "add-point":
		return orgAddPoint(ctx, iostreams, a, rest)
	default:
		return fmt.Errorf("symrelate org: unknown subcommand %q", verb)
	}
}

func orgAdd(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	fs := flag.NewFlagSet("org add", flag.ContinueOnError)
	fs.SetOutput(iostreams.Stderr)
	name := fs.String("name", "", "organization name (required)")
	notes := fs.String("notes", "", "free-text notes")
	if err := fs.Parse(args); err != nil {
		return err
	}

	o, err := a.Contacts.CreateOrganization(ctx, contact.OrganizationInput{Name: *name, Notes: *notes})
	if err != nil {
		return err
	}
	return printJSON(iostreams, o)
}

func orgShow(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: symrelate org show <id>")
	}
	o, err := a.Contacts.GetOrganization(ctx, args[0])
	if err != nil {
		return err
	}
	return printJSON(iostreams, o)
}

func orgList(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	fs := flag.NewFlagSet("org list", flag.ContinueOnError)
	fs.SetOutput(iostreams.Stderr)
	classification := fs.String("classification", "", "filter by classification")
	limit := fs.Int("limit", page.DefaultLimit, "max results")
	offset := fs.Int("offset", 0, "result offset")
	if err := fs.Parse(args); err != nil {
		return err
	}

	result, err := a.Contacts.ListOrganizations(ctx, app.ListOrganizationsOptions{
		Classification: contact.Classification(*classification),
		Page:           page.Request{Limit: *limit, Offset: *offset},
	})
	if err != nil {
		return err
	}
	return printJSON(iostreams, result)
}

func orgUpdate(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	fs := flag.NewFlagSet("org update", flag.ContinueOnError)
	fs.SetOutput(iostreams.Stderr)
	name := fs.String("name", "", "new name")
	notes := fs.String("notes", "", "new notes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: symrelate org update [--name ...] [--notes ...] <id>")
	}

	upd := contact.OrganizationUpdate{}
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "name":
			upd.Name = name
		case "notes":
			upd.Notes = notes
		}
	})

	o, err := a.Contacts.UpdateOrganization(ctx, fs.Arg(0), upd)
	if err != nil {
		return err
	}
	return printJSON(iostreams, o)
}

func orgDelete(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: symrelate org delete <id>")
	}
	if err := a.Contacts.DeleteOrganization(ctx, args[0]); err != nil {
		return err
	}
	fmt.Fprintln(iostreams.Stdout, "deleted")
	return nil
}

// orgErase is the audited privacy-erasure workflow — see contactErase.
func orgErase(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: symrelate org erase <id>")
	}
	summary, err := a.Security.EraseOrganization(ctx, args[0])
	if err != nil {
		return err
	}
	return printJSON(iostreams, summary)
}

func orgTag(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: symrelate org tag <id> <tag>")
	}
	if err := a.Contacts.AddOrganizationTag(ctx, args[0], args[1]); err != nil {
		return err
	}
	fmt.Fprintln(iostreams.Stdout, "tagged")
	return nil
}

func orgClassify(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: symrelate org classify <id> <classification>")
	}
	if err := a.Contacts.SetOrganizationClassification(ctx, args[0], contact.Classification(args[1])); err != nil {
		return err
	}
	fmt.Fprintln(iostreams.Stdout, "classified")
	return nil
}

func orgAddPoint(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	fs := flag.NewFlagSet("org add-point", flag.ContinueOnError)
	fs.SetOutput(iostreams.Stderr)
	kind := fs.String("kind", "", "email|phone|address|url|handle")
	value := fs.String("value", "", "contact point value")
	label := fs.String("label", "", "label, e.g. hq/billing")
	preferred := fs.Bool("preferred", false, "mark as preferred")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: symrelate org add-point --kind ... --value ... <id>")
	}

	cp, err := a.Contacts.AddOrganizationContactPoint(ctx, fs.Arg(0), contact.ContactPointInput{
		Kind: contact.ContactPointKind(*kind), RawValue: *value, Label: *label, IsPreferred: *preferred,
	})
	if err != nil {
		return err
	}
	return printJSON(iostreams, cp)
}
