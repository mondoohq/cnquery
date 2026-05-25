// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// ScheduledTask is the parsed representation of a single Windows Task
// Scheduler XML definition (the kind stored under C:\Windows\System32\Tasks).
type ScheduledTask struct {
	URI         string
	Author      string
	Description string
	Source      string
	Date        *time.Time

	RunAsUser string
	RunLevel  string
	LogonType string
	GroupID   string

	Enabled                    bool
	Hidden                     bool
	AllowStartOnDemand         bool
	RunOnlyIfNetworkAvailable  bool
	StopIfGoingOnBatteries     bool
	DisallowStartIfOnBatteries bool
	MultipleInstancesPolicy    string
	ExecutionTimeLimit         string
	Priority                   int

	Triggers []map[string]any
	Actions  []map[string]any
}

// ParseScheduledTask reads a Task Scheduler XML definition.
//
// Task XML on disk is usually UTF-16 LE with a BOM, occasionally UTF-8 with
// or without a BOM. The Go xml decoder doesn't transcode UTF-16, so detect
// the BOM up front and feed the decoder UTF-8 bytes regardless of what the
// XML prolog declares.
func ParseScheduledTask(r io.Reader) (*ScheduledTask, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "could not read scheduled task xml")
	}

	utf8Bytes, err := normalizeToUTF8(raw)
	if err != nil {
		return nil, err
	}

	dec := xml.NewDecoder(bytes.NewReader(utf8Bytes))
	// The XML prolog may still declare encoding="UTF-16"; since we've already
	// transcoded the bytes, treat any charset as a UTF-8 passthrough.
	dec.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) {
		return input, nil
	}

	var x xmlTask
	if err := dec.Decode(&x); err != nil {
		return nil, errors.Wrap(err, "could not parse scheduled task xml")
	}

	return x.toScheduledTask(), nil
}

func normalizeToUTF8(data []byte) ([]byte, error) {
	switch {
	case len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE:
		dec := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
		out, _, err := transform.Bytes(dec, data[2:])
		if err != nil {
			return nil, errors.Wrap(err, "could not decode utf-16 LE task xml")
		}
		return out, nil
	case len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF:
		dec := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
		out, _, err := transform.Bytes(dec, data[2:])
		if err != nil {
			return nil, errors.Wrap(err, "could not decode utf-16 BE task xml")
		}
		return out, nil
	case len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF:
		return data[3:], nil
	}
	return data, nil
}

// xmlTask mirrors the on-disk Task Scheduler XML schema (namespace
// http://schemas.microsoft.com/windows/2004/02/mit/task). Field tags omit
// the namespace because encoding/xml matches by local name when no namespace
// is declared on the field.
type xmlTask struct {
	XMLName          xml.Name            `xml:"Task"`
	RegistrationInfo xmlRegistrationInfo `xml:"RegistrationInfo"`
	Triggers         xmlTriggers         `xml:"Triggers"`
	Principals       xmlPrincipals       `xml:"Principals"`
	Settings         xmlSettings         `xml:"Settings"`
	Actions          xmlActions          `xml:"Actions"`
}

type xmlRegistrationInfo struct {
	Date        string `xml:"Date"`
	Author      string `xml:"Author"`
	URI         string `xml:"URI"`
	Description string `xml:"Description"`
	Source      string `xml:"Source"`
}

type xmlPrincipals struct {
	Principal []xmlPrincipal `xml:"Principal"`
}

type xmlPrincipal struct {
	ID        string `xml:"id,attr"`
	UserID    string `xml:"UserId"`
	GroupID   string `xml:"GroupId"`
	RunLevel  string `xml:"RunLevel"`
	LogonType string `xml:"LogonType"`
}

type xmlSettings struct {
	Enabled                    *bool  `xml:"Enabled"`
	Hidden                     *bool  `xml:"Hidden"`
	AllowStartOnDemand         *bool  `xml:"AllowStartOnDemand"`
	RunOnlyIfNetworkAvailable  *bool  `xml:"RunOnlyIfNetworkAvailable"`
	StopIfGoingOnBatteries     *bool  `xml:"StopIfGoingOnBatteries"`
	DisallowStartIfOnBatteries *bool  `xml:"DisallowStartIfOnBatteries"`
	MultipleInstancesPolicy    string `xml:"MultipleInstancesPolicy"`
	ExecutionTimeLimit         string `xml:"ExecutionTimeLimit"`
	Priority                   *int   `xml:"Priority"`
}

