package cvss

var msgMapping = map[Severity]string{
	None:     "no",
	Low:      "low",
	Medium:   "medium",
	High:     "high",
	Critical: "critical",
	Unknown:  "unknown",
}

func (severity Severity) RatingName() string {
	msg, ok := msgMapping[severity]
	if ok {
		return msg
	} else {
		return msgMapping[Unknown]
	}
}
