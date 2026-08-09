// Package ows — SQLite driver registration.
package ows

import (
	"database/sql"
	_ "modernc.org/sqlite" // register sqlite driver (pure Go, CGO-free)
)

// DriverName is the SQLite driver name for sql.Open.
const DriverName = "sqlite"

func init() {
	// Verify driver is registered
	db, err := sql.Open(DriverName, ":memory:")
	if err == nil {
		db.Close()
	}
}
