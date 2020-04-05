package main

import "github.com/jmoiron/sqlx"

func init() { // nolint:gochecknoinits
	// Why even bother converting names? A single greppable string across all
	// your source code is better than any odd conversion scheme you could ever
	// come up with.
	sqlx.NameMapper = func(v string) string { return v }
}