type xmlTriggers struct {
	BootTrigger               []xmlTrigger         `xml:"BootTrigger"`
	CalendarTrigger           []xmlCalendarTrigger `xml:"CalendarTrigger"`
	EventTrigger              []xmlEventTrigger    `xml:"EventTrigger"`
	IdleTrigger               []xmlTrigger         `xml:"IdleTrigger"`
	LogonTrigger              []xmlLogonTrigger    `xml:"LogonTrigger"`
	RegistrationTrigger       []xmlTrigger         `xml:"RegistrationTrigger"`
	TimeTrigger               []xmlTrigger         `xml:"TimeTrigger"`
	SessionStateChangeTrigger []xmlSessionTrigger  `xml:"SessionStateChangeTrigger"`
}

type xmlTriggerCommon struct {
	ID                 string         `xml:"id,attr"`
	Enabled            *bool          `xml:"Enabled"`
	StartBoundary      string         `xml:"StartBoundary"`
	EndBoundary        string         `xml:"EndBoundary"`
	ExecutionTimeLimit string         `xml:"ExecutionTimeLimit"`
	Repetition         *xmlRepetition `xml:"Repetition"`
}

type xmlTrigger struct {
	xmlTriggerCommon
	Delay string `xml:"Delay"`
}

type xmlCalendarTrigger struct {
	xmlTriggerCommon
	RandomDelay     string             `xml:"RandomDelay"`
	ScheduleByDay   *xmlScheduleByDay  `xml:"ScheduleByDay"`
	ScheduleByWeek  *xmlScheduleByWeek `xml:"ScheduleByWeek"`
	ScheduleByMonth *xmlAnyContent     `xml:"ScheduleByMonth"`
}

type xmlEventTrigger struct {
	xmlTriggerCommon
	Subscription string `xml:"Subscription"`
	Delay        string `xml:"Delay"`
}

type xmlLogonTrigger struct {
	xmlTriggerCommon
	UserID string `xml:"UserId"`
	Delay  string `xml:"Delay"`
}

type xmlSessionTrigger struct {
	xmlTriggerCommon
	UserID      string `xml:"UserId"`
	StateChange string `xml:"StateChange"`
	Delay       string `xml:"Delay"`
}

type xmlScheduleByDay struct {
	DaysInterval int `xml:"DaysInterval"`
}

type xmlScheduleByWeek struct {
	WeeksInterval int            `xml:"WeeksInterval"`
	DaysOfWeek    *xmlAnyContent `xml:"DaysOfWeek"`
}

// xmlAnyContent captures arbitrary nested XML as the raw inner contents.
// Useful for unstructured trigger subtrees we don't model in detail.
type xmlAnyContent struct {
	InnerXML string `xml:",innerxml"`
}

type xmlRepetition struct {
	Interval          string `xml:"Interval"`
	Duration          string `xml:"Duration"`
	StopAtDurationEnd *bool  `xml:"StopAtDurationEnd"`
}

type xmlActions struct {
	Context string `xml:"Context,attr"`

	Exec        []xmlExecAction        `xml:"Exec"`
	ComHandler  []xmlComHandlerAction  `xml:"ComHandler"`
	SendEmail   []xmlSendEmailAction   `xml:"SendEmail"`
	ShowMessage []xmlShowMessageAction `xml:"ShowMessage"`
}

type xmlExecAction struct {
	Command          string `xml:"Command"`
	Arguments        string `xml:"Arguments"`
	WorkingDirectory string `xml:"WorkingDirectory"`
}

type xmlComHandlerAction struct {
	ClassID string `xml:"ClassId"`
	Data    string `xml:"Data"`
}

type xmlSendEmailAction struct {
	From    string `xml:"From"`
	To      string `xml:"To"`
	Subject string `xml:"Subject"`
	Body    string `xml:"Body"`
	Server  string `xml:"Server"`
}

type xmlShowMessageAction struct {
	Title string `xml:"Title"`
	Body  string `xml:"Body"`
}

