package core

import "strconv"

func (c *mqlAuditCvss) id() (string, error) {
	score, _ := c.Score()
	vector, _ := c.Vector()
	return "cvss/" + strconv.FormatFloat(score, 'f', 2, 64) + "/vector/" + vector, nil
}

func (c *mqlAuditAdvisory) id() (string, error) {
	return c.Mrn()
}

func (c *mqlAuditCve) id() (string, error) {
	return c.Mrn()
}

func (c *mqlAuditExploit) id() (string, error) {
	return c.Mrn()
}
