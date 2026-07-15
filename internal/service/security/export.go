package security

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/danieljustus/symaira-relate/internal/errs"
)

// ExportPerson is a data-minimized export record: identity and contact
// points only. Notes and audit metadata are deliberately left out of
// exports by default (see docs/PRIVACY.md's data-minimization policy).
type ExportPerson struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"display_name"`
	GivenName   string   `json:"given_name,omitempty"`
	FamilyName  string   `json:"family_name,omitempty"`
	Emails      []string `json:"emails,omitempty"`
	Phones      []string `json:"phones,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// ExportJSON writes every person as a JSON array to w.
func (s *Service) ExportJSON(ctx context.Context, w io.Writer) error {
	const op = "security.ExportJSON"
	people, err := s.exportPeople(ctx)
	if err != nil {
		return errs.Internal(op, "failed to load persons for export", err)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(people); err != nil {
		return errs.Internal(op, "failed to write JSON export", err)
	}
	return recordAudit(ctx, s.db, "export_json", "", "", fmt.Sprintf("persons=%d", len(people)))
}

// ExportCSV writes one row per person (id, names, joined emails/phones/
// tags) to w.
func (s *Service) ExportCSV(ctx context.Context, w io.Writer) error {
	const op = "security.ExportCSV"
	people, err := s.exportPeople(ctx)
	if err != nil {
		return errs.Internal(op, "failed to load persons for export", err)
	}

	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"id", "display_name", "given_name", "family_name", "emails", "phones", "tags"}); err != nil {
		return errs.Internal(op, "failed to write CSV header", err)
	}
	for _, p := range people {
		row := []string{p.ID, p.DisplayName, p.GivenName, p.FamilyName, strings.Join(p.Emails, ";"), strings.Join(p.Phones, ";"), strings.Join(p.Tags, ";")}
		if err := cw.Write(row); err != nil {
			return errs.Internal(op, "failed to write CSV row", err)
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return errs.Internal(op, "failed to flush CSV", err)
	}
	return recordAudit(ctx, s.db, "export_csv", "", "", fmt.Sprintf("persons=%d", len(people)))
}

func (s *Service) exportPeople(ctx context.Context) ([]ExportPerson, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, display_name, given_name, family_name FROM persons ORDER BY display_name COLLATE NOCASE, id")
	if err != nil {
		return nil, err
	}
	var out []ExportPerson
	for rows.Next() {
		var p ExportPerson
		if err := rows.Scan(&p.ID, &p.DisplayName, &p.GivenName, &p.FamilyName); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	for i := range out {
		if out[i].Emails, err = s.contactPointValues(ctx, out[i].ID, "email"); err != nil {
			return nil, err
		}
		if out[i].Phones, err = s.contactPointValues(ctx, out[i].ID, "phone"); err != nil {
			return nil, err
		}
		if out[i].Tags, err = s.personTags(ctx, out[i].ID); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Service) contactPointValues(ctx context.Context, personID, kind string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT raw_value FROM contact_points WHERE person_id = ? AND kind = ? ORDER BY is_preferred DESC, created_at", personID, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Service) personTags(ctx context.Context, personID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT t.name FROM tags t JOIN entity_tags et ON et.tag_id = t.id WHERE et.person_id = ? ORDER BY t.name", personID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
