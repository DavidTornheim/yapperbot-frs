package main

// 4/5/24 -- DAT -- add comment about what the config file will be named.

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
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
	"yapperbot-frs/src/frslist"
	"yapperbot-frs/src/ga"
	"yapperbot-frs/src/messages"
	"yapperbot-frs/src/rfc"
	"yapperbot-frs/src/yapperconfig"

	"cgt.name/pkg/go-mwclient"
	"cgt.name/pkg/go-mwclient/params"
	"github.com/mashedkeyboard/ybtools/v2"
	"github.com/metal3d/go-slugify"
)

func init() {
	ybtools.SetupBot(ybtools.BotSettings{TaskName: "FRS", BotUser: "Yapperbot"})
	// This will load from config-FRS.yml
	ybtools.ParseTaskConfig(&yapperconfig.Config)
}

func main() {
	w := ybtools.CreateAndAuthenticateClient(ybtools.DefaultMaxlag)

	rand.Seed(time.Now().UnixNano())

	frslist.Populate()
	rfc.LoadRfcsDone(w)
	defer ybtools.SaveEditLimit()

	ga.FetchGATopics()

	processCategory(w, "Category:Wikipedia requests for comment", true)
	processCategory(w, "Category:Good article nominees", false)
	finishRun(w)
}

