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
	"flag"
	"log"
)

// Commandline options
var connectionURL string
var resetDatabaseFlag bool
var loadBWBList bool
var syncBWBSnapshots bool
var helpFlag bool

func init() {
	flag.BoolVar(&helpFlag, "help", false, "Show help text.")
	flag.StringVar(&connectionURL, "connection", "", "postgres://user:password@host/database")
	flag.BoolVar(&resetDatabaseFlag, "reset", false, "Reset the database before running.")
	flag.BoolVar(&syncBWBSnapshots, "sync", false, "Keep syncing the BWB snapshots.")
	flag.BoolVar(&loadBWBList, "loadbwb", false, "Load the BWBIdList")
}

func main() {
	flag.Parse()

	// Print help information and exit
	if helpFlag {
		flag.PrintDefaults()
		return
	}

	// Exit if no connection string is given
	if connectionURL == "" {
		flag.PrintDefaults()
		return
	}

	// Reset the database
	if resetDatabaseFlag {
		log.Println("Resetting database.")
		err := ResetDatabase(connectionURL)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Load the BWBIdList
	if loadBWBList {
		log.Println("Loading BWBIdList.")
		err := StoreBWBIdList(connectionURL)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Sync the BWB snapshots
	if syncBWBSnapshots {
		go keepBWBSynced(connectionURL)
	}

	// Start the webserver
	startWebServer()

}
