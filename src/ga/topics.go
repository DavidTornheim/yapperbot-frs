package ga

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
	"regexp"
	"yapperbot-frs/src/yapperconfig"

//	"github.com/mashedkeyboard/ybtools/v2"
	"github.com/DavidTornheim/ybtools"
)

// gaTopics is a map storing the Good Article topics in the form {"subtopic": "topic"}
var gaTopics map[string]string

// gaTopicsRegex matches each GA topic from the on-wiki list of GA topics.
var gaTopicsRegex *regexp.Regexp

// gaSubtopicRegex matches each GA subtopic from within a topic on the on-wiki list of GA topics.
var gaSubtopicRegex *regexp.Regexp

func init() {
	gaTopics = map[string]string{
		"Miscellaneous": "Miscellaneous",
	}

	// This regex is used in FetchGATopics to get a list of the subtopics for each topic category
	gaTopicsRegex = regexp.MustCompile(`'''(.*?)'''\s*?<br>\s*?\n((?:\[\[[^|]*\|(?:[^|]*)]](?:{{·}})?\n?)+)`)

	// This regex is used in FetchGATopics to get each subtopic without any trash around it
	gaSubtopicRegex = regexp.MustCompile(`\[\[[^|]*\|([^|]*)]]`)
}

// FetchGATopics fetches the latest GA topics from the Good Article noms page.
func FetchGATopics() {
	text, err := ybtools.FetchWikitext(yapperconfig.Config.GAGuidelinesHeaderPageID)
	if err != nil {
		ybtools.PanicErr("Failed to fetch Good Articles topics with error ", err)
	}
	matches := gaTopicsRegex.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		// match is in the form [full match, topic, all subtopic links]
		subtopics := gaSubtopicRegex.FindAllStringSubmatch(match[2], -1)
		for _, subtopic := range subtopics {
			// subtopic is in the form [full match, subtopic matched]
			gaTopics[subtopic[1]] = match[1]
		}
	}
}