// processCategory takes a mwclient instance, a category name, and a bool indicating if the category contains RfCs.
// it then iterates through the pages in the category, checking whether they've already been processed; if they've not,
// and they're applicable, they'll be sent for a feedback request. This is the main program loop.
func processCategory(w *mwclient.Client, category string, rfcCat bool) {
	var startStamp, startID string
	var newRunfile bool
	var parameters params.Values

	if rfcCat {
		// gets a list of all active RfCs. We'll manage which ones to deal with later
		parameters = params.Values{
			"action":    "query",
			"prop":      "revisions",
			"generator": "embeddedin",
			"geititle":  "Template:Rfc",
			"rvprop":    "content",
			"rvslots":   "main",
		}
	} else {
		startStamp, startID = loadFromRunfile(category)
		if startStamp == "" {
			startStamp = time.Now().Format(time.RFC3339)
			// Set our runfile to store this now, as there's potentially going to be nothing in the queue
			newRunfile = true
		}

		parameters = params.Values{
			"action":       "query",
			"prop":         "revisions|categories",
			"generator":    "categorymembers",
			"gcmtitle":     category,
			"gcmsort":      "timestamp",
			"rvprop":       "content",
			"rvslots":      "main",
			"clprop":       "timestamp",
			"clcategories": category,
			"gcmdir":       "descending",
			"gcmstart":     time.Now().Add(-time.Hour).Format(time.RFC3339), // give it at least an hour of tranquility before invites go out
			"gcmend":       startStamp,                                      // this is gcmend not gcmstart as it's going down from the most recent
		}
	}

	var firstItem string
	query := w.NewQuery(parameters)

	for query.Next() {
		pages := ybtools.GetPagesFromQuery(query.Resp())
		// There seems to be no guarantee that the value of pages will be ordered, in any way.
		if len(pages) > 0 {
			if !rfcCat {
				// on the first item of the entire set, and the first item ONLY, save the timestamp and the page id into a var to write to runfile later
				// RfCs don't use this as they're given IDs and don't need it
				if firstItem == "" {
					// because pages is unordered, we have to make a separate, limited request for this
					// this gets us the first (latest) item
					firstItemParams := parameters
					firstItemParams["gcmlimit"] = "1"
					firstItemResp, err := w.Get(firstItemParams)
					if err != nil {
						ybtools.PanicErr("Failed to get firstItemResp with err ", err)
					}

					firstItemRespPages := ybtools.GetPagesFromQuery(firstItemResp)
					if len(firstItemRespPages) != 1 {
						ybtools.PanicErr("firstItemRespPages returned more than one page! Dying.")
					}

					var runfileBuilder strings.Builder
					runfileBuilder.WriteString(ybtools.GetCategorisationTimestampFromPage(firstItemRespPages[0], category))
					runfileBuilder.WriteString(";")

					firstItemPageID, err := firstItemRespPages[0].GetInt64("pageid")
					if err != nil {
						ybtools.PanicErr("Failed to get pageid from the first item in the queue with error message ", err)
					}
					// Remember to do this! Golang by default turns integers just into the
					// corresponding unicode sequence with string(n) - e.g. string(5)
					// returns "\x05"
					runfileBuilder.WriteString(strconv.FormatInt(firstItemPageID, 10))
					firstItem = runfileBuilder.String()
				}
			}

		PAGELOOP:
			for index, page := range pages {
				pageIDInt, err := page.GetInt64("pageid")
				if err != nil {
					ybtools.PanicErr("Failed to get pageid from page in category ", category, " with index ", index, ", error was: ", err)
				}
				pageID := strconv.FormatInt(pageIDInt, 10) // format it into a string integer

				pageTitle, err := page.GetString("title")
				if err != nil {
					log.Println("Failed to get title from page ID", pageID, "so skipping it")
					continue
				}

				pageContent, err := ybtools.GetContentFromPage(page)
				if err != nil {
					log.Println("getContentFromPage failed on page ID", pageID, "so skipping it")
					continue
				}

				if rfcCat {
					// (content, title, excludeDone)
					rfcsToProcess, err := extractRfcs(pageContent, pageTitle, false)
					if err != nil {
						ybtools.PanicErr("extractRfcs errored with ", err)
					}
					rfcsDone := make([]rfc.RfC, 0, len(rfcsToProcess))

				RFCLOOP:
					for _, rfc := range rfcsToProcess {
						if rfc.ID == "" {
							log.Println("RfC has no ID yet on page", pageTitle, "so skipping that RfC")
							continue RFCLOOP
						} else if rfc.FeedbackDone {
							log.Println("RfC feedback already done for an RfC on", pageTitle, "so skipping that RfC")
						} else {
							log.Println("Requesting feedback for an RfC on", pageTitle)
							requestFeedbackFor(rfc, w)
						}
						rfcsDone = append(rfcsDone, rfc)
					}
					if len(rfcsDone) > 0 {
						rfc.MarkRfcsDone(rfcsDone)
					}
				} else {
					// Because each article can only have one GA nomination at a time, it's not necessary to do the full gamut of RfC checks here
					// we can instead just pass it on to requestFeedbackFor after checking that it's not the same page we did first last time
					// to do that check, we check whether the page ID and timestamp are the same (both stored in the runfile) - if they are, it's the same page
					if (pageID == startID) && (ybtools.GetCategorisationTimestampFromPage(page, category) == startStamp) {
						// it's the first page from last time, we're probably at the end - skip over it
						continue PAGELOOP
					} else {
						requestFeedbackFor(extractGANom(pageContent, pageTitle), w)
					}
				}
			}
		} else if newRunfile && len(pages) == 0 {
			// if it's a new file and no pages are picked up, just create the runfile so future runs will know where to start from
			log.Println("No pages found, and a new runfile, so creating runfile with current time for", category)
			firstItem = startStamp + ";"
		}
	}
	if query.Err() == nil {
		log.Println("Finished the queue for category", category, "so ending here")
	} else {
		ybtools.PanicErr("Errored while querying for relevant new pages with error: ", query.Err())
	}

	// If it uses a runfile, and there actually is something to write
	if !rfcCat && len(firstItem) > 0 {
		// Store the done timestamp and page id into the runfile for next use
		err := ioutil.WriteFile(slugify.Marshal(category)+".frsrunfile", []byte(firstItem), 0644)
		if err != nil {
			ybtools.PanicErr("Failed to write timestamp and id to runfile")
		}
	}
}

// finishRun is called at the end of the FRS run, once everything has completed successfully.
// The invocation of finishRun is what starts the message queue processing. This is only a
// separate function really so that we can scope the frslist FinishRun and rfc SaveRfcsDone
// defers into here; this means that if something goes awfully wrong somewhere else in the
// program, we don't end up saving rubbish data after having sent nothing at all, but
// it also means if something goes wrong in the actual sending, the lists are kept up to date.
func finishRun(w *mwclient.Client) {
	defer frslist.FinishRun(w)
	defer rfc.SaveRfcsDone(w)

	// this below line is critical to run, because without it nothing will actually be sent;
	// however, we do NOT want to defer it, because if we do, it would still run on panicks.
	// if something has gone wrong, we don't want to send messages, so we oughtn't run this.
	messages.SendMessageQueue(w)
}
