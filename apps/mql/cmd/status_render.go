// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/muesli/termenv"
	"go.mondoo.com/mql/v13/cli/theme"
	"go.mondoo.com/mql/v13/cli/theme/colors"
)

// RenderOptions controls how RenderCli formats the status screen. Color is
// disabled for non-TTY output and in tests so the rendered string is
// deterministic plain text.
type RenderOptions struct {
	Color bool
	Width int
}

// styler applies (or, when disabled, passes through) terminal styling. When
// Color is off it returns the input verbatim, guaranteeing escape-free output
// that tests and pipes can assert on. The palette is captured at construction
// so the semantic helpers don't reach into a global mid-render.
type styler struct {
	on      bool
	profile termenv.Profile
	palette colors.Theme
}

// defaultRenderColor reports whether the status screen should be rendered with
// color, based on the detected terminal profile (which already honors NO_COLOR
// and non-TTY output).
func defaultRenderColor() bool {
	return colors.Profile != termenv.Ascii
}

func newStyler(color bool) styler {
	p := colors.Profile
	if !color {
		p = termenv.Ascii
	}
	return styler{on: color, profile: p, palette: theme.DefaultTheme.Colors}
}

func (st styler) fg(c termenv.Color, s string) string {
	if !st.on {
		return s
	}
	return st.profile.String(s).Foreground(c).String()
}

func (st styler) bold(s string) string {
	if !st.on {
		return s
	}
	return st.profile.String(s).Bold().String()
}

// semantic roles mapped onto the theme palette
func (st styler) value(s string) string  { return st.bold(s) }
func (st styler) dim(s string) string    { return st.fg(st.palette.Disabled, s) }
func (st styler) ok(s string) string     { return st.fg(st.palette.Success, s) }
func (st styler) bad(s string) string    { return st.fg(st.palette.Error, s) }
func (st styler) warn(s string) string   { return st.fg(st.palette.Warn, s) }
func (st styler) cmd(s string) string    { return st.fg(st.palette.Command, s) }
func (st styler) accent(s string) string { return st.fg(st.palette.Secondary, s) }

// RenderCli renders the human-facing status screen as a string. It performs no
// I/O and reads no globals: everything it needs is on the Status value, which
// makes it directly unit-testable.
func (s Status) RenderCli(opts RenderOptions) string {
	st := newStyler(opts.Color)
	var b strings.Builder

	s.renderHeader(&b, st)
	s.renderHealth(&b, st)
	s.renderSystem(&b, st)
	s.renderMql(&b, st)
	s.renderPlatform(&b, st)
	s.renderProviders(&b, st)
	s.renderFooter(&b, st)

	// trim trailing whitespace from every line so framed blank rows ("│") and
	// short rows don't leave ragged trailing spaces
	lines := strings.Split(b.String(), "\n")
	for i, ln := range lines {
		lines[i] = strings.TrimRight(ln, " ")
	}
	return strings.Join(lines, "\n")
}

const ruleWidth = 46

// section writes an open-frame header: "── Title ─────────". There is no right
// border, which sidesteps glyph-width alignment problems entirely.
func (st styler) section(b *strings.Builder, title string) {
	fmt.Fprintf(b, "\n  %s %s %s\n", st.dim("──"), st.value(title), st.dim(strings.Repeat("─", ruleWidth)))
}

// row writes a left-ruled body line: "│  label  value".
func (st styler) row(b *strings.Builder, label, value string) {
	fmt.Fprintf(b, "  %s  %s  %s\n", st.dim("│"), st.dim(padRight(label, 9)), value)
}

// rowRaw writes a left-ruled body line without a label column, for tables and
// free-form rows under a section.
func (st styler) rowRaw(b *strings.Builder, value string) {
	fmt.Fprintf(b, "  %s  %s\n", st.dim("│"), value)
}

