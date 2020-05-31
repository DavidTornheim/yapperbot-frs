package yapperconfig

import (
	"encoding/json"
	"log"

	"cgt.name/pkg/go-mwclient"
	"github.com/antonholmquist/jason"
	"github.com/mashedkeyboard/ybtools"
)

//
// Yapperbot-FRS, the Feedback Request Service bot for Wikipedia
// Copyright (C) 2020 Naypta

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
//

// OpeningJSON and ClosingJSON are the first and last lines of any JSON object generated by Yapperbot to store on-wiki.
const OpeningJSON string = `{"DO NOT TOUCH THIS PAGE":"This page is used internally by Yapperbot to make the Feedback Request Service work.",`
const ClosingJSON string = `}`

// SerializeToJSON takes in any serializable object and returns the serialized JSON string
func SerializeToJSON(serializable interface{}) string {
	serialized, err := json.Marshal(serializable)
	if err != nil {
		log.Fatal("Failed to serialize object, dumping what I was trying to serialize: ", serializable)
	}
	return string(serialized)
}

// LoadJSONFromPageID takes a mwclient and a pageID, then loads and deserializes the contained JSON.
// It returns the deserialised JSON in a jason.Object pointer.
func LoadJSONFromPageID(w *mwclient.Client, pageID string) *jason.Object {
	storedJSON, err := ybtools.FetchWikitext(w, pageID)
	if err != nil {
		log.Fatal("Failed to fetch sent count page with error ", err)
	}
	parsedJSON, err := jason.NewObjectFromBytes([]byte(storedJSON))
	if err != nil {
		log.Fatal("Failed to parse sent count JSON with error ", err)
	}
	return parsedJSON
}