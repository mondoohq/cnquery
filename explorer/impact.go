package explorer

func (i *Impact) Rating() string {
	if i == nil {
		return "unknown"
	}
	switch {
	case i.Value == 0:
		return "none"
	case i.Value > 0 && i.Value < 40:
		return "low"
	case i.Value >= 40 && i.Value < 70:
		return "medium"
	case i.Value >= 70 && i.Value < 90:
		return "high"
	case i.Value >= 90:
		return "critical"
	}
	return "unknown"
}
