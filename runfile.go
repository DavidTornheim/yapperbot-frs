package main

// 2024-05-10	DAT	Use DavidTornheim/ytbools rather than Mashedkeyboard's version

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
	"io/ioutil"
	"strings"

//	"github.com/mashedkeyboard/ybtools/v2"
	"github.com/DavidTornheim/ybtools"
	"github.com/metal3d/go-slugify"
)

// loadFromRunfile takes a category name, and loads the applicable .frsrunfile file, if there is one.
// The .frsrunfile file stores the timestamp of the last processed page in the category, and its page ID.
// This is used to track our progress through the category, and prevent us from sending messages about the
// same page twice.
// The function returns the timestamp and the page ID, both as strings.
func loadFromRunfile(category string) (timestamp, pageID string) {
	var startRunfile []byte
	// runfile stores the last categorisation timestamp
	runfileName := slugify.Marshal(category) + ".frsrunfile"
	startRunfile, err := ioutil.ReadFile(runfileName)
	if err != nil {
		// the runfile doesn't exist probably, try creating it
		err := ioutil.WriteFile(runfileName, []byte(""), 0644)
		if err != nil {
			ybtools.PanicErr("Failed to create runfile with error ", err)
		}
		return "", ""
	}
	splitStartRunfile := strings.SplitN(string(startRunfile), ";", 2)

	switch len(splitStartRunfile) {
	case 1:
		splitStartRunfile = append(splitStartRunfile, "")
	case 2:
		break
	default:
		ybtools.PanicErr("Corrupt runfile for category ", category)
	}

	return splitStartRunfile[0], splitStartRunfile[1]
}
