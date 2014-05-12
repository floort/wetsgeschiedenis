package main

/*
 * Copyright (c) 2014 Floor Terra <floort@gmail.com>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

import (
	"database/sql"
	_ "github.com/lib/pq"
)

func ResetDatabase(connectionURL string) error {
	// Connect to the database
	db, err := sql.Open("postgres", connectionURL)
	if err != nil {
		return err
	}
	defer db.Close()

	// Create all tables
	_, err = db.Query(`
		-- Drop all Tables
		DROP TABLE IF EXISTS bwb_snapshots;
		DROP TABLE IF EXISTS bwb_documents;
		-- Recreate all tables
		CREATE TABLE bwb_documents
		(
			bwbid			character varying(32) 	PRIMARY KEY,
			officieletitel	character varying(2048)	NOT NULL,
			titel			character varying(1024)	NOT NULL,
			status			character varying(64)	NOT NULL,
			regelingsoort	character varying(64)	NOT NULL,
			startdatum		date					NULL,
			vervaldatum		date					NULL
		);
		CREATE TABLE bwb_snapshots
		(
			hash	character varying(128)	PRIMARY KEY,
			bwbid	character varying(32)	REFERENCES bwb_documents(bwbid),
			pubdate	date					NOT NULL,
			content	text					NOT NULL
		);
		CREATE INDEX pubdate_idx ON bwb_snapshots(pubdate);
		`)
	if err != nil {
		return err
	}
	return nil
}
