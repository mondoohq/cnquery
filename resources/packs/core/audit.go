package core

import "strconv"

func (c *lumiAuditCvss) id() (string, error) {
	score, _ := c.Score()
	vector, _ := c.Vector()
	return "cvss/" + strconv.FormatFloat(score, 'f', 2, 64) + "/vector/" + vector, nil
}

func (c *lumiAuditAdvisory) id() (string, error) {
	return c.Mrn()
}

func (c *lumiAuditCve) id() (string, error) {
	return c.Mrn()
}

func (c *lumiAuditExploit) id() (string, error) {
	return c.Mrn()
}
