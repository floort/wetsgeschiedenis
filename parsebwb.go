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
	"encoding/xml"
)

type WetgevingType struct {
	BWBID     string        `xml:"bwb-id,attr"`
	DTDVersie string        `xml:"dtdversie,attr"`
	ID        string        `xml:"id,attr"`
	Soort     string        `xml:"soort,attr"`
	Lang      string        `xml:"lang,attr"`
	Intitule  IntituleType  `xml:"intitule"`
	Aanhef    AanhefType    `xml:"wet-besluit>aanhef"`
	Artikelen []ArtikelType `xml:"wet-besluit>wettekst>titeldeel>artikel"`
}

type IntituleType struct {
	Id   string `xml:"id,attr"`
	Data string `xml:",chardata"`
}

type AanhefType struct {
	Wij         string   `xml:"wij"`
	Considerans []string `xml:"considerans>considerans.al"`
	Afkondiging []string `xml:"afkondiging>al"`
}

type ArtikelType struct {
	Status   string `xml:"status,attr"`
	Label    string `xml:"kop>label"`
	Nr       string `xml:"kop>nr"`
	Tekst    string `xml:"al"`
}

func (w *WetgevingType) Validate() bool {
	return true
}

func ParseBWB(document string) (WetgevingType, error) {
	wetgeving := WetgevingType{}
	err := xml.Unmarshal([]byte(document), &wetgeving)
	return wetgeving, err
}
