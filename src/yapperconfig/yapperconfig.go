package yapperconfig

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

// configObject is the local implementation of ybtools' configuration.
// each of these keys are just pulled straight from the config-frs yml file
// in the application directory by ybtools.
// this doesn't include EditLimit, which is handled by ybtools directly
type configObject struct {
	FRSPageID                string
	SentCountPageID          string
	GAGuidelinesHeaderPageID string
	RFCsDonePageID           string
}

// Config is the global configuration object. This should only really ever be read from.
var Config configObject

// Example config file:
// https://github.com/DavidTornheim/yapperbot-frs/config-frs.testwiki.yml
//   frspageid: 110754 # DO NOT CHANGE THIS PAGEID unless the WP:FRS page is completely different
//   gaguidelinesheaderpageid: 110769 # DO NOT CHANGE THIS PAGEID unless the GA topics are now on a completely different page
//   sentcountpageid: 110772 # DO NOT CHANGE THIS PAGEID
//   rfcsdonepageid: 111355 # DO NOT CHANGE THIS PAGEID
// https://github.com/mashedkeyboard/yapperbot-frs/config-frs.testwiki.yml