func (s Status) renderSystem(b *strings.Builder, st styler) {
	st.section(b, "System")

	host := st.value(s.Client.Hostname)
	if s.Client.IP != "" {
		host += " " + st.dim("· "+s.Client.IP)
	}
	st.row(b, "Host", host)

	plat := st.dim("unknown")
	if s.Client.Platform != nil {
		plat = s.Client.Platform.Name + " " + st.value(s.Client.Platform.Version) + " " + st.dim("· "+s.Client.Platform.Arch)
	}
	st.row(b, "Platform", plat)
}

func (s Status) renderMql(b *strings.Builder, st styler) {
	st.section(b, "mql")

	ver := st.value(s.Client.Version)
	if s.updateAvailable() {
		ver = st.dim(s.Client.Version) + "  " + st.accent("→") + "  " + st.warn(s.Client.LatestVersion) +
			"   " + st.warn("⚠ update recommended")
	}
	st.row(b, "Version", ver)
	st.row(b, "API", "v"+s.Client.APIVersion)

	if s.Client.ConfigFile != "" {
		st.row(b, "Config", st.value(s.Client.ConfigFile))
	} else {
		st.row(b, "Config", st.dim("defaults — no config file loaded"))
	}
	st.row(b, "Updates", s.Client.UpdatesURL)
}

func (s Status) renderPlatform(b *strings.Builder, st styler) {
	st.section(b, "Mondoo Platform")
	st.row(b, "Endpoint", s.Upstream.API.Endpoint)

	statusStr := st.bad(s.Upstream.API.Status)
	if s.Upstream.API.Status == "SERVING" {
		statusStr = st.ok("SERVING")
	}
	sync := st.ok("✓ in sync")
	if s.Upstream.API.Version != s.Client.APIVersion {
		// A client running a newer API version than the server is expected
		// during a staged rollout (the CLI updates ahead of the platform) and
		// is not a problem, so only flag a mismatch when the client trails the
		// server (or the versions can't be compared numerically).
		if clientAPINewer(s.Client.APIVersion, s.Upstream.API.Version) {
			sync = st.ok("✓ client ahead")
		} else {
			sync = st.warn("⚠ version mismatch")
		}
	}
	st.row(b, "Status", statusStr+"   "+st.dim("API v"+s.Upstream.API.Version)+" "+sync)
	st.row(b, "Time", s.Upstream.API.Timestamp)

	if s.Client.Registered {
		st.row(b, "Client", st.value(s.Client.Mrn))
		st.row(b, "Account", st.value(s.Client.ServiceAccount))
	} else {
		st.row(b, "Client", st.bad("✗ not registered")+" "+st.dim("—")+" run "+st.cmd("mql login"))
	}

	if len(s.Upstream.Features) > 0 {
		st.row(b, "Features", st.dim(strings.Join(s.Upstream.Features, ", ")))
	}

	// surface upstream warnings (clock skew, slow round-trip, etc.) as their own
	// highlighted lines so they aren't silently dropped
	for _, w := range s.Upstream.Warnings {
		st.rowRaw(b, st.warn("⚠ "+w))
	}
}

func (s Status) renderHeader(b *strings.Builder, st styler) {
	meta := fmt.Sprintf("%s · %s · %s", s.Client.Version, s.osArch(), s.Client.Timestamp)
	fmt.Fprintf(b, "\n  %s    %s\n", st.bold("mql status"), st.dim(meta))
}

// osArch returns "<os>/<arch>" for the header, or "unknown" when platform info
// could not be determined.
func (s Status) osArch() string {
	if s.Client.Platform == nil {
		return "unknown"
	}
	return s.Client.Platform.Name + "/" + s.Client.Platform.Arch
}

