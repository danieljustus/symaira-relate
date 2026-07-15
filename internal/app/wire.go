package app

import (
	"database/sql"

	"github.com/danieljustus/symaira-relate/internal/xdg"
)

// wire constructs the App and its service container. Later issues extend
// this function as new domain services land; it stays the single place
// that assembles the dependency graph.
func wire(paths xdg.Paths, db *sql.DB) *App {
	return &App{
		Paths: paths,
		DB:    db,
	}
}
