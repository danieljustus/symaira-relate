// Package app wires the storage layer to the domain services and is the
// behavior boundary the CLI (and, later, an MCP server) is allowed to
// depend on: no entry point may import internal/storage or a
// internal/service/* package directly. Pure data types in internal/domain
// are shared vocabulary and may be imported directly by entry points.
package app

import (
	"context"
	"database/sql"

	"github.com/danieljustus/symaira-relate/internal/errs"
	contactsvc "github.com/danieljustus/symaira-relate/internal/service/contact"
	relationshipsvc "github.com/danieljustus/symaira-relate/internal/service/relationship"
	"github.com/danieljustus/symaira-relate/internal/storage/sqlite"
	"github.com/danieljustus/symaira-relate/internal/xdg"
)

// App is the process-wide service container. Fields are populated
// incrementally as domain packages land; every field is safe to use
// concurrently.
type App struct {
	Paths xdg.Paths
	DB    *sql.DB

	Contacts      *contactsvc.Service
	Relationships *relationshipsvc.Service
}

// Aliases for the domain services' option types, so CLI commands can name
// them without importing internal/service/* directly.
type (
	ListPersonsOptions       = contactsvc.ListPersonsOptions
	ListOrganizationsOptions = contactsvc.ListOrganizationsOptions
	FollowUpFilter           = relationshipsvc.FollowUpFilter
)

const (
	FollowUpFilterAll      = relationshipsvc.FollowUpFilterAll
	FollowUpFilterOpen     = relationshipsvc.FollowUpFilterOpen
	FollowUpFilterOverdue  = relationshipsvc.FollowUpFilterOverdue
	FollowUpFilterUpcoming = relationshipsvc.FollowUpFilterUpcoming
)

// Open resolves XDG paths, ensures they exist, and opens the migrated
// database. Callers must Close the returned App.
func Open(ctx context.Context) (*App, error) {
	const op = "app.Open"

	paths, err := xdg.Resolve()
	if err != nil {
		return nil, errs.Internal(op, "failed to resolve XDG paths", err)
	}
	if err := paths.EnsureDirs(); err != nil {
		return nil, errs.Internal(op, "failed to create profile directories", err)
	}

	db, err := sqlite.Open(ctx, paths.DatabasePath())
	if err != nil {
		return nil, err
	}

	return wire(paths, db), nil
}

// OpenAt opens a database at an explicit path instead of the resolved XDG
// data directory (used for restoring a backup into a clean profile, or for
// tests).
func OpenAt(ctx context.Context, dbPath string, paths xdg.Paths) (*App, error) {
	db, err := sqlite.Open(ctx, dbPath)
	if err != nil {
		return nil, err
	}
	return wire(paths, db), nil
}

// OpenMemory opens an in-memory database, wired with the same service
// container as production. Intended for tests.
func OpenMemory(ctx context.Context) (*App, error) {
	db, err := sqlite.OpenMemory(ctx)
	if err != nil {
		return nil, err
	}
	return wire(xdg.Paths{}, db), nil
}

func (a *App) Close() error {
	if a.DB == nil {
		return nil
	}
	return a.DB.Close()
}
