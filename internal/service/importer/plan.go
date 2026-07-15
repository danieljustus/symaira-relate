package importer

import (
	"context"
	"database/sql"
	"strconv"

	contactdomain "github.com/danieljustus/symaira-relate/internal/domain/contact"
	"github.com/danieljustus/symaira-relate/internal/domain/importer"
	"github.com/danieljustus/symaira-relate/internal/errs"
)

// Plan computes duplicate candidates for rows already parsed by
// ParseVCard/ParseCSV. It only reads — a dry run is simply "call Plan and
// show the result," and Apply is a separate, explicit step, so there is
// no code path where planning alone can write data.
func (s *Service) Plan(ctx context.Context, source importer.SourceKind, rows []importer.ImportRow, issues []importer.ValidationIssue) (*importer.ImportPlan, error) {
	const op = "importer.Plan"

	plan := &importer.ImportPlan{Source: source, Rows: rows, Issues: issues}
	for _, row := range rows {
		candidates, err := s.duplicateCandidates(ctx, source, row)
		if err != nil {
			return nil, errs.Internal(op, "failed to compute duplicate candidates", err)
		}
		plan.Duplicates = append(plan.Duplicates, candidates...)
	}
	return plan, nil
}

func (s *Service) duplicateCandidates(ctx context.Context, source importer.SourceKind, row importer.ImportRow) ([]importer.DuplicateCandidate, error) {
	var out []importer.DuplicateCandidate
	seen := map[string]bool{} // personID -> already has an exact-source-rematch candidate

	// 1. Exact idempotency match: this exact source was imported before.
	if row.SourceRef != "" {
		personID, name, err := s.findImportSource(ctx, source, row.SourceRef)
		if err != nil {
			return nil, err
		}
		if personID != "" {
			out = append(out, importer.DuplicateCandidate{
				RowNumber: row.RowNumber, ExistingPersonID: personID, ExistingName: name,
				MatchedOn: "source:" + string(source) + ":" + row.SourceRef, ExactSourceRematch: true,
			})
			seen[personID] = true
		}
	}

	// 2. Normalized contact-point matches.
	for _, email := range row.Emails {
		matches, err := s.contactPointMatches(ctx, contactdomain.ContactPointEmail, contactdomain.NormalizeEmail(email))
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			if seen[m.id] {
				continue
			}
			out = append(out, importer.DuplicateCandidate{RowNumber: row.RowNumber, ExistingPersonID: m.id, ExistingName: m.name, MatchedOn: "email:" + email})
		}
	}
	for _, phone := range row.Phones {
		matches, err := s.contactPointMatches(ctx, contactdomain.ContactPointPhone, contactdomain.NormalizePhone(phone))
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			if seen[m.id] {
				continue
			}
			out = append(out, importer.DuplicateCandidate{RowNumber: row.RowNumber, ExistingPersonID: m.id, ExistingName: m.name, MatchedOn: "phone:" + phone})
		}
	}

	// 3. Normalized exact name match — a weaker signal, still
	// deterministic and explainable.
	nameMatches, err := s.nameMatches(ctx, row.DisplayName)
	if err != nil {
		return nil, err
	}
	for _, m := range nameMatches {
		if seen[m.id] {
			continue
		}
		out = append(out, importer.DuplicateCandidate{RowNumber: row.RowNumber, ExistingPersonID: m.id, ExistingName: m.name, MatchedOn: "name:" + row.DisplayName})
	}

	return dedupeCandidates(out), nil
}

// dedupeCandidates collapses multiple candidates for the same row+person
// down to the strongest match: an exact source rematch wins over a
// contact-point/name match, and only one entry per (row, person) survives.
func dedupeCandidates(in []importer.DuplicateCandidate) []importer.DuplicateCandidate {
	best := map[[2]string]importer.DuplicateCandidate{}
	var order [][2]string
	for _, c := range in {
		key := [2]string{strconv.Itoa(c.RowNumber), c.ExistingPersonID}
		existing, ok := best[key]
		if !ok {
			best[key] = c
			order = append(order, key)
			continue
		}
		if c.ExactSourceRematch && !existing.ExactSourceRematch {
			best[key] = c
		}
	}
	out := make([]importer.DuplicateCandidate, 0, len(order))
	for _, key := range order {
		out = append(out, best[key])
	}
	return out
}

type personMatch struct{ id, name string }

func (s *Service) findImportSource(ctx context.Context, source importer.SourceKind, sourceRef string) (personID, name string, err error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT p.id, p.display_name FROM import_sources src
		JOIN persons p ON p.id = src.person_id
		WHERE src.source_kind = ? AND src.source_ref = ?`, string(source), sourceRef)
	if err := row.Scan(&personID, &name); err != nil {
		if err == sql.ErrNoRows {
			return "", "", nil
		}
		return "", "", err
	}
	return personID, name, nil
}

func (s *Service) contactPointMatches(ctx context.Context, kind contactdomain.ContactPointKind, normalized string) ([]personMatch, error) {
	if normalized == "" {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT p.id, p.display_name FROM contact_points cp
		JOIN persons p ON p.id = cp.person_id
		WHERE cp.kind = ? AND cp.normalized_value = ?`, string(kind), normalized)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPersonMatches(rows)
}

func (s *Service) nameMatches(ctx context.Context, displayName string) ([]personMatch, error) {
	if displayName == "" {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, display_name FROM persons WHERE display_name = ? COLLATE NOCASE`, displayName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPersonMatches(rows)
}

func scanPersonMatches(rows *sql.Rows) ([]personMatch, error) {
	var out []personMatch
	for rows.Next() {
		var m personMatch
		if err := rows.Scan(&m.id, &m.name); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