func (x *xmlTask) toScheduledTask() *ScheduledTask {
	t := &ScheduledTask{
		URI:                     strings.TrimSpace(x.RegistrationInfo.URI),
		Author:                  x.RegistrationInfo.Author,
		Description:             x.RegistrationInfo.Description,
		Source:                  x.RegistrationInfo.Source,
		Date:                    parseTaskTime(x.RegistrationInfo.Date),
		MultipleInstancesPolicy: x.Settings.MultipleInstancesPolicy,
		ExecutionTimeLimit:      x.Settings.ExecutionTimeLimit,
		// Defaults per Task Scheduler schema where the element is absent.
		Enabled:            boolOrDefault(x.Settings.Enabled, true),
		Hidden:             boolOrDefault(x.Settings.Hidden, false),
		AllowStartOnDemand: boolOrDefault(x.Settings.AllowStartOnDemand, true),
		// Battery/network settings default to false except DisallowStartIfOnBatteries
		// and StopIfGoingOnBatteries, both of which default to true.
		RunOnlyIfNetworkAvailable:  boolOrDefault(x.Settings.RunOnlyIfNetworkAvailable, false),
		StopIfGoingOnBatteries:     boolOrDefault(x.Settings.StopIfGoingOnBatteries, true),
		DisallowStartIfOnBatteries: boolOrDefault(x.Settings.DisallowStartIfOnBatteries, true),
		Priority:                   intOrDefault(x.Settings.Priority, 7),
	}

	if len(x.Principals.Principal) > 0 {
		p := x.Principals.Principal[0]
		t.RunAsUser = p.UserID
		t.RunLevel = p.RunLevel
		t.LogonType = p.LogonType
		t.GroupID = p.GroupID
	}

	t.Triggers = collectTriggers(&x.Triggers)
	t.Actions = collectActions(&x.Actions)
	return t
}

func collectTriggers(in *xmlTriggers) []map[string]any {
	var out []map[string]any
	for _, tr := range in.BootTrigger {
		m := triggerCommon("BootTrigger", &tr.xmlTriggerCommon)
		setIfNotEmpty(m, "delay", tr.Delay)
		out = append(out, m)
	}
	for _, tr := range in.CalendarTrigger {
		m := triggerCommon("CalendarTrigger", &tr.xmlTriggerCommon)
		setIfNotEmpty(m, "randomDelay", tr.RandomDelay)
		switch {
		case tr.ScheduleByDay != nil:
			m["scheduleType"] = "Daily"
			m["daysInterval"] = int64(tr.ScheduleByDay.DaysInterval)
		case tr.ScheduleByWeek != nil:
			m["scheduleType"] = "Weekly"
			m["weeksInterval"] = int64(tr.ScheduleByWeek.WeeksInterval)
		case tr.ScheduleByMonth != nil:
			m["scheduleType"] = "Monthly"
		}
		out = append(out, m)
	}
	for _, tr := range in.EventTrigger {
		m := triggerCommon("EventTrigger", &tr.xmlTriggerCommon)
		setIfNotEmpty(m, "subscription", tr.Subscription)
		setIfNotEmpty(m, "delay", tr.Delay)
		out = append(out, m)
	}
	for _, tr := range in.IdleTrigger {
		out = append(out, triggerCommon("IdleTrigger", &tr.xmlTriggerCommon))
	}
	for _, tr := range in.LogonTrigger {
		m := triggerCommon("LogonTrigger", &tr.xmlTriggerCommon)
		setIfNotEmpty(m, "userId", tr.UserID)
		setIfNotEmpty(m, "delay", tr.Delay)
		out = append(out, m)
	}
	for _, tr := range in.RegistrationTrigger {
		m := triggerCommon("RegistrationTrigger", &tr.xmlTriggerCommon)
		setIfNotEmpty(m, "delay", tr.Delay)
		out = append(out, m)
	}
	for _, tr := range in.TimeTrigger {
		m := triggerCommon("TimeTrigger", &tr.xmlTriggerCommon)
		setIfNotEmpty(m, "delay", tr.Delay)
		out = append(out, m)
	}
	for _, tr := range in.SessionStateChangeTrigger {
		m := triggerCommon("SessionStateChangeTrigger", &tr.xmlTriggerCommon)
		setIfNotEmpty(m, "userId", tr.UserID)
		setIfNotEmpty(m, "stateChange", tr.StateChange)
		setIfNotEmpty(m, "delay", tr.Delay)
		out = append(out, m)
	}
	return out
}

