package main

import (
	"kaepora/internal/util"

	"github.com/jmoiron/sqlx"
)

// migrateSpoilerLogs compresses uncompressed spoiler logs
// TODO remove
func migrateSpoilerLogs() error {
	sqlx.NameMapper = func(v string) string { return v }
	db, err := sqlx.Connect("sqlite3", "./kaepora.db")
	if err != nil {
		return err
	}

	type match struct {
		ID         util.UUIDAsBlob
		SpoilerLog []byte
	}

	var matches []match
	if err := db.Select(&matches, `SELECT ID, SpoilerLog FROM Match`); err != nil {
		return err
	}

	stmt, err := db.Prepare(`UPDATE Match SET SpoilerLog = ? WHERE ID = ?`)
	if err != nil {
		return err
	}

	for _, v := range matches {
		// Skip compressed data, expected zlib header as canary. HACK
		if v.SpoilerLog[0] == 0x78 && v.SpoilerLog[1] == 0xDA {
			continue
		}

		log, err := util.NewZLIBBlob(v.SpoilerLog)
		if err != nil {
			return err
		}

		if _, err := stmt.Exec(log, v.ID); err != nil {
			return err
		}
	}

	if _, err := db.Exec("VACUUM"); err != nil {
		return err
	}

	return nil
}