// renderHealth renders the at-a-glance summary: one dot per dimension (Client,
// Platform, Updates) colored by state.
func (s Status) renderHealth(b *strings.Builder, st styler) {
	b.WriteString("\n")

	switch {
	case !s.Client.Registered:
		st.summaryRow(b, st.bad("●"), "Client", st.bad("not registered"), st.bad("✗ action needed"))
	case s.Client.PingPongError != nil:
		st.summaryRow(b, st.bad("●"), "Client", st.value("registered"), st.bad("✗ authentication failed"))
	default:
		st.summaryRow(b, st.ok("●"), "Client", st.value("registered"), st.ok("✓ authenticated"))
	}

	if s.platformHealthy() {
		body := "reachable " + st.dim("·") + " " + st.value("SERVING")
		st.summaryRow(b, st.ok("●"), "Platform", body, st.ok("✓ healthy"))
	} else {
		status := s.Upstream.API.Status
		if status == "" {
			status = "unreachable"
		}
		st.summaryRow(b, st.bad("●"), "Platform", st.bad(status), st.bad("✗ unreachable"))
	}

	var parts []string
	if s.updateAvailable() {
		parts = append(parts, "mql "+st.dim(s.Client.Version)+" "+st.accent("→")+" "+st.warn(s.Client.LatestVersion))
	}
	if n := s.outdatedCount(); n > 0 {
		parts = append(parts, st.warn(fmt.Sprintf("%d providers outdated", n)))
	}
	if len(parts) == 0 {
		st.summaryRow(b, st.ok("●"), "Updates", st.ok("up to date"), "")
	} else {
		st.summaryRow(b, st.warn("●"), "Updates", strings.Join(parts, " "+st.dim("·")+" "), "")
	}
}

func (st styler) summaryRow(b *strings.Builder, dot, label, body, trailing string) {
	fmt.Fprintf(b, "  %s %s  %s", dot, st.value(padRight(label, 9)), body)
	if trailing != "" {
		fmt.Fprintf(b, "   %s", trailing)
	}
	b.WriteString("\n")
}

func (s Status) platformHealthy() bool { return s.Upstream.API.Status == "SERVING" }

// clientAPINewer reports whether the client's API version is strictly newer
// than the server's. Both are major-version strings (e.g. "13"); if either
// isn't a plain integer (a development "unstable" build, say), we can't prove
// the client is ahead and return false so the caller falls back to warning.
func clientAPINewer(clientVersion, serverVersion string) bool {
	client, err := strconv.Atoi(clientVersion)
	if err != nil {
		return false
	}
	server, err := strconv.Atoi(serverVersion)
	if err != nil {
		return false
	}
	return client > server
}

// updateAvailable reports whether a newer mql release exists. Development
// builds ("unstable") never count as outdated.
func (s Status) updateAvailable() bool {
	return s.Client.LatestVersion != "" &&
		s.Client.Version != s.Client.LatestVersion &&
		s.Client.Version != "unstable"
}

func (s Status) outdatedCount() int {
	n := 0
	for _, p := range s.Client.Providers {
		if p.Outdated {
			n++
		}
	}
	return n
}

// registryReachable reports whether provider version data was retrieved. When
// the registry is unreachable every provider's Latest is empty, so version
// drift cannot be shown.
func (s Status) registryReachable() bool {
	if len(s.Client.Providers) == 0 {
		return true
	}
	for _, p := range s.Client.Providers {
		if p.Latest != "" {
			return true
		}
	}
	return false
}

// providerTableLimit caps how many outdated providers the drift table lists
// before collapsing the rest into a "+N more" line.
const providerTableLimit = 8

