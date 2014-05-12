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
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/xml"
	"fmt"
	_ "github.com/lib/pq"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

const BWBidListURL = "http://wetten.overheid.nl/BWBIdService/BWBIdList.xml.zip"

type BWBIdServiceResultaat struct {
	XMLName           xml.Name       `xml:"BWBIdServiceResultaat"`
	GegenereerdOp     string         `xml:"GegenereerdOp"`
	RegelingInfoLijst []RegelingInfo `xml:"RegelingInfoLijst>RegelingInfo"`
}

type RegelingInfo struct {
	XMLName                xml.Name `xml:"RegelingInfo"`
	BWBId                  string   `xml:"BWBId"`
	DatumLaatsteWijziging  string   `xml:"DatumLaatsteWijziging"`
	VervalDatum            string   `xml:"VervalDatum"`
	OfficieleTitel         string   `xml:"OfficieleTitel"`
	Titel                  string   `xml:"CiteertitelLijst>Citeertitel>titel"`
	Status                 string   `xml:"CiteertitelLijst>Citeertitel>status"`
	InwerkingtredingsDatum string   `xml:"CiteertitelLijst>Citeertitel>InwerkingtredingsDatum"`
	RegelingSoort          string   `xml:"RegelingSoort"`
}

type BWBDocument struct {
	Id             string
	OfficieleTitel string
	Titel          string
	Status         string
	RegelingSoort  string
	StartDatum     time.Time
	VervalDatum    time.Time
}

func generateBWBDocuments(db *sql.DB) chan BWBDocument {
	// Return a chan through which all BWBDocuments are fed.
	ch := make(chan BWBDocument, 16)
	go func() { // Get all BWBDocuments in a goroutine
		rows, err := db.Query("SELECT bwbid, officieletitel, titel, status, regelingsoort, startdatum, vervaldatum FROM bwb_documents")
		if err != nil {
			// Log error, close the chanel and stop execution on a database error
			log.Println(err)
			close(ch)
			return
		}
		for rows.Next() {
			// Insert each row from bwb_documents in a BWBDocument struct
			var bwbid, officieletitel, titel, status, regelingsoort, startdatum, vervaldatum string
			if err := rows.Scan(&bwbid, &officieletitel, &titel, &status, &regelingsoort, &startdatum, &vervaldatum); err != nil {
				log.Println(err)
				continue
			}

			doc := BWBDocument{bwbid, officieletitel, titel, status, regelingsoort, time.Time{}, time.Time{}}
			// Parse the date objects
			doc.StartDatum, err = time.Parse("02-01-2006", startdatum)
			if err != nil {
				doc.StartDatum = time.Time{}
			}
			doc.VervalDatum, err = time.Parse("02-01-2006", vervaldatum)
			if err != nil {
				doc.VervalDatum = time.Time{}
			}
			ch <- doc // Return the document through the channel
		}
		close(ch) // Close the chanel when done
	}()
	return ch
}

func DownloadBWBIdList() (*BWBIdServiceResultaat, error) {
	log.Println("Downloading BWBIdList zip.")
	bwblist := new(BWBIdServiceResultaat)
	resp, err := http.Get(BWBidListURL)
	if err != nil {
		return bwblist, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return bwblist, err
	}
	defer resp.Body.Close()
	bodyreader := bytes.NewReader(body)

	zipfile, err := zip.NewReader(bodyreader, int64(bodyreader.Len()))
	if err != nil {
		return bwblist, err
	}

	for _, f := range zipfile.File {
		if f.Name == "BWBIdList.xml" {
			reader, err := f.Open()
			if err != nil {
				return bwblist, err
			}
			decoder := xml.NewDecoder(reader)
			bwblist := new(BWBIdServiceResultaat)
			err = decoder.Decode(bwblist)
			if err != nil {
				return bwblist, err
			}
			return bwblist, nil
		}
	}
	return bwblist, nil
}

