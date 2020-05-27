package frslist

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"cgt.name/pkg/go-mwclient"
	"cgt.name/pkg/go-mwclient/params"
	"github.com/mashedkeyboard/ybtools"

	// needs to be blank-imported to make the driver work
	_ "github.com/go-sql-driver/mysql"
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

var checkedUsers map[string]bool
var usersToRemove []string
var regexEscapeNeeded *regexp.Regexp

const queryTemplate string = `SELECT actor_user.actor_name FROM revision_userindex
INNER JOIN actor_user ON actor_user.actor_name = ? AND actor_id = rev_actor
WHERE rev_timestamp > ? LIMIT 1;`

func init() {
	regexEscapeNeeded = regexp.MustCompile(`[.*+?^${}()|[\]\\]`)
	checkedUsers = map[string]bool{}
}

func pruneUsersFromList(text string, w *mwclient.Client, dbserver, dbuser, dbpassword, db string) {
	var regexBuilder strings.Builder
	var ignoredOutputFromQueryRow string

	conn, err := sql.Open("mysql", dbuser+":"+dbpassword+"@tcp("+dbserver+")/"+db)
	if err != nil {
		log.Fatal("DSN invalid with error ", err)
	}
	if err := conn.Ping(); err != nil {
		log.Fatal(err)
	}
	query, err := conn.Prepare(queryTemplate)
	if err != nil {
		log.Fatal(err)
	}
	defer query.Close()

	var editsSinceStamp string = time.Now().AddDate(-3, 0, 0).Format("20060102210405") // format in line with https://www.mediawiki.org/wiki/Manual:Timestamp

	for _, header := range list {
		for _, user := range header {
			if checkedUsers[user.Username] {
				continue
			}

			checkedUsers[user.Username] = true
			// We have no use whatsoever for the output of this, we just want to see if it errors.
			// That being said, Scan() doesn't let us just pass nothing, so we have to have the
			// slight pain of having a stupid additional variable.
			err := query.QueryRow(user.Username, editsSinceStamp).Scan(&ignoredOutputFromQueryRow)
			if err != nil {
				if err == sql.ErrNoRows {
					log.Println("Queuing", user.Username, "for pruning")
					usersToRemove = append(usersToRemove, user.Username)
				} else {
					log.Fatal("Failed when querying DB with error ", err)
				}
			}
		}
	}

	regexBuilder.WriteString(`(?i)\* ?{{frs user\|(`) // Write the start of the regex
	regexUsersToRemove := make([]string, len(usersToRemove))
	for i, user := range usersToRemove {
		regexUsersToRemove[i] = regexEscapeNeeded.ReplaceAllString(user, `\$0`)
	}
	regexBuilder.WriteString(strings.Join(regexUsersToRemove, "|"))
	regexBuilder.WriteString(`)\|(\d+)}}\n?`)

	fmt.Println("=========================================================================================")
	fmt.Println("======================================= WIKITEXT ========================================")
	fmt.Println("=========================================================================================")
	fmt.Println(regexp.MustCompile(regexBuilder.String()).ReplaceAllString(text, ""))
	fmt.Println("=========================================================================================")
	fmt.Println("===================================== REMOVED USERS =====================================")
	fmt.Println("=========================================================================================")
	fmt.Println(strings.Join(usersToRemove, "\n"))
}

func checkUsersForPruning(w *mwclient.Client, users []string, userLookup map[string]bool) []string {
	log.Println("Calling mw")
	query := w.NewQuery(params.Values{
		"action":  "query",
		"list":    "usercontribs",
		"uclimit": "1",
		"ucend":   time.Now().AddDate(-3, 0, 0).Format(time.RFC3339), // RFC 3339 representation of the timestamp 3 years ago
		"ucuser":  strings.Join(users, "|"),
	})
	log.Println("users:", strings.Join(users, "|"))

	for query.Next() {
		// log.Println("Query iteration")
		contribsArray := ybtools.GetThingFromQuery(query.Resp(), "usercontribs")
		// log.Println(query.Resp())
		for _, contrib := range contribsArray {
			username, err := contrib.GetString("user")
			if err != nil {
				continue
			}
			// remove the user from the list, now that we know that they have contributed
			// since the deadline
			delete(userLookup, username)
		}
	}
	if query.Err() != nil {
		log.Fatal("Errored while querying user list with error: ", query.Err())
	}

	// now reformulate the users array with the remaining users, who are all the users
	// who haven't contributed since the deadline
	users = make([]string, len(userLookup))
	i := 0
	for username := range userLookup {
		users[i] = username
		i++
	}

	return users
}