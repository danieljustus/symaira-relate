package app

import (
	"database/sql"

	contactsvc "github.com/danieljustus/symaira-relate/internal/service/contact"
	relationshipsvc "github.com/danieljustus/symaira-relate/internal/service/relationship"
	securitysvc "github.com/danieljustus/symaira-relate/internal/service/security"
	"github.com/danieljustus/symaira-relate/internal/xdg"
)

// wire constructs the App and its service container. Later issues extend
// this function as new domain services land; it stays the single place
// that assembles the dependency graph.
func wire(paths xdg.Paths, db *sql.DB) *App {
	return &App{
		Paths:         paths,
		DB:            db,
		Contacts:      contactsvc.New(db),
		Relationships: relationshipsvc.New(db),
		Security:      securitysvc.New(db),
	}
}
