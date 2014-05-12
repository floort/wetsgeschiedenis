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
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
)

const DateFmt = "02-01-2006"

type BWBstatistics struct {
	ID           string
	Title        string
	NumSnapshots uint32
}

type StatusPage struct {
	TotalSnapshots uint32
	BWBStats       []BWBstatistics
}

type SinglePage struct {
	Title         string
	BWBID         string
	Content       string
	ParsedContent WetgevingType
	PubDate       time.Time
	PrintableTime string
	Versions      []string
}

func SimpleTimeFmt(t time.Time) string {
	return t.Format(DateFmt)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	// Connect to the database
	db, err := sql.Open("postgres", connectionURL)
	if err != nil {
		log.Println(err)
		http.Error(w, "Database error", http.StatusInternalServerError)
	}
	defer db.Close()

	// Lookup snapshot statistics
	status := new(StatusPage)
	rows, err := db.Query(`SELECT bwb_snapshots.bwbid, COUNT(bwb_snapshots.pubdate), titel 
						FROM bwb_documents, bwb_snapshots 
						WHERE bwb_documents.bwbid=bwb_snapshots.bwbid 
						GROUP BY bwb_snapshots.bwbid, titel 
						ORDER BY COUNT(bwb_snapshots.pubdate) DESC;`)
	if err != nil {
		log.Println(err)
		http.Error(w, "Database error", http.StatusInternalServerError)
	}
	for rows.Next() {
		var bwbid, titel string
		var count uint32
		if err := rows.Scan(&bwbid, &count, &titel); err != nil {
			log.Println(err)
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		stats := BWBstatistics{bwbid, titel, count}
		status.BWBStats = append(status.BWBStats, stats)
		status.TotalSnapshots += count
	}
	if err := rows.Err(); err != nil {
		log.Println(err)
		http.Error(w, "Database error", http.StatusInternalServerError)
	}
	t, _ := template.ParseFiles("status.html")
	t.Execute(w, status)
}

func singleHandler(w http.ResponseWriter, r *http.Request) {
	// Connect to the database
	db, err := sql.Open("postgres", connectionURL)
	if err != nil {
		log.Println(err)
		http.Error(w, "Database error", http.StatusInternalServerError)
	}
	defer db.Close()
	// Path should be of form "/single/[bwbid]/[date]/"
	path := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	var bwbid, date string
	if len(path) == 2 { // Display latest version when date is not specified
		bwbid = path[1]
	} else if len(path) == 3 {
		bwbid = path[1]
		date = path[2]
	} else {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	page := SinglePage{BWBID: bwbid}
	if date == "" {
		err = db.QueryRow("SELECT content, pubdate, (SELECT titel FROM bwb_documents WHERE bwbid=$1) FROM bwb_snapshots WHERE bwbid=$1 ORDER BY pubdate DESC LIMIT 1", bwbid).Scan(&page.Content, &page.PubDate, &page.Title)
	} else {
		err = db.QueryRow("SELECT content, pubdate, (SELECT titel FROM bwb_documents WHERE bwbid=$1) FROM bwb_snapshots WHERE bwbid=$1 AND pubdate <= $2 ORDER BY pubdate DESC LIMIT 1", bwbid, date).Scan(&page.Content, &page.PubDate, &page.Title)
	}
	if err == sql.ErrNoRows {
		// Document not found
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Println(err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	page.PrintableTime = SimpleTimeFmt(page.PubDate)
	page.ParsedContent, err = ParseBWB(page.Content)
	if err != nil {
		log.Println(err)
		http.Error(w, "Parse error", http.StatusInternalServerError)
		return
	}
	log.Printf("%#v\n", page.ParsedContent)
	rows, err := db.Query(`SELECT pubdate FROM bwb_snapshots WHERE bwbid=$1 ORDER BY pubdate ASC;`, bwbid)
	if err != nil {
		log.Println(err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	for rows.Next() {
		var pubdate time.Time
		if err := rows.Scan(&pubdate); err != nil {
			log.Println(err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		page.Versions = append(page.Versions, SimpleTimeFmt(pubdate))
	}
	t, err := template.ParseFiles("single.html")
	if err != nil {
		log.Println(err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
	t.Execute(w, page)
}

func startWebServer() {
	http.HandleFunc("/status/", statusHandler)
	http.HandleFunc("/single/", singleHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
