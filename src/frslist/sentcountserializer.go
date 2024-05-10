package frslist

// 2024-05-10	DAT	Use DavidTornheim/ytbools rather than mashedkeyboard's version

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
	"github.com/antonholmquist/jason"
//	"github.com/mashedkeyboard/ybtools/v2"
	"github.com/DavidTornheim/ybtools"
)

// deserializeSentCount takes a jason JSON object containing the SentCount.json
// information, and adds the sent counts into a map, mapping headers to usernames
// and usernames to numbers of messages sent. It returns this map as a
// map[string]map[string]uint16.
func deserializeSentCount(json *jason.Object) (sc map[string]map[string]uint16) {
	sc = map[string]map[string]uint16{} // initialise the map
	headers, err := json.GetObject("headers")
	if err != nil {
		ybtools.PanicErr("Failed to deserialize sent count headers, is the JSON invalid?")
	}
	for header, users := range headers.Map() {
		sc[header] = map[string]uint16{} // initialise the submap

		users, err := users.Object()
		if err != nil {
			ybtools.PanicErr("users wasn't an object, I can't handle this! the JSON seems invalid.")
		}
		for user, count := range users.Map() {
			count, err := count.Int64()
			if err != nil {
				ybtools.PanicErr("count wasn't a valid number, I can't handle this! the JSON seems invalid.")
			}
			// this never needs to be an int64, it's just that the library doesn't have arbitrary size int handling
			// converting it back down to uint16 at least saves a little memory in the long run, not that it hugely matters
			sc[header][user] = uint16(count)
		}
	}
	return
}
