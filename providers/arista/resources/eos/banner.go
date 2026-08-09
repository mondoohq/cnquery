// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package eos

import (
	"bufio"
	"strings"
)

// Banners holds the login and message-of-the-day banner text.
//
// EOS renders banners in running-config as a header line followed by the
// literal banner body and an `EOF` terminator:
//
//	banner login
//	Authorized users only. All activity is monitored.
//	EOF
//	banner motd
//	   ****************************
//	   *  Restricted system       *
//	   ****************************
//	EOF
//
// The login banner is shown before authentication and is what notice-and-
// consent requirements are checked against; the MOTD banner is shown after.
// An empty string means no banner text is configured.
type Banners struct {
	Login string
	Motd  string
}

// ParseBanners extracts the login and MOTD banner bodies from running-config.
//
// Banner bodies are captured verbatim, including leading whitespace, because
// the indentation is part of the displayed text rather than config nesting.
// That also means the body must be read from the raw lines: the indentation
// heuristics the section parsers use do not apply inside a banner.
func ParseBanners(runningConfig string) *Banners {
	b := &Banners{}

	scanner := bufio.NewScanner(strings.NewReader(runningConfig))
	for scanner.Scan() {
		line := scanner.Text()
		var target *string
		switch strings.TrimSpace(line) {
		case "banner login":
			target = &b.Login
		case "banner motd":
			target = &b.Motd
		case "no banner login":
			b.Login = ""
			continue
		case "no banner motd":
			b.Motd = ""
			continue
		default:
			continue
		}

		// Collect the body verbatim until the EOF terminator. A config that
		// ends mid-banner (truncated capture) simply yields what was read.
		body := []string{}
		for scanner.Scan() {
			bodyLine := scanner.Text()
			if strings.TrimSpace(bodyLine) == "EOF" {
				break
			}
			body = append(body, bodyLine)
		}
		*target = strings.Join(body, "\n")
	}

	return b
}