func (s Status) renderProviders(b *strings.Builder, st styler) {
	st.section(b, "Providers")

	total := len(s.Client.Providers)
	outdated := s.outdatedCount()
	current := total - outdated

	if !s.registryReachable() {
		st.rowRaw(b, st.value(strconv.Itoa(total))+" "+st.dim("installed")+"   "+
			st.warn("registry unreachable — provider versions unknown"))
		return
	}

	st.rowRaw(b, st.value(strconv.Itoa(total))+" "+st.dim("installed")+" "+st.dim("·")+" "+
		st.warn(strconv.Itoa(outdated)+" outdated")+" "+st.dim("·")+" "+
		st.ok(strconv.Itoa(current)+" current"))

	if outdated > 0 {
		st.rowRaw(b, "")
		st.rowRaw(b, st.dim(padRight("PROVIDER", 18)+padRight("INSTALLED", 11)+"LATEST"))

		shown := 0
		for _, p := range s.Client.Providers {
			if !p.Outdated {
				continue
			}
			if shown >= providerTableLimit {
				break
			}
			st.rowRaw(b, st.value(padRight(p.Name, 18))+st.dim(padRight(p.Installed, 9))+
				st.accent("→")+"  "+st.warn(p.Latest))
			shown++
		}
		if outdated > providerTableLimit {
			st.rowRaw(b, st.dim(fmt.Sprintf("+ %d more outdated", outdated-providerTableLimit))+" "+
				st.dim("·")+" "+st.cmd("mql providers list")+st.dim(" to see all"))
		}
	}

	currentNames := make([]string, 0, current)
	for _, p := range s.Client.Providers {
		if !p.Outdated {
			currentNames = append(currentNames, p.Name)
		}
	}
	if len(currentNames) > 0 {
		st.rowRaw(b, "")
		extra := 0
		const currentLimit = 12
		if len(currentNames) > currentLimit {
			extra = len(currentNames) - currentLimit
			currentNames = currentNames[:currentLimit]
		}
		line := st.ok("✓ current") + "  " + st.dim(strings.Join(currentNames, ", "))
		if extra > 0 {
			line += st.dim(fmt.Sprintf(", +%d", extra))
		}
		st.rowRaw(b, line)
	}

	if outdated > 0 {
		st.rowRaw(b, st.dim("providers refresh automatically on next run")+" "+st.dim("·")+" "+
			st.cmd("mql providers install <name>")+st.dim(" to update one now"))
	}
}

func (s Status) renderFooter(b *strings.Builder, st styler) {
	// authFailed is the "registered but the platform rejected the credential"
	// case (revoked or expired service account) — distinct from never having
	// registered at all, and remediated by re-registering with a fresh token.
	authFailed := s.Client.Registered && s.Client.PingPongError != nil
	hasError := !s.Client.Registered || s.Client.PingPongError != nil
	actionable := hasError || s.updateAvailable() || s.outdatedCount() > 0

	b.WriteString("\n")
	switch {
	case !s.Client.Registered:
		fmt.Fprintf(b, "  %s %s\n", st.bad("✗ not registered"), st.dim("· exit 1 · next steps:"))
	case authFailed:
		fmt.Fprintf(b, "  %s %s\n", st.bad("✗ authentication failed"), st.dim("· exit 1 · next steps:"))
	case !actionable:
		fmt.Fprintf(b, "  %s\n\n", st.ok("✓ all systems healthy"))
		return
	default:
		fmt.Fprintf(b, "  %s\n", st.dim("next steps:"))
	}

	if !s.Client.Registered {
		st.footerStep(b, "mql login", "register this client with Mondoo Platform")
	}
	if authFailed {
		st.footerStep(b, "mql login --token <token>", "re-register with a fresh token from Mondoo Platform")
	}
	if s.updateAvailable() {
		st.footerStep(b, "visit "+s.Client.UpdatesURL, "upgrade "+s.Client.Version+" → "+s.Client.LatestVersion)
	}
	if s.outdatedCount() > 0 {
		st.footerStep(b, "mql providers install <name>", fmt.Sprintf("update %d outdated providers", s.outdatedCount()))
	}
	b.WriteString("\n")
}

func (st styler) footerStep(b *strings.Builder, command, desc string) {
	fmt.Fprintf(b, "    %s %s    %s\n", st.cmd("→"), st.cmd(padRight(command, 28)), st.dim(desc))
}

// padRight pads s with spaces to a width of n display columns. It counts runes
// rather than bytes so multi-byte content still aligns.
func padRight(s string, n int) string {
	width := utf8.RuneCountInString(s)
	if width >= n {
		return s
	}
	return s + strings.Repeat(" ", n-width)
}
