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
	"log"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"yapperbot-frs/src/yapperconfig"

	"cgt.name/pkg/go-mwclient"
	"cgt.name/pkg/go-mwclient/params"
//	"github.com/mashedkeyboard/ybtools/v2"
	"github.com/DavidTornheim/ybtools"
)

// list is the overall list of FRSUsers mapped to their headers.
var list map[string][]*FRSUser

// listHeaders is just a boring old list of headers, we have a getter for it later.
// It's used to keep track of which headers we have.
var listHeaders []string

// frsWeightedUser extends FRSUser to add a weighting component. It's only used within frslist.
type frsWeightedUser struct {
	*FRSUser
	// weight will represent our probability for this user to be selected
	weight float64
	// _hasAllHeaderChecked is a simple boolean check to make sure we don't halve the probability twice
	_hasAllHeaderChecked bool
}

// checkWeightForAllHeader takes a string representation of the "All [type]s" header, and checks whether
// the frsWeightedUser is contained within that header. If it is, it will halve the user's weighting, to
// encourage users subscribed to specific categories to be selected.
func (u *frsWeightedUser) checkWeightForAllHeader(allHeader string) {
	if !u._hasAllHeaderChecked {
		// if the all header is set, give those users half the probability of receiving the message.
		// we should try and make sure our messages are being sent to specific categories more of the time,
		// but we should still make sure users under the all headers receive messages.
		// this needs to be done here so that they are ordered correctly; as we're later inverting the probabilities,
		// the weight also has to be doubled, not halved, counterintuitively
		if allHeader != "" && u.Header == allHeader {
			u.weight = u.weight * 2
		}
		u._hasAllHeaderChecked = true
	}
}

// sentCount maps headers down to users, and then users down to the number of messages they've received this month.
var sentCount map[string]map[string]uint16 // {header: {user: count sent}}
// sentCountMux is a simple mutex to make sure that, if we ever add goroutines, we don't start overwriting
// SentCount simultaneously.
var sentCountMux sync.Mutex

// listParserRegex looks at the Feedback Request Service list, and finds each header and its users.
var listParserRegex *regexp.Regexp

// userParserRegex looks over the contents of a FRS list header, and finds each user within the header.
var userParserRegex *regexp.Regexp

// randomGenerator is our random number generator for this function, separated so we can separately seed it.
var randomGenerator *rand.Rand

func init() {
	// This regex matches on the Feedback Request Service list.
	// The first group matches the header (minus the ===s)
	// The second matches all of the contents underneath that header
	listParserRegex = regexp.MustCompile(`===(.*?)===\n((?i:\*\s*{{frs user.*?}}\n*)+)`)

	// This regex matches each user individually in a section of the FRS list.
	// The first group matches the user name
	// The second group matches the requested limit
	userParserRegex = regexp.MustCompile(`(?i){{frs user\|([^|]*)(?:\|(\d+))?}}`)

	randomGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))

	list = map[string][]*FRSUser{}
	sentCount = map[string]map[string]uint16{}
}

// Populate sets up the FRSList list as appropriate for the start of the program.
func Populate() {
	populateFrsList()
	populateSentCount()
}

// GetListHeaders is a simple getter for listHeaders
func GetListHeaders() []string {
	return listHeaders
}

