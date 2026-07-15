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
		Name:  "contact",
		Short: "Manage people (add, show, list, update, delete)",
		Run:   runContact,
	})
}

func runContact(ctx context.Context, iostreams IO, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: symrelate contact <add|show|list|update|delete|erase|tag|classify|add-point> ...")
	}
	verb, rest := args[0], args[1:]

	a, err := app.Open(ctx)
	if err != nil {
		return err
	}
	defer a.Close()

	switch verb {
	case "add":
		return contactAdd(ctx, iostreams, a, rest)
	case "show":
		return contactShow(ctx, iostreams, a, rest)
	case "list":
		return contactList(ctx, iostreams, a, rest)
	case "update":
		return contactUpdate(ctx, iostreams, a, rest)
	case "delete":
		return contactDelete(ctx, iostreams, a, rest)
	case "erase":
		return contactErase(ctx, iostreams, a, rest)
	case "tag":
		return contactTag(ctx, iostreams, a, rest)
	case "classify":
		return contactClassify(ctx, iostreams, a, rest)
	case "add-point":
		return contactAddPoint(ctx, iostreams, a, rest)
	default:
		return fmt.Errorf("symrelate contact: unknown subcommand %q", verb)
	}
}

func contactAdd(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	fs := flag.NewFlagSet("contact add", flag.ContinueOnError)
	fs.SetOutput(iostreams.Stderr)
	name := fs.String("name", "", "display name (required)")
	given := fs.String("given", "", "given name")
	family := fs.String("family", "", "family name")
	notes := fs.String("notes", "", "free-text notes")
	email := fs.String("email", "", "email contact point")
	phone := fs.String("phone", "", "phone contact point")
	if err := fs.Parse(args); err != nil {
		return err
	}

	p, err := a.Contacts.CreatePerson(ctx, contact.PersonInput{
		DisplayName: *name, GivenName: *given, FamilyName: *family, Notes: *notes,
	})
	if err != nil {
		return err
	}
	if *email != "" {
		if _, err := a.Contacts.AddPersonContactPoint(ctx, p.ID, contact.ContactPointInput{Kind: contact.ContactPointEmail, RawValue: *email}); err != nil {
			return err
		}
	}
	if *phone != "" {
		if _, err := a.Contacts.AddPersonContactPoint(ctx, p.ID, contact.ContactPointInput{Kind: contact.ContactPointPhone, RawValue: *phone}); err != nil {
			return err
		}
	}

	p, err = a.Contacts.GetPerson(ctx, p.ID)
	if err != nil {
		return err
	}
	return printJSON(iostreams, p)
}

func contactShow(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: symrelate contact show <id>")
	}
	p, err := a.Contacts.GetPerson(ctx, args[0])
	if err != nil {
		return err
	}
	return printJSON(iostreams, p)
}

func contactList(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	fs := flag.NewFlagSet("contact list", flag.ContinueOnError)
	fs.SetOutput(iostreams.Stderr)
	classification := fs.String("classification", "", "filter by classification")
	limit := fs.Int("limit", page.DefaultLimit, "max results")
	offset := fs.Int("offset", 0, "result offset")
	if err := fs.Parse(args); err != nil {
		return err
	}

	result, err := a.Contacts.ListPersons(ctx, app.ListPersonsOptions{
		Classification: contact.Classification(*classification),
		Page:           page.Request{Limit: *limit, Offset: *offset},
	})
	if err != nil {
		return err
	}
	return printJSON(iostreams, result)
}

func contactUpdate(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	fs := flag.NewFlagSet("contact update", flag.ContinueOnError)
	fs.SetOutput(iostreams.Stderr)
	name := fs.String("name", "", "new display name")
	notes := fs.String("notes", "", "new notes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: symrelate contact update [--name ...] [--notes ...] <id>")
	}

	upd := contact.PersonUpdate{}
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "name":
			upd.DisplayName = name
		case "notes":
			upd.Notes = notes
		}
	})

	p, err := a.Contacts.UpdatePerson(ctx, fs.Arg(0), upd)
	if err != nil {
		return err
	}
	return printJSON(iostreams, p)
}

func contactDelete(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: symrelate contact delete <id>")
	}
	if err := a.Contacts.DeletePerson(ctx, args[0]); err != nil {
		return err
	}
	fmt.Fprintln(iostreams.Stdout, "deleted")
	return nil
}

// contactErase is the audited privacy-erasure workflow (see
// docs/PRIVACY.md): unlike delete, it records an audit event with counts
// of what was removed and returns that summary to the caller.
func contactErase(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: symrelate contact erase <id>")
	}
	summary, err := a.Security.EraseContact(ctx, args[0])
	if err != nil {
		return err
	}
	return printJSON(iostreams, summary)
}

func contactTag(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: symrelate contact tag <id> <tag>")
	}
	if err := a.Contacts.AddPersonTag(ctx, args[0], args[1]); err != nil {
		return err
	}
	fmt.Fprintln(iostreams.Stdout, "tagged")
	return nil
}

func contactClassify(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: symrelate contact classify <id> <classification>")
	}
	if err := a.Contacts.SetPersonClassification(ctx, args[0], contact.Classification(args[1])); err != nil {
		return err
	}
	fmt.Fprintln(iostreams.Stdout, "classified")
	return nil
}

func contactAddPoint(ctx context.Context, iostreams IO, a *app.App, args []string) error {
	fs := flag.NewFlagSet("contact add-point", flag.ContinueOnError)
	fs.SetOutput(iostreams.Stderr)
	kind := fs.String("kind", "", "email|phone|address|url|handle")
	value := fs.String("value", "", "contact point value")
	label := fs.String("label", "", "label, e.g. work/home")
	preferred := fs.Bool("preferred", false, "mark as preferred")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: symrelate contact add-point --kind ... --value ... <id>")
	}

	cp, err := a.Contacts.AddPersonContactPoint(ctx, fs.Arg(0), contact.ContactPointInput{
		Kind: contact.ContactPointKind(*kind), RawValue: *value, Label: *label, IsPreferred: *preferred,
	})
	if err != nil {
		return err
	}
	return printJSON(iostreams, cp)
}