func StoreBWBIdList(connectionURL string) error {
	// Connect to the database
	db, err := sql.Open("postgres", connectionURL)
	if err != nil {
		return err
	}
	defer db.Close()
	bwblist, err := DownloadBWBIdList()
	if err != nil {
		return err
	}
	log.Println("Filling bwb_documents table")
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT INTO bwb_documents " +
		"(bwbid, officieletitel, titel, status, regelingsoort, startdatum, vervaldatum) " +
		"VALUES ($1, $2, $3, $4, $5, $6, $7)")
	if err != nil {
		return err
	}

	for i := range bwblist.RegelingInfoLijst {
		var err error
		if bwblist.RegelingInfoLijst[i].InwerkingtredingsDatum == "" {
			if bwblist.RegelingInfoLijst[i].VervalDatum == "" {
				_, err = stmt.Exec(bwblist.RegelingInfoLijst[i].BWBId,
					bwblist.RegelingInfoLijst[i].OfficieleTitel,
					bwblist.RegelingInfoLijst[i].Titel,
					bwblist.RegelingInfoLijst[i].Status,
					bwblist.RegelingInfoLijst[i].RegelingSoort,
					nil,
					nil)
			} else {
				_, err = stmt.Exec(bwblist.RegelingInfoLijst[i].BWBId,
					bwblist.RegelingInfoLijst[i].OfficieleTitel,
					bwblist.RegelingInfoLijst[i].Titel,
					bwblist.RegelingInfoLijst[i].Status,
					bwblist.RegelingInfoLijst[i].RegelingSoort,
					nil,
					bwblist.RegelingInfoLijst[i].VervalDatum)
			}
		} else {
			if bwblist.RegelingInfoLijst[i].VervalDatum == "" {
				_, err = stmt.Exec(bwblist.RegelingInfoLijst[i].BWBId,
					bwblist.RegelingInfoLijst[i].OfficieleTitel,
					bwblist.RegelingInfoLijst[i].Titel,
					bwblist.RegelingInfoLijst[i].Status,
					bwblist.RegelingInfoLijst[i].RegelingSoort,
					bwblist.RegelingInfoLijst[i].InwerkingtredingsDatum,
					nil)
			} else {
				_, err = stmt.Exec(bwblist.RegelingInfoLijst[i].BWBId,
					bwblist.RegelingInfoLijst[i].OfficieleTitel,
					bwblist.RegelingInfoLijst[i].Titel,
					bwblist.RegelingInfoLijst[i].Status,
					bwblist.RegelingInfoLijst[i].RegelingSoort,
					bwblist.RegelingInfoLijst[i].InwerkingtredingsDatum,
					bwblist.RegelingInfoLijst[i].VervalDatum)
			}
		}
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func getbwburl(bwbid string, date time.Time) string {
	v := url.Values{}
	v.Set("regelingID", bwbid)
	v.Set("geldigheidsdatum", fmt.Sprintf("%02d-%02d-%04d", date.Day(), date.Month(), date.Year()))
	return "http://wetten.overheid.nl/xml.php?" + v.Encode()
}

func keepBWBSynced(connectionURL string) {

	// Connect to the database
	db, err := sql.Open("postgres", connectionURL)
	if err != nil {
		log.Println(err)
		return
	}

	bwbs := make(chan string)
	go ConcurrentSyncer(db, 8, bwbs)
	rows, err := db.Query("SELECT bwbid FROM bwb_documents WHERE regelingsoort='wet'")
	if err != nil {
		log.Println(err)
		return
	}
	for rows.Next() {
		var bwbid string
		if err := rows.Scan(&bwbid); err != nil {
			log.Println(err)
		}
		/*if bwbid == "BWBR0011468" || bwbid == "BWBR0013409" || bwbid == "BWBR0009950" || bwbid == "BWBR0001854" || bwbid == "BWBR0005537" || bwbid == "BWBR0001840" || bwbid == "BWBR0001886" || bwbid == "BWBR0024282" || bwbid == "BWBR0020368‎" {
			go LoadBWBSnapshots(db, bwbid)
		}*/
		bwbs <- bwbid
	}
	if err := rows.Err(); err != nil {
		log.Println(err)
	}

}

func ConcurrentSyncer(db *sql.DB, n int, bwbs chan string) {
	done := make(chan bool)
	for i := 0; i < n; i++ {
		go LoadBWBSnapshots(db, <-bwbs, done)
	}
	for bwbid := range bwbs {
		_ = <-done
		go LoadBWBSnapshots(db, bwbid, done)
	}

}

func LoadBWBSnapshots(db *sql.DB, bwbid string, done chan bool) {
	// Get the starting date to scrape from
	var date time.Time
	err := db.QueryRow("SELECT pubdate FROM bwb_snapshots WHERE bwbid=$1 ORDER BY pubdate DESC LIMIT 1", bwbid).Scan(&date)
	if err == sql.ErrNoRows {
		// The fifth of may 2002 is the smallest date the API will return results for
		date = time.Date(2002, time.May, 1, 12, 0, 0, 0, time.UTC)
	} else if err != nil {
		log.Println(err)
	}
	if date.Before(time.Date(2002, time.May, 1, 12, 0, 0, 0, time.UTC)) {
		date = time.Date(2002, time.May, 1, 12, 0, 0, 0, time.UTC)
	}

	largestep := 2 * 31 * 24 * time.Hour // Two months
	smallstep := 24 * time.Hour          // One day
	timestep := largestep
	insertmode := false // Seeking for new versions with 1 month increments, insert with 1 day increments.
	for now := time.Now(); date.Before(now); date = date.Add(timestep) {
		url := getbwburl(bwbid, date)
		//log.Println(url)
		resp, err := http.Get(url)
		if err != nil {
			log.Println(err)
			break
		}
		if resp.StatusCode != 200 {
			log.Println(resp.Status)
			break
		}
		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			break
		}
		resp.Body.Close()
		h := sha256.New()
		io.WriteString(h, string(content))
		hash := fmt.Sprintf("%x", h.Sum(nil))
		var resulthash string
		err = db.QueryRow("SELECT hash FROM bwb_snapshots WHERE hash=$1", hash).Scan(&resulthash)
		switch {
		case err == sql.ErrNoRows:
			if insertmode {
				log.Println("NEW VERSION", bwbid, date)
				_, err = db.Exec(`INSERT INTO bwb_snapshots 
					(hash, bwbid, pubdate, content) 
					VALUES
					($1, $2, $3, $4);`, hash, bwbid, date, string(content))
				if err != nil {
					log.Println(err)
				}
				timestep = largestep
				insertmode = false
			} else {
				timestep = smallstep
				date.Add(-largestep)
				insertmode = true
			}
		case err != nil:
			log.Println(err)
		default:
			// Skip insertion if hash allready exists
			// log.Println(bwbid, date)
		}

	}
	log.Println("Sync of", bwbid, "complete.")
	done <- true
}