// GetUsersFromHeaders takes a list of headers and an integer number of users n, and returns a randomly selected portion of the users
// from the headers, with a total size of maximum n. It won't pick the same user twice, and weights the users based on how far through their limit
// they are, in an attempt to spread things out a bit. It may pick less than n if there are less users available.
func GetUsersFromHeaders(headers []string, allHeader string, n int) (returnedUsers []*FRSUser) {
	var weightedUsers []*frsWeightedUser
	// used to check in o(1) time whether we've already
	// selected this user, just on another header
	var usersSelected = map[string]bool{}

	// start our returnedUsers off with a zero-length slice of cap n
	returnedUsers = make([]*FRSUser, 0, n)

	// unlimitedUsers stores all of our users who have no limit set.
	// calculatedWeights stores all of the weights that we have otherwise
	// calculated.
	// these two are used to later calculate the median of the calculated weights,
	// and set each of the unlimited users to be weighted the same as the median.
	var unlimitedUsers []*frsWeightedUser
	var calculatedWeights []float64

	// Get a list of all the eligible users in the header
	for _, header := range headers {
		for _, user := range list[header] {
			if !user.ExceedsLimit() {
				var weight float64
				if user.Limited {
					if user.GetCount() == 0 {
						// if the user has not been sent anything, prioritise them for
						// sending; this seems a reasonable way of weighting
						weight = 0
					} else {
						// the user has been sent something, and has a limit set.
						// weight them on the basis of their relative position
						// within their limit. this means users with a higher limit
						// will be more likely to receive more messages than users
						// with a lower limit, and vice versa;
						// however, it also keeps users with high limits from receiving
						// all the messages, when other users are lacking anything sent.
						weight = float64(user.GetCount()) / float64(user.Limit)
					}

					// shift each weight forward by 1 to avoid issues with dividing by zero, and to avoid
					// zero probabilities; we want users with no current sent messages to be top-priority,
					// definitely not zero priority
					weight = weight + 1

					// check the user for inclusion in the allHeader, halve their probability if they are,
					// and then append them to the list of users
					wUser := &frsWeightedUser{FRSUser: user, weight: weight}
					wUser.checkWeightForAllHeader(allHeader)
					weightedUsers = append(weightedUsers, wUser)
				} else {
					// if the user has no limit set, add them to unlimitedUsers as well as weightedUsers;
					// we'll set their weight to the median weight later
					wUser := &frsWeightedUser{FRSUser: user}
					unlimitedUsers = append(unlimitedUsers, wUser)
					weightedUsers = append(weightedUsers, wUser)
				}
			}
		}
	}

	// If there are any users in the header who have no limits set, set their weighting to the median of the limited users
	if len(unlimitedUsers) > 0 {
		median, hasMedian := calculateMedian(calculatedWeights)
		if !hasMedian {
			// if all users are unlimited, then just set their weights to 1 -
			// they're all the same anyway then, so it doesn't make a difference
			median = 1
		}
		for _, user := range unlimitedUsers {
			user.weight = median
			// even for unlimited users, we want to give people in specific category headers more of a chance,
			// so we should still run the AllHeader weight check here
			user.checkWeightForAllHeader(allHeader)
		}
	}

	// Check if we actually have an opportunity to randomly select at all here
	if len(weightedUsers) <= n {
		// very small list, or very large n
		// just give the entire list
		for _, wuser := range weightedUsers {
			returnedUsers = append(returnedUsers, wuser.FRSUser)
		}
		return
	}

	// Sort the list into increasing order of sent count this month
	sort.Slice(weightedUsers, func(i, j int) bool {
		return weightedUsers[i].weight < weightedUsers[j].weight
	})

	// Calculate cumulative sent counts for the users
	var cumulativeSentCount float64

	for _, user := range weightedUsers {
		// take the reciprocal of the weight, and use it cumulatively.
		// we do this here to ensure that our sent counts are used
		// as a decentive for sending new messages.
		cumulativeSentCount += 1 / user.weight
		weight := float64(cumulativeSentCount)

		user.weight = weight
	}

	// Reseed to avoid getting the same or similar sequences every time
	// when we have large batches
	randomGenerator.Seed(time.Now().UnixNano())

	// Select a random user each time based on our weights
	var i = 0
	for i < n {
		// adjust our random value to be within our bounds - going up to the
		// final weight as a maximum value possible
		randomValue := randomGenerator.Float64() * weightedUsers[len(weightedUsers)-1].weight

		selectedUserIndex := sort.Search(len(weightedUsers), func(i int) bool {
			// find the smallest weight user whose weight is greater than our random selection
			return weightedUsers[i].weight > randomValue
		})

		if selectedUserIndex > len(weightedUsers)-1 {
			// the selection index hasn't been found; this is probably because we've exhausted our search somehow
			// just return what we've got, and mark in the log that this happened - it shouldn't ever happen
			log.Println("WARNING: Exhausted search space for selectedUserIndex, returning what we can get: asked for", n, "and got", len(returnedUsers))
			return
		}

		// make sure we haven't already picked this user
		if !usersSelected[weightedUsers[selectedUserIndex].Username] {
			// assuming we haven't, add them to our list...
			returnedUsers = append(returnedUsers, weightedUsers[selectedUserIndex].FRSUser)
			// mark them as picked...
			usersSelected[weightedUsers[selectedUserIndex].Username] = true
			// and increase the number of users we've picked up to n
			i++
		}

		// make sure we're not keeping the same users around, potentially selecting them
		// multiple times
		if selectedUserIndex+1 == len(weightedUsers) {
			// this is the end of the slice; just chop the end off
			weightedUsers = weightedUsers[:selectedUserIndex]
		} else {
			// we need to chop this element, and only this element, out
			weightedUsers = append(weightedUsers[:selectedUserIndex], weightedUsers[selectedUserIndex+1:]...)
		}
	}

	return
}

// FinishRun for now just calls saveSentCounts, but could do something else too in future
func FinishRun(w *mwclient.Client) {
	saveSentCounts(w)
}