func triggerCommon(kind string, c *xmlTriggerCommon) map[string]any {
	m := map[string]any{
		"type":    kind,
		"enabled": boolOrDefault(c.Enabled, true),
	}
	setIfNotEmpty(m, "id", c.ID)
	setIfNotEmpty(m, "startBoundary", c.StartBoundary)
	setIfNotEmpty(m, "endBoundary", c.EndBoundary)
	setIfNotEmpty(m, "executionTimeLimit", c.ExecutionTimeLimit)
	if c.Repetition != nil {
		rep := map[string]any{}
		setIfNotEmpty(rep, "interval", c.Repetition.Interval)
		setIfNotEmpty(rep, "duration", c.Repetition.Duration)
		if c.Repetition.StopAtDurationEnd != nil {
			rep["stopAtDurationEnd"] = *c.Repetition.StopAtDurationEnd
		}
		if len(rep) > 0 {
			m["repetition"] = rep
		}
	}
	return m
}

func collectActions(in *xmlActions) []map[string]any {
	var out []map[string]any
	for _, a := range in.Exec {
		m := map[string]any{"type": "Exec"}
		setIfNotEmpty(m, "command", a.Command)
		setIfNotEmpty(m, "arguments", a.Arguments)
		setIfNotEmpty(m, "workingDirectory", a.WorkingDirectory)
		out = append(out, m)
	}
	for _, a := range in.ComHandler {
		m := map[string]any{"type": "ComHandler"}
		setIfNotEmpty(m, "classId", a.ClassID)
		setIfNotEmpty(m, "data", a.Data)
		out = append(out, m)
	}
	for _, a := range in.SendEmail {
		m := map[string]any{"type": "SendEmail"}
		setIfNotEmpty(m, "from", a.From)
		setIfNotEmpty(m, "to", a.To)
		setIfNotEmpty(m, "subject", a.Subject)
		setIfNotEmpty(m, "body", a.Body)
		setIfNotEmpty(m, "server", a.Server)
		out = append(out, m)
	}
	for _, a := range in.ShowMessage {
		m := map[string]any{"type": "ShowMessage"}
		setIfNotEmpty(m, "title", a.Title)
		setIfNotEmpty(m, "body", a.Body)
		out = append(out, m)
	}
	return out
}

func boolOrDefault(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func intOrDefault(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

func setIfNotEmpty(m map[string]any, key, value string) {
	if value != "" {
		m[key] = value
	}
}

// parseTaskTime accepts the local-time ISO 8601 form Task Scheduler writes
// ("2018-04-12T13:33:55") as well as ones with timezone offsets. Empty or
// unparseable values return nil so the caller can leave the field unset.
func parseTaskTime(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, s); err == nil {
			return &ts
		}
	}
	return nil
}

// TaskPathFromFilePath converts the on-disk task file path (under the Tasks
// root directory) into the Task Scheduler URI form. Example:
//
//	root      = "C:\\Windows\\System32\\Tasks"
//	full path = "C:\\Windows\\System32\\Tasks\\Microsoft\\Windows\\Defrag\\ScheduledDefrag"
//	returns   = "\\Microsoft\\Windows\\Defrag\\ScheduledDefrag"
//
// The Task Scheduler uses backslash-separated paths regardless of the host
// filesystem the file is read from, so both slash styles are normalized.
func TaskPathFromFilePath(root, full string) string {
	norm := func(s string) string { return strings.ReplaceAll(s, "/", "\\") }
	r := strings.TrimRight(norm(root), "\\")
	f := norm(full)
	if !strings.HasPrefix(strings.ToLower(f), strings.ToLower(r)+"\\") {
		return ""
	}
	rel := f[len(r):]
	rel = strings.TrimRight(rel, "\\")
	if rel == "" {
		return ""
	}
	return rel
}

// TaskLeafName returns the leaf task name from a Task Scheduler path.
func TaskLeafName(taskPath string) string {
	if taskPath == "" {
		return ""
	}
	parts := strings.Split(taskPath, "\\")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	return ""
}
