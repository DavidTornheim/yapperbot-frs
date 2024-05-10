package rfc

// 5/10/24 -- DAT -- Use DavidTornheim/ytbools rather than mashedkeyboard's version
//		     add fmt module for print statement
// 4/5/24 -- DAT -- add print statements to understand code.

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

import (
	"reflect"
	"strings"
	"yapperbot-frs/src/yapperconfig"

	"cgt.name/pkg/go-mwclient"
	"cgt.name/pkg/go-mwclient/params"
//	"github.com/mashedkeyboard/ybtools/v2"
	"github.com/DavidTornheim/ybtools/v2"
	"fmt"
)

// doneRfcs maps found this session and completed/already-completed
// RfCs IDs to true (again, only used for o(n) lookups)
var doneRfcs map[string]bool = map[string]bool{}

// loadedRfcs maps *already-completed* RfC IDs to true.
// It only contains RfCs that were in the JSON at the start.
// We need this to be separate so we can keep ones out of doneRfcs that aren't in the category anymore
var loadedRfcs map[string]bool = map[string]bool{}

// MarkRfcsDone takes a series of RfC objects,
// and adds the RfCs to the list of completed RfCs.
func MarkRfcsDone(rfcsDone []RfC) {
// Debug only
	fmt.Println("routine:  MarkRfcsDone   from:  frs/src/rfc/done.go")
//
	for _, rfc := range rfcsDone {
		doneRfcs[rfc.ID] = true
	}
}

// LoadRfcsDone loads the RFCs that have already been marked as done into loadedRfcs.
// It needs to be called before the start of each session that includes an RfC lookup.
func LoadRfcsDone(w *mwclient.Client) {
	rfcsDoneJSON := ybtools.LoadJSONFromPageID(yapperconfig.Config.RFCsDonePageID)
	rfcsDoneList, err := rfcsDoneJSON.GetStringArray("rfcsdone")
	if err != nil {
		ybtools.PanicErr("rfcsdone not found in rfcsDoneJSON! the JSON looks corrupt.")
	}
	for _, rfcID := range rfcsDoneList {
		loadedRfcs[rfcID] = true
	}
}

// AlreadyDone takes an RfC ID and returns whether it's already included in either
// loadedRfcs or doneRfcs.
func AlreadyDone(rfcID string) bool {
	if loadedRfcs[rfcID] {
		return true
	}
	return doneRfcs[rfcID]
}

// SaveRfcsDone takes an mwclient, and serializes
// the doneRfcs map, before saving it on-wiki.
func SaveRfcsDone(w *mwclient.Client) {
	// Only update the list of RfCs done if it's actually changed -
	// i.e. if the list of doneRfcs is not deeply equal to the list of
	// loadedRfcs (bigger, smaller, changed in any way).
	if !reflect.DeepEqual(doneRfcs, loadedRfcs) {
		var rfcsDoneJSONBuilder strings.Builder
		var rfcsDoneSlice []string = []string{}

		for rfcid := range doneRfcs {
			rfcsDoneSlice = append(rfcsDoneSlice, rfcid)
		}

		rfcsDoneJSONBuilder.WriteString(yapperconfig.OpeningJSON)
		rfcsDoneJSONBuilder.WriteString(`"rfcsdone":`)
		rfcsDoneJSONBuilder.WriteString(ybtools.SerializeToJSON(rfcsDoneSlice))
		rfcsDoneJSONBuilder.WriteString(yapperconfig.ClosingJSON)

		// Updating this list must be done under all circumstances; we cannot
		// wait for maxlag here, it's important that this is kept valid and correct
		// to prevent us sending multiple messages.
		ybtools.NoMaxlagDo(func() (err error) {
			err = w.Edit(params.Values{
				"pageid":  yapperconfig.Config.RFCsDonePageID,
				"summary": "Updating list of completed RfCs",
				"bot":     "true",
				"text":    rfcsDoneJSONBuilder.String(),
			})
			if err != nil {
				ybtools.PanicErr("Failed to update RfC page ", yapperconfig.Config.RFCsDonePageID, " to list completed RfCs, with error ", err)
			}
			return
		}, w)
	}
}