// populateFrsList fetches the wikitext of the FRS subscriptions page, and processes the page against
// the listParserRegex and userParserRegex. Together, those parse the headers in the file, along with
// the users that are subscribed, turning them into FRSUser objects and storing them in `list`.
func populateFrsList() string {
	text, err := ybtools.FetchWikitext(yapperconfig.Config.FRSPageID)
	if err != nil {
		ybtools.PanicErr("Failed to fetch and parse FRS page with error ", err)
	}

	for _, match := range listParserRegex.FindAllStringSubmatch(text, -1) {
		// match is [entire match, header, contents]
		var users []*FRSUser
		for _, usermatched := range userParserRegex.FindAllStringSubmatch(match[2], -1) {
			// usermatched is [entire match, user name, requested limit]
			if usermatched[2] == "0" {
				// The user has explicitly requested no limit
				// we only need to set the username; bool default is false, and numeric default is zero
				users = append(users, &FRSUser{Username: usermatched[1], Header: match[1]})
			} else if usermatched[2] != "" {
				// The user has a limit set
				if limit, err := strconv.ParseInt(usermatched[2], 10, 16); err == nil {
					users = append(users, &FRSUser{Username: usermatched[1], Header: match[1], Limit: uint16(limit), Limited: true})
				} else {
					log.Println("User", usermatched[1], "has an invalid limit of", usermatched[2], "so ignoring")
				}
			} else {
				// The user does not have a set limit
				// Use the default value of 1
				users = append(users, &FRSUser{Username: usermatched[1], Header: match[1], Limit: 1, Limited: true})
			}
		}
		list[match[1]] = users
	}

	listHeaders = make([]string, len(list))
	i := 0
	for header := range list {
		listHeaders[i] = header
		i++
	}

	return text
}

// populateSentCount fetches the SentCount page, and checks it's of the right month.
// If it's a previous month, then it just leaves the `sentCount` map blank; if it's
// the same month listed on the JSON file, it will parse the JSON and load it into `sentCount`.
func populateSentCount() {
	// This is stored on the page with ID sentCountPageID.
	// It is made up of something that looks like this:
	// {"month": "2020-05", "headers": {"category": {"username": 8}}}
	// where username had been sent 8 messages in the month of May 2020 and the header "category".
	parsedJSON := ybtools.LoadJSONFromPageID(yapperconfig.Config.SentCountPageID)

	contentMonth, _ := parsedJSON.GetString("month")
	// yes, really, you have to specify time formats with a specific time in Go
	// *rolls eyes*
	// https://golang.org/pkg/time/#Time.Format
	if contentMonth != time.Now().Format("2006-01") {
		log.Println("contentMonth is not the current month, so data resets!")
	} else {
		sentCount = deserializeSentCount(parsedJSON)
	}
}

// saveSentCounts serializes our `sentCount` map into JSON, so we can save it on-wiki
// and load it again when we need to for the next run.
func saveSentCounts(w *mwclient.Client) {
	var sentCountJSONBuilder strings.Builder
	sentCountJSONBuilder.WriteString(yapperconfig.OpeningJSON)
	sentCountJSONBuilder.WriteString(`"month":"`)
	sentCountJSONBuilder.WriteString(time.Now().Format("2006-01"))
	sentCountJSONBuilder.WriteString(`","headers":`)
	sentCountJSONBuilder.WriteString(ybtools.SerializeToJSON(sentCount))
	sentCountJSONBuilder.WriteString(yapperconfig.ClosingJSON)

	// this is in userspace, and it's really desperately necessary - do not count this for edit limiting
	// for the same reason, we have no maxlag wait - we need this to run under all circumstances, to ensure
	// that people's limits are respected
	ybtools.NoMaxlagDo(func() (err error) {
		err = w.Edit(params.Values{
			"pageid":   yapperconfig.Config.SentCountPageID,
			"summary":  "FRS run complete, updating sentcounts",
			"notminor": "true",
			"bot":      "true",
			"text":     sentCountJSONBuilder.String(),
		})
		if err == nil {
			log.Println("Successfully updated sentcounts")
		} else {
			if err.Error() == "edit successful, but did not change page" {
				log.Println("WARNING: Successfully updated sentcounts, but they didn't change - if anything was done this session, something is wrong!")
			} else {
				ybtools.PanicErr("Failed to update sentcounts with error ", err)
			}
		}
		return
	}, w)
}

// calculateMedian takes a slice of float64s and returns the median if there is one, and a bool indicating if a median
// could be calculated (i.e. if the given slice has a length greater than zero).
func calculateMedian(calculatedWeights []float64) (float64, bool) {
	if len(calculatedWeights) > 0 {
		sort.Float64s(calculatedWeights)
		middleIndex := len(calculatedWeights) / 2
		if len(calculatedWeights)%2 == 0 {
			return (calculatedWeights[middleIndex-1] + calculatedWeights[middleIndex]) / 2, true
		}
		return calculatedWeights[middleIndex], true
	}
	return 0, false
}
